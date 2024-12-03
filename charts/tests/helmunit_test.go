//go:build helmunit

package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/k8s"
)

func TestMain(m *testing.M) {
	code := m.Run()

	// After all tests have run `go-snaps` will sort snapshots
	snaps.Clean(m, snaps.CleanOpts{Sort: true})

	os.Exit(code)
}

// An example of how to verify the rendered template object of a Helm Chart given various inputs.
func TestHelmNICTemplate(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		valuesFile  string
		releaseName string
		namespace   string
		expectedErr error
	}{
		"default values file": {
			valuesFile:  "",
			expectedErr: nil,
			releaseName: "default",
			namespace:   "default",
		},
		"daemonset": {
			valuesFile:  "testdata/daemonset.yaml",
			expectedErr: nil,
			releaseName: "daemonset",
			namespace:   "default",
		},
		"namespace": {
			valuesFile:  "",
			expectedErr: nil,
			releaseName: "namespace",
			namespace:   "nginx-ingress",
		},
		"plus": {
			valuesFile:  "testdata/plus.yaml",
			expectedErr: nil,
			releaseName: "plus",
			namespace:   "default",
		},
		"ingressClass": {
			valuesFile:  "testdata/ingress-class.yaml",
			expectedErr: nil,
			releaseName: "ingress-class",
			namespace:   "default",
		},
		"globalConfig": {
			valuesFile:  "testdata/global-configuration.yaml",
			expectedErr: nil,
			releaseName: "global-configuration",
			namespace:   "gc",
		},
		"customResources": {
			valuesFile:  "testdata/custom-resources.yaml",
			expectedErr: nil,
			releaseName: "custom-resources",
			namespace:   "custom-resources",
		},
		"appProtectWAF": {
			valuesFile:  "testdata/app-protect-waf.yaml",
			expectedErr: nil,
			releaseName: "appprotect-waf",
			namespace:   "appprotect-waf",
		},
		"appProtectWAFV5": {
			valuesFile:  "testdata/app-protect-wafv5.yaml",
			expectedErr: nil,
			releaseName: "appprotect-wafv5",
			namespace:   "appprotect-wafv5",
		},
		"appProtectDOS": {
			valuesFile:  "testdata/app-protect-dos.yaml",
			expectedErr: nil,
			releaseName: "appprotect-dos",
			namespace:   "appprotect-dos",
		},
	}

	// Path to the helm chart we will test
	helmChartPath, err := filepath.Abs("../nginx-ingress")
	if err != nil {
		t.Fatal("Failed to open helm chart path ../nginx-ingress")
	}

	for testName, tc := range tests {
		t.Run(testName, func(t *testing.T) {
			options := &helm.Options{
				KubectlOptions: k8s.NewKubectlOptions("", "", tc.namespace),
			}

			if tc.valuesFile != "" {
				options.ValuesFiles = []string{tc.valuesFile}
			}

			output := helm.RenderTemplate(t, options, helmChartPath, tc.releaseName, make([]string, 0))

			snaps.MatchSnapshot(t, output)
			t.Log(output)
		})
	}
}
