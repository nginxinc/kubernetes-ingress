//go:build gcp

package main

import (
	"context"
	"errors"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	testNamespace = "test-namespace"
	// See https://storage.cloud.google.com/cloud-marketplace-tools/reporting_secrets/fake_reporting_secret.yaml
	fakeConsumerID    = "project:pr-xxxx-fake-xxxx"
	fakeEntitlementID = "ffffffff-ffff-ffff-ffff-ffffffffffff"
	fakeReportingKey  = "ewogICJ0eXBlIjogInNlcnZpY2VfYWNjb3VudCIsCiAgInByb2plY3RfaWQiOiAiY2xvdWQtbWFya2V0cGxhY2UtdG9vbHMiLAogICJwcml2YXRlX2tleV9pZCI6ICJmNGZiMGQ2MzNhZDQ3YjEwZTJhNDRjM2ZjMGZiYjA3NTk4NzgyY2JjIiwKICAicHJpdmF0ZV9rZXkiOiAiLS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tXG5NSUlFdkFJQkFEQU5CZ2txaGtpRzl3MEJBUUVGQUFTQ0JLWXdnZ1NpQWdFQUFvSUJBUUN6MG9iVmxtVjA4MFVoXG5uY3h0d2dyTU1JNmd0K2NhNlRqdk4vbk5naDFudk1UMXR0NnptYjZCMGI0Sk1vWExHN2pjL3F5OWJLa0tDOFA2XG5lWDJjWnA1ZEhOUlBFUExUYzFWTnBWY3BPNmQxUXRsVm9NYWUzVmMyMlF3MzBCalNHS1MvWE9Wb2RsTytQLzY3XG5xM1FaT3pZbjE4MjA2OFd3L0VlT3E3YmFvT3BieVRDRVQ2RTcrOG8raU1ET1JIZW5rMTRrMkZ6R24zK2NqOUZIXG5TNHFCL1dSKzQxZm5lNW91NU5JSXE0aWVYVWdlNzZ6OFYxZDN1dWF5cFhhY0NtTCtMZFVyYXY1b0RIN3lESUJ6XG5MaGFGS29lSGJ6RFpTdGdSaUJpVjZFbzZhTWxuN0NwTGxqbVNPdGorbVNSbHYzdEU2blU5N1ZqOEptaWNCQ3pWXG5JdDluNHRzSEFnTUJBQUVDZ2dFQUJSbnFwNklweEZJUnV0STlBT1laZVZFWC9XT0tMbFRCditPTGRoMkQvbDZOXG5ZSUpzdWVxWmRzUlU5WnpWelNlL0FhSHdrNC9DdklnTjZndjFRM0VUMEFTOUx5QmJkYzByOVNwdmJtVGFBTXBzXG5oa1hXZ2ZPUExGS205VDhLM1RidXdZWlRTYmlGblZHTkdwVFN2bjUrc0UycWNSWDVMZWFubEZxM0NCSmxqa1NsXG5aY08rUnNEUjRQNDBFUWUwaWQxazNTYW9JMHRBS0taNGNCbGRhS1JBVFRXNHBzNkdvSG9yOVI2NDBYY1ZRSzlWXG5lTlJPamMwTmltRnlKUWZUS2NXaUxIMVhiQTRUQ29GV0hOdUNlL2hWUVRSVnJSOXVoMXVUSnB5UG40QkMzZnpSXG5NdHFyNXlhTG5ybTMyY0Qrb0YrVnZYN1JlVmtMZnZMSk5DVURTOEF1cVFLQmdRRDA4N2RHUFJhTWJGOUV5bmVqXG45S3RqSlNBclhzYmJwblJQRzlSa0FUSTlGaXBtTFpaczRTaDc5OWdYNVdPNXRVNldBYVVqTUR1M3pzTGZCNCtPXG5NNE9MQml0L3kwVzFBZlM0TUY2TmpmcElaSUlsOWk1Z0JLUnBFQXB0dmVtSW9GZFlRY0xTb2F0SFJlcHBlN2liXG5iMHFnYi9UVnNFYXhhZ1B2KzhIN3pEcUZXUUtCZ1FDNzdzNUg0UjN1WDRUS3JROWU0SVFNNm9qUEIxQk9EWG1jXG5reTJCUG5LclNBb1cycllZRXpnNXpFQmpyendsOU1nK0lhQXVubS9ydENFZ0o2S2ZKVXdYQ3hWcTZjV21yVHhCXG5hQ2ZHTEozai85alhNQ2tHT2tUYUhUdjllcHUwbC90ZzM5NkxCSUxMaWhnOW96UVozbHlrazdHOXhYcmFEbW5nXG5KTEQ5a3NkM1h3S0JnRWNxL0NmREhlY0VvWlZhQWZLMzVvZXl4S3IxS1crdDZBTUlBZWhnVkpsYzlFcWxtaHZlXG5PeVh4ZDI1UjdteUpXZURKYjVKT3REc09McDRnRXp4c2lSNStWMnNVd3hiNUQ0SG9ROEI2N0tuVjBkNTNyVGVtXG5nYUlveis3Y2k1cHZnNUVYNGlQU1p2SVpSU2NLbEROTTNYREp0bWZUaEdhTmQ4Rms4eEpXWHZaWkFvR0FZUE04XG5SWWFUMjFJNWZoa3pVYjIvUWE2SWIwMFZsMzZLRzBVdDkzdlF5aDI2M3JscnNSWFJMcmY1QzdQdDhxTEozb3VZXG5TQlNDSm5WaGxXWDlGZDYyMXpobmp5VVVTdjBabGFCMnpGeGVBNjRNSGs4QkN1NXFjSjhlUUpETTNLaC9EU1hRXG5kNlVYR0l1Z0g4UWU3NjF2MjVNNTRXMk1DQXZoZ0xsTStUT01aVDhDZ1lCcjV5R1piMlBGZitKbWZmTU1ENE56XG5lbEp1S1JlYzRielpsNFBGQ1c3OHpJMi9DZTZjOHRnWGN0OEptTUM2R2kxaERCMS8vZlZISjcxRnlSYjBIYXVPXG5TWHZ2VXQ1OGEvL1Bsc1ZUbGNucGJ2UEVRRTY0SjZ6UjlnK0RLNWhEOGo4UUxVaFhkdHFYYjVhUXA4QzlzNG5PXG5icURQZ3VoL0tJaVA1WSthUE9laG53PT1cbi0tLS0tRU5EIFBSSVZBVEUgS0VZLS0tLS1cbiIsCiAgImNsaWVudF9lbWFpbCI6ICJ4eHgtZmFrZS1yZXBvcnRlci14eHhAY2xvdWQtbWFya2V0cGxhY2UtdG9vbHMuaWFtLmdzZXJ2aWNlYWNjb3VudC5jb20iLAogICJjbGllbnRfaWQiOiAiMTA1ODYyMDM3ODQ1Mjk5NzI3ODIzIiwKICAiYXV0aF91cmkiOiAiaHR0cHM6Ly9hY2NvdW50cy5nb29nbGUuY29tL28vb2F1dGgyL2F1dGgiLAogICJ0b2tlbl91cmkiOiAiaHR0cHM6Ly9vYXV0aDIuZ29vZ2xlYXBpcy5jb20vdG9rZW4iLAogICJhdXRoX3Byb3ZpZGVyX3g1MDlfY2VydF91cmwiOiAiaHR0cHM6Ly93d3cuZ29vZ2xlYXBpcy5jb20vb2F1dGgyL3YxL2NlcnRzIiwKICAiY2xpZW50X3g1MDlfY2VydF91cmwiOiAiaHR0cHM6Ly93d3cuZ29vZ2xlYXBpcy5jb20vcm9ib3QvdjEvbWV0YWRhdGEveDUwOS94eHgtZmFrZS1yZXBvcnRlci14eHglNDBjbG91ZC1tYXJrZXRwbGFjZS10b29scy5pYW0uZ3NlcnZpY2VhY2NvdW50LmNvbSIKfQ=="
)

func init() {
	productCode = "test-service"
}

// Create a new Secret that contains data fields populated with values that are
// expected to pass validation, but will be rejected by GCP Service API if/when
// an attempt is made to record the metrics for billing purposes.
func newFakeReportingSecret(t *testing.T, name, namespace string) *v1.Secret {
	t.Helper()
	return &v1.Secret{
		TypeMeta: meta_v1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		Type: v1.SecretTypeOpaque,
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"consumer-id":    []byte(fakeConsumerID),
			"entitlement-id": []byte(fakeEntitlementID),
			"reporting-key":  []byte(fakeReportingKey),
		},
	}
}

// Testing kubernetes client that has access to the supplied secret.
func newTestClientWithSecret(t *testing.T, secret *v1.Secret) (*fake.Clientset, error) {
	t.Helper()
	client := fake.NewSimpleClientset()
	ns := secret.ObjectMeta.Namespace
	if ns == "" {
		ns = testNamespace
	}
	if _, err := client.CoreV1().Secrets(ns).Create(context.TODO(), secret, meta_v1.CreateOptions{}); err != nil {
		return nil, err
	}
	return client, nil
}

// Verify expected behavior when the marketplace reporting secret has missing
// fields, incorrect type, etc.
func TestGetAndValidateUsageSecret(t *testing.T) {
	tests := []struct {
		name          string
		fullName      string
		secret        *v1.Secret
		expectedError error
	}{
		{
			name:     "google-fake-secret",
			fullName: testNamespace + "/" + "google-fake-secret",
			secret:   newFakeReportingSecret(t, "google-fake-secret", testNamespace),
		},
		{
			name:     "empty-secret",
			fullName: testNamespace + "/" + "empty-secret",
			secret: &v1.Secret{
				TypeMeta: meta_v1.TypeMeta{
					Kind:       "Secret",
					APIVersion: "v1",
				},
				Type: v1.SecretTypeOpaque,
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "empty-secret",
					Namespace: testNamespace,
				},
				Data: map[string][]byte{},
			},
			expectedError: errSecretDataMissingField,
		},
		{
			name:     "missing-consumer-id-secret",
			fullName: testNamespace + "/" + "missing-consumer-id-secret",
			secret: &v1.Secret{
				TypeMeta: meta_v1.TypeMeta{
					Kind:       "Secret",
					APIVersion: "v1",
				},
				Type: v1.SecretTypeOpaque,
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "missing-consumer-id-secret",
					Namespace: testNamespace,
				},
				Data: map[string][]byte{
					"reporting-key": []byte("test-key"),
				},
			},
			expectedError: errSecretDataMissingField,
		},
		{
			name:     "missing-reporting-key-secret",
			fullName: testNamespace + "/" + "missing-reporting-key-secret",
			secret: &v1.Secret{
				TypeMeta: meta_v1.TypeMeta{
					Kind:       "Secret",
					APIVersion: "v1",
				},
				Type: v1.SecretTypeOpaque,
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "missing-reporting-key-secret",
					Namespace: testNamespace,
				},
				Data: map[string][]byte{
					"consumer-id": []byte("test-id"),
				},
			},
			expectedError: errSecretDataMissingField,
		},
		{
			name:     "incorrect-secret-type",
			fullName: testNamespace + "/" + "incorrect-secret-type",
			secret: &v1.Secret{
				TypeMeta: meta_v1.TypeMeta{
					Kind:       "Secret",
					APIVersion: "v1",
				},
				Type: v1.SecretTypeBasicAuth,
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "incorrect-secret-type",
					Namespace: testNamespace,
				},
				Data: map[string][]byte{
					"username": []byte("test-user"),
					"password": []byte("test-password"),
				},
			},
			expectedError: errSecretTypeMismatch,
		},
	}
	t.Parallel()
	for _, test := range tests {
		tst := test
		t.Run(tst.name, func(t *testing.T) {
			t.Parallel()
			client, err := newTestClientWithSecret(t, tst.secret)
			if err != nil {
				t.Fatal(err)
			}
			_, err = getAndValidateUsageSecret(client, tst.fullName)
			switch {
			case tst.expectedError == nil && err != nil:
				t.Errorf("Didn't expect an error, got %v", err)
			case tst.expectedError != nil && !errors.Is(err, tst.expectedError):
				t.Errorf("Expected %v, got %v", tst.expectedError, err)
			}
		})
	}
}

// This test is included to ensure that the ubbConfigurationTemplate string is
// parsed correctly without throwing an error.
func TestBuildUBBAgentConfig(t *testing.T) {
	tests := []struct {
		name            string
		consumerID      string
		reportingKey    string
		intervalSeconds int
		expectedError   error
	}{
		{
			name:            "google-fake-secret",
			consumerID:      fakeConsumerID,
			reportingKey:    fakeReportingKey,
			intervalSeconds: 30,
		},
		{
			name: "empty-default",
		},
	}
	t.Parallel()
	for _, test := range tests {
		tst := test
		t.Run(tst.name, func(t *testing.T) {
			t.Parallel()
			_, err := buildUBBAgentConfig(tst.consumerID, tst.reportingKey, tst.intervalSeconds)
			switch {
			case tst.expectedError == nil && err != nil:
				t.Errorf("Didn't expect an error, got %v", err)
			case tst.expectedError != nil && !errors.Is(err, tst.expectedError):
				t.Errorf("Expected %v, got %v", tst.expectedError, err)
			}
		})
	}
}

// Verify expected behavior of the UBB Agent metrics function. When given an
// invalid reporting credential, UBB Agent will fail to record metrics and after
// a threshold is reached, the function will emit an error to the channel.
// Expectations:
//   - Function will periodically get the UBB Agent status
//   - After reaching threshold, emit an error to channel
//   - Function will continue to emit errors until shutdown
//
// NOTE: This is a relatively long running test; expect to take 2-3 minutes.
func TestNewStartUBBAgentMeteringWithShutdown(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping long running test")
	}
	// For testing purposes reduce the interval between attempting to send
	// metrics to GCP service endpoint.
	intervalSeconds := 10
	config, err := buildUBBAgentConfig(fakeConsumerID, fakeReportingKey, intervalSeconds)
	if err != nil {
		t.Fatalf("caught an unexpected error from buildUBBAgentConfig: %v", err)
	}
	shutdownCh := make(chan struct{}, 1)
	shutdown := func() { shutdownCh <- struct{}{} }
	defer shutdown()

	errorCh := make(chan error, 1)
	defer close(errorCh)
	errorCount := 0
	go func() {
		for {
			_, ok := <-errorCh
			if !ok {
				return
			}
			t.Log("Error received, increasing count")
			errorCount++
		}
	}()

	newStartUBBAgentMetering(config, shutdownCh)(errorCh)
	reportingDuration := time.Duration(intervalSeconds) * time.Second
	// Sleep long enough for error count to have ticked up by at least one.
	time.Sleep(reportingDuration*(allowedUBBReportingAgentFailureCount+1) + ubbReportingAgentStatusCheckTick + 5*time.Second)
	if errorCount == 0 {
		t.Errorf("Expected error count to be greater than zero: %d", errorCount)
	}
	priorErrorCount := errorCount
	// Sleep for another "reporting duration + check time"ish, error count should have increased
	time.Sleep(reportingDuration + ubbReportingAgentStatusCheckTick + 5*time.Second)
	if priorErrorCount == errorCount {
		t.Errorf("Expected error count to increase: was %d, now %d", priorErrorCount, errorCount)
	}
	shutdown()
	// Sleep to give agent time to shutdown
	time.Sleep(10 * time.Second)
	priorErrorCount = errorCount
	// Sleep for another "reporting duration + check time"ish, error count should not have increased
	time.Sleep(reportingDuration + ubbReportingAgentStatusCheckTick + 5*time.Second)
	if priorErrorCount != errorCount {
		t.Errorf("Expected error count to remain same after UBBAgent shutdown: expected %d, got %d", priorErrorCount, errorCount)
	}
}
