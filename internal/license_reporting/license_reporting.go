package licensereporting

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	nl "github.com/nginxinc/kubernetes-ingress/internal/logger"

	clusterInfo "github.com/nginxinc/kubernetes-ingress/internal/common_cluster_info"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

var (
	reportingDir  = "/etc/nginx/reporting"
	reportingFile = "tracking.info"
)

type licenseInfo struct {
	Integration      string `json:"integration"`
	ClusterID        string `json:"cluster_id"`
	ClusterNodeCount int    `json:"cluster_node_count"`
	InstallationID   string `json:"installation_id"`
}

func newLicenseInfo(clusterID, installationID string, clusterNodeCount int) *licenseInfo {
	return &licenseInfo{
		Integration:      "nic",
		ClusterID:        clusterID,
		InstallationID:   installationID,
		ClusterNodeCount: clusterNodeCount,
	}
}

func writeLicenseInfo(l *slog.Logger, info *licenseInfo) {
	jsonData, err := json.Marshal(info)
	if err != nil {
		nl.Errorf(l, "failed to marshal LicenseInfo to JSON: %v", err)
		return
	}
	filePath := filepath.Join(reportingDir, reportingFile)
	if err := os.WriteFile(filePath, jsonData, 0o600); err != nil {
		nl.Errorf(l, "failed to write license reporting info to file: %v", err)
	}
}

// LicenseReporter can start the license reporting process
type LicenseReporter struct {
	config LicenseReporterConfig
}

// LicenseReporterConfig contains the information needed for license reporting
type LicenseReporterConfig struct {
	Period          time.Duration
	K8sClientReader kubernetes.Interface
	PodNSName       types.NamespacedName
}

// NewLicenseReporter creates a new LicenseReporter
func NewLicenseReporter(client kubernetes.Interface) *LicenseReporter {
	return &LicenseReporter{
		config: LicenseReporterConfig{
			Period:          24 * time.Hour,
			K8sClientReader: client,
			PodNSName:       types.NamespacedName{Namespace: os.Getenv("POD_NAMESPACE"), Name: os.Getenv("POD_NAME")},
		},
	}
}

// Start begins the license report writer process for NIC
func (lr *LicenseReporter) Start(ctx context.Context) {
	wait.JitterUntilWithContext(ctx, lr.collectAndWrite, lr.config.Period, 0.1, true)
}

func (lr *LicenseReporter) collectAndWrite(ctx context.Context) {
	l := nl.LoggerFromContext(ctx)
	clusterID, err := clusterInfo.GetClusterID(ctx, lr.config.K8sClientReader)
	if err != nil {
		nl.Errorf(l, "Error collecting ClusterIDS: %v", err)
	}
	nodeCount, err := clusterInfo.GetNodeCount(ctx, lr.config.K8sClientReader)
	if err != nil {
		nl.Errorf(l, "Error collecting ClusterNodeCount: %v", err)
	}
	installationID, err := clusterInfo.GetInstallationID(ctx, lr.config.K8sClientReader, lr.config.PodNSName)
	if err != nil {
		nl.Errorf(l, "Error collecting InstallationID: %v", err)
	}
	info := newLicenseInfo(clusterID, installationID, nodeCount)
	writeLicenseInfo(l, info)
}
