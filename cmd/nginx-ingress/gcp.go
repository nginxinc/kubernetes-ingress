//go:build gcp

package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"text/template"
	"time"

	ubb "github.com/GoogleCloudPlatform/ubbagent/sdk"
	"github.com/golang/glog"
	"github.com/nginxinc/kubernetes-ingress/internal/k8s"
	api_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	// The name of the environment variable that will contain the GCP reporting
	// secret name; it must be in the format <namespace>/<name>.
	gcpReportingSecretEnvironmentKey = "GCP_REPORTING_SECRET_NAME" //#nosec G101
	// The maximum number of (possibly transient) reporting errors to permit
	// before terminating processes because usage has not been reported.
	allowedUBBReportingAgentFailureCount = 5
	// The number of seconds between sending usage reports to the GCP
	// service endpoint.
	ubbReportingIntervalSeconds = 60
	// The duration between checking usage reporting agent status.
	ubbReportingAgentStatusCheckTick = 30 * time.Second
	// The UBB Agent configuration template
	ubbConfigurationTemplate = `---
identities:
  - name: gcp
    gcp:
      encodedServiceAccountKey: {{ .ReportingKey }}
metrics:
  - name: cpu_usage_pod_hour
    type: int
    passthrough: {}
    endpoints:
      - name: servicecontrol
endpoints:
  - name: servicecontrol
    servicecontrol:
      identity: gcp
      serviceName: {{ .ServiceName }}
      consumerId: {{ .ConsumerID }}
sources:
  - name: cpu_usage_pod_hour_heartbeat
    heartbeat:
      metric: cpu_usage_pod_hour
      intervalSeconds: {{ .IntervalSeconds }}
      value:
        int64Value: {{ .IntervalSeconds }}
      labels:
        auto: true
`
)

var (
	errReportingSecretRequired = errors.New("gcp marketplace requires " + gcpReportingSecretEnvironmentKey + " environment variable to be set")
	errNGINXPlusRequired       = errors.New("gcp marketplace requires nginx-plus flag")
	errTooManyMeterFailures    = errors.New("too many failures sending usage metrics to GCP")
	errSecretTypeMismatch      = errors.New("usage Secret must be of the type Opaque")
	errSecretDataMissingField  = errors.New("usage Secret is missing a required data field")
	// This value will be populated during build and must match the fully-qualified
	// service name associated with the product in GCP Marketplace.
	productCode string
)

func init() {
	startupCheckFn = checkGCPMetering
}

// This function will verify that the kubernetes deployment has the usage reporting
// secret and necessary command line flags set. If all validations pass, the
// startMeteringFn and stopMeteringFn vars will be updated for GCP usage reporting.
func checkGCPMetering() error {
	gcpReportingSecret := os.Getenv(gcpReportingSecretEnvironmentKey)
	if gcpReportingSecret == "" {
		return errReportingSecretRequired
	}
	if !*nginxPlus {
		return errNGINXPlusRequired
	}
	_, client := createConfigAndKubeClient()
	secret, err := getAndValidateUsageSecret(client, gcpReportingSecret)
	if err != nil {
		return err
	}

	config, err := buildUBBAgentConfig(string(secret.Data["consumer-id"]), string(secret.Data["reporting-key"]), ubbReportingIntervalSeconds)
	if err != nil {
		return err
	}

	shutdown := make(chan struct{}, 1)
	startMeteringFn = newStartUBBAgentMetering(config, shutdown)
	stopMeteringFn = func() {
		shutdown <- struct{}{}
	}
	return nil
}

// Retrieves the named secret from Kubernetes and validates that the payload
// meets the requirements for usage based reporting.
func getAndValidateUsageSecret(client kubernetes.Interface, secretNsName string) (secret *api_v1.Secret, err error) {
	ns, name, err := k8s.ParseNamespaceName(secretNsName)
	if err != nil {
		return nil, fmt.Errorf("could not parse the %v argument: %w", secretNsName, err)
	}
	secret, err = client.CoreV1().Secrets(ns).Get(context.TODO(), name, meta_v1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("could not get %v: %w", secretNsName, err)
	}
	if secret.Type != api_v1.SecretTypeOpaque {
		return nil, errSecretTypeMismatch
	}
	if secret.Data["consumer-id"] == nil || len(secret.Data["consumer-id"]) == 0 {
		return nil, fmt.Errorf("consumer-id field is required: %w", errSecretDataMissingField)
	}
	if secret.Data["reporting-key"] == nil || len(secret.Data["reporting-key"]) == 0 {
		return nil, fmt.Errorf("reporting-key field is required: %w", errSecretDataMissingField)
	}
	return secret, nil
}

// Generate a UBBAgent configuration from the embedded template and parameters.
func buildUBBAgentConfig(consumerID, reportingKey string, intervalSeconds int) ([]byte, error) {
	substitutions := struct {
		ServiceName     string
		ReportingKey    string
		ConsumerID      string
		IntervalSeconds int
	}{
		ServiceName:     productCode,
		ReportingKey:    reportingKey,
		ConsumerID:      consumerID,
		IntervalSeconds: intervalSeconds,
	}
	var agentConfig bytes.Buffer
	agentConfigTemplate, err := template.New("agentConfig").Parse(ubbConfigurationTemplate)
	if err != nil {
		return nil, err
	}
	if err := agentConfigTemplate.Execute(&agentConfig, substitutions); err != nil {
		return nil, err
	}
	return agentConfig.Bytes(), nil
}

// Set the startMeteringFn to create a UBB agent and periodically check
// that the usage metrics are reaching GCP.
func newStartUBBAgentMetering(config []byte, shutdown <-chan struct{}) func(done chan<- error) {
	return func(done chan<- error) {
		// Creating the agent starts it too
		agent, err := ubb.NewAgent(config, "")
		if err != nil {
			done <- err
			return
		}

		// Periodically verify that UBB agent is reporing metrics, sending
		// an error to the channel if the duration since last successful
		// receipt of metrics is too long. Function will not exit until
		// something is received from the shutdown channel.
		go func() {
			ticker := time.NewTicker(ubbReportingAgentStatusCheckTick)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					status := agent.GetStatus()
					if status.CurrentFailureCount > allowedUBBReportingAgentFailureCount {
						glog.Infof("GCP UBBAgent has reported failure too many times: %+v", status)
						done <- errTooManyMeterFailures
					}

				case <-shutdown:
					glog.Info("Received shutdown, terminating GCP usage agent")
					done <- agent.Shutdown()
					return
				}
			}
		}()
	}
}
