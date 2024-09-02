package licensereporting

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
)

func TestNewLicenseInfo(t *testing.T) {
	info := newLicenseInfo("test-cluster", "test-installation", 5)

	if info.Integration != "nic" {
		t.Errorf("newLicenseInfo() Integration = %v, want %v", info.Integration, "nic")
	}
	if info.ClusterID != "test-cluster" {
		t.Errorf("newLicenseInfo() ClusterID = %v, want %v", info.ClusterID, "test-cluster")
	}
	if info.InstallationID != "test-installation" {
		t.Errorf("newLicenseInfo() InstallationID = %v, want %v", info.InstallationID, "test-installation")
	}
	if info.ClusterNodeCount != 5 {
		t.Errorf("newLicenseInfo() ClusterNodeCount = %v, want %v", info.ClusterNodeCount, 5)
	}
}

func TestWriteLicenseInfo(t *testing.T) {
	tempDir := t.TempDir()
	oldReportingDir := reportingDir
	reportingDir = tempDir
	defer func() { reportingDir = oldReportingDir }()

	info := newLicenseInfo("test-cluster", "test-installation", 5)
	writeLicenseInfo(info)

	filePath := filepath.Join(tempDir, reportingFile)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatalf("Expected file %s to exist, but it doesn't", filePath)
	}

	/* #nosec G304 */
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	var readInfo licenseInfo
	err = json.Unmarshal(content, &readInfo)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if readInfo != *info {
		t.Errorf("Written info does not match original. Got %+v, want %+v", readInfo, *info)
	}
}

func TestNewLicenseReporter(t *testing.T) {
	cfg := LicenseReporterConfig{
		Period:          time.Minute,
		K8sClientReader: fake.NewSimpleClientset(),
		PodNSName:       types.NamespacedName{Namespace: "default", Name: "test-pod"},
	}

	reporter := NewLicenseReporter(cfg)
	if reporter == nil {
		t.Fatal("NewLicenseReporter() returned nil")
	}
	if reporter.config != cfg {
		t.Errorf("NewLicenseReporter() config = %v, want %v", reporter.config, cfg)
	}
}
