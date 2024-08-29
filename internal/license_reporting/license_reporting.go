package licensereporting

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/golang/glog"
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
	Integration      string `json:"Integration"`
	ClusterID        string `json:"ClusterID"`
	ClusterNodeCount int    `json:"ClusterNodeCount"`
	InstallationID   string `json:"InstallationID"`
}

func newLicenseInfo(clusterID, installationID string, clusterNodeCount int) *licenseInfo {
	return &licenseInfo{
		Integration:      "nic",
		ClusterID:        clusterID,
		InstallationID:   installationID,
		ClusterNodeCount: clusterNodeCount,
	}
}

func writeLicenseInfo(info *licenseInfo) {
	jsonData, err := json.Marshal(info)
	if err != nil {
		glog.Errorf("failed to marshal LicenseInfo to JSON: %v", err)
	}

	filePath := filepath.Join(reportingDir, reportingFile)
	if err := os.WriteFile(filePath, jsonData, 0o600); err != nil {
		glog.Errorf("failed to write license reporting info to file: %v", err)
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
func NewLicenseReporter(cfg LicenseReporterConfig) *LicenseReporter {
	return &LicenseReporter{
		config: cfg,
	}
}

// Start begins the license report writer process for NIC
func (lr *LicenseReporter) Start(ctx context.Context) {
	wait.JitterUntilWithContext(ctx, lr.collectAndWrite, lr.config.Period, 0.1, true)
}

func (lr *LicenseReporter) collectAndWrite(ctx context.Context) {
	clusterID, err := clusterInfo.GetClusterID(ctx, lr.config.K8sClientReader)
	if err != nil {
		glog.Errorf("Error collecting ClusterIDS: %v", err)
	}

	nodeCount, err := clusterInfo.GetNodeCount(ctx, lr.config.K8sClientReader)
	if err != nil {
		glog.Errorf("Error collecting ClusterNodeCount: %v", err)
	}

	installationID, err := clusterInfo.GetInstallationID(ctx, lr.config.K8sClientReader, lr.config.PodNSName)
	if err != nil {
		glog.Errorf("Error collecting InstallationID: %v", err)
	}

	info := newLicenseInfo(clusterID, installationID, nodeCount)
	writeLicenseInfo(info)
}
