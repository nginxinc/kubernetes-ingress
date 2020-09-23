package k8s

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	conf_v1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"
	networking "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createTestConfiguration() *Configuration {
	lbc := LoadBalancerController{
		ingressClass:        "nginx",
		useIngressClassOnly: true,
	}
	return NewConfiguration(lbc.HasCorrectIngressClass, false)
}

func TestAddIngressForRegularIngress(t *testing.T) {
	configuration := createTestConfiguration()

	// no problems are expected for all cases
	var expectedProblems []ConfigurationProblem

	// Add a new Ingress

	ing := createTestIngress("ingress", "foo.example.com")
	expectedChanges := []ResourceChange{
		{
			Op: AddOrUpdate,
			Resource: &FullIngress{
				Ingress: ing,
				ValidHosts: map[string]bool{
					"foo.example.com": true,
				},
				ChildWarnings: map[string][]string{},
			},
		},
	}

	changes, problems := configuration.AddOrUpdateIngress(ing)
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("AddOrUpdateIngress() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("AddOrUpdateIngress() returned unexpected result (-want +got):\n%s", diff)
	}

	// Update the Ingress

	updatedIng := ing.DeepCopy()
	updatedIng.Annotations["nginx.org/max_fails"] = "1"

	expectedChanges = []ResourceChange{
		{
			Op: AddOrUpdate,
			Resource: &FullIngress{
				Ingress: updatedIng,
				ValidHosts: map[string]bool{
					"foo.example.com": true,
				},
				ChildWarnings: map[string][]string{},
			},
		},
	}

	changes, problems = configuration.AddOrUpdateIngress(updatedIng)
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("AddOrUpdateIngress() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("AddOrUpdateIngress() returned unexpected result (-want +got):\n%s", diff)
	}

	// Make the Ingress invalid

	invalidIng := updatedIng.DeepCopy()
	invalidIng.Generation++
	invalidIng.Spec.Rules = []networking.IngressRule{
		{
			Host:             "foo.example.com",
			IngressRuleValue: networking.IngressRuleValue{},
		},
		{
			Host:             "foo.example.com",
			IngressRuleValue: networking.IngressRuleValue{},
		},
	}

	expectedChanges = []ResourceChange{
		{
			Op: Delete,
			Resource: &FullIngress{
				Ingress: updatedIng,
				ValidHosts: map[string]bool{
					"foo.example.com": true,
				},
				ChildWarnings: map[string][]string{},
			},
			Error: `spec.rules[1].host: Duplicate value: "foo.example.com"`,
		},
	}

	changes, problems = configuration.AddOrUpdateIngress(invalidIng)
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("AddOrUpdateIngress() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("AddOrUpdateIngress() returned unexpected result (-want +got):\n%s", diff)
	}

	// Restore the Ingress

	expectedChanges = []ResourceChange{
		{
			Op: AddOrUpdate,
			Resource: &FullIngress{
				Ingress: updatedIng,
				ValidHosts: map[string]bool{
					"foo.example.com": true,
				},
				ChildWarnings: map[string][]string{},
			},
		},
	}

	changes, problems = configuration.AddOrUpdateIngress(updatedIng)
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("AddOrUpdateIngress() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("AddOrUpdateIngress() returned unexpected result (-want +got):\n%s", diff)
	}

	// Update the host of the Ingress

	updatedHostIng := updatedIng.DeepCopy()
	updatedHostIng.Generation++
	updatedHostIng.Spec.Rules = []networking.IngressRule{
		{
			Host:             "bar.example.com",
			IngressRuleValue: networking.IngressRuleValue{},
		},
	}

	expectedChanges = []ResourceChange{
		{
			Op: AddOrUpdate,
			Resource: &FullIngress{
				Ingress: updatedHostIng,
				ValidHosts: map[string]bool{
					"bar.example.com": true,
				},
				ChildWarnings: map[string][]string{},
			},
		},
	}

	changes, problems = configuration.AddOrUpdateIngress(updatedHostIng)
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("AddOrUpdateIngress() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("AddOrUpdateIngress() returned unexpected result (-want +got):\n%s", diff)
	}

	// Delete Ingress

	expectedChanges = []ResourceChange{
		{
			Op: Delete,
			Resource: &FullIngress{
				Ingress: updatedHostIng,
				ValidHosts: map[string]bool{
					"bar.example.com": true,
				},
				ChildWarnings: map[string][]string{},
			},
		},
	}

	changes, problems = configuration.DeleteIngress("default/ingress")
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("DeleteIngress() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("DeleteIngress() returned unexpected result (-want +got):\n%s", diff)
	}
}

func TestAddInvalidIngress(t *testing.T) {
	configuration := createTestConfiguration()

	ing := createTestIngress("ingress", "foo.example.com", "foo.example.com")

	var expectedChanges []ResourceChange
	expectedProblems := []ConfigurationProblem{
		{
			Object:  ing,
			IsError: true,
			Reason:  "Rejected",
			Message: `spec.rules[1].host: Duplicate value: "foo.example.com"`,
		},
	}

	changes, problems := configuration.AddOrUpdateIngress(ing)
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("AddOrUpdateIngress() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("AddOrUpdateIngress() returned unexpected result (-want +got):\n%s", diff)
	}
}

func TestDeleteNonExistingIngress(t *testing.T) {
	configuration := createTestConfiguration()

	var expectedChanges []ResourceChange
	var expectedProblems []ConfigurationProblem

	changes, problems := configuration.DeleteIngress("default/ingress")
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("DeleteIngress() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("DeleteIngress() returned unexpected result (-want +got):\n%s", diff)
	}
}

func TestAddIngressForMergeableIngresses(t *testing.T) {
	configuration := createTestConfiguration()

	// Add  minion-1

	minion1 := createTestIngressMinion("ingress-minion-1", "foo.example.com", "/path-1")
	var expectedChanges []ResourceChange
	expectedProblems := []ConfigurationProblem{
		{
			Object:  minion1,
			Reason:  "NoIngressMasterFound",
			Message: "Ingress master is invalid or doesn't exist",
		},
	}

	changes, problems := configuration.AddOrUpdateIngress(minion1)
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("AddOrUpdateIngress() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("AddOrUpdateIngress() returned unexpected result (-want +got):\n%s", diff)
	}

	// Add master

	master := createTestIngressMaster("ingress-master", "foo.example.com")
	expectedChanges = []ResourceChange{
		{
			Op: AddOrUpdate,
			Resource: &FullIngress{
				Ingress: master,
				ValidHosts: map[string]bool{
					"foo.example.com": true,
				},
				IsMaster: true,
				Minions: []*FullMinion{
					{
						Ingress: minion1,
						ValidPaths: map[string]bool{
							"/path-1": true,
						},
					},
				},
				ChildWarnings: map[string][]string{},
			},
		},
	}
	expectedProblems = nil

	changes, problems = configuration.AddOrUpdateIngress(master)
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("AddOrUpdateIngress() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("AddOrUpdateIngress() returned unexpected result (-want +got):\n%s", diff)
	}

	// Add minion-2

	minion2 := createTestIngressMinion("ingress-minion-2", "foo.example.com", "/path-2")
	expectedChanges = []ResourceChange{
		{
			Op: AddOrUpdate,
			Resource: &FullIngress{
				Ingress: master,
				ValidHosts: map[string]bool{
					"foo.example.com": true,
				},
				IsMaster: true,
				Minions: []*FullMinion{
					{
						Ingress: minion1,
						ValidPaths: map[string]bool{
							"/path-1": true,
						},
					},
					{
						Ingress: minion2,
						ValidPaths: map[string]bool{
							"/path-2": true,
						},
					},
				},
				ChildWarnings: map[string][]string{},
			},
		},
	}
	expectedProblems = nil

	changes, problems = configuration.AddOrUpdateIngress(minion2)
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("AddOrUpdateIngress() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("AddOrUpdateIngress() returned unexpected result (-want +got):\n%s", diff)
	}

	// Update minion-1

	updatedMinion1 := minion1.DeepCopy()
	updatedMinion1.Annotations["nginx.org/proxy-connect-timeout"] = "10s"
	expectedChanges = []ResourceChange{
		{
			Op: AddOrUpdate,
			Resource: &FullIngress{
				Ingress: master,
				ValidHosts: map[string]bool{
					"foo.example.com": true,
				},
				IsMaster: true,
				Minions: []*FullMinion{
					{
						Ingress: updatedMinion1,
						ValidPaths: map[string]bool{
							"/path-1": true,
						},
					},
					{
						Ingress: minion2,
						ValidPaths: map[string]bool{
							"/path-2": true,
						},
					},
				},
				ChildWarnings: map[string][]string{},
			},
		},
	}
	expectedProblems = nil

	changes, problems = configuration.AddOrUpdateIngress(updatedMinion1)
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("AddOrUpdateIngress() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("AddOrUpdateIngress() returned unexpected result (-want +got):\n%s", diff)
	}

	// Make minion-1 invalid

	invalidMinion1 := updatedMinion1.DeepCopy()
	invalidMinion1.Generation++
	invalidMinion1.Spec.Rules = []networking.IngressRule{
		{
			Host:             "example.com",
			IngressRuleValue: networking.IngressRuleValue{},
		},
		{
			Host:             "example.com",
			IngressRuleValue: networking.IngressRuleValue{},
		},
	}

	expectedChanges = []ResourceChange{
		{
			Op: AddOrUpdate,
			Resource: &FullIngress{
				Ingress: master,
				ValidHosts: map[string]bool{
					"foo.example.com": true,
				},
				IsMaster: true,
				Minions: []*FullMinion{
					{
						Ingress: minion2,
						ValidPaths: map[string]bool{
							"/path-2": true,
						},
					},
				},
				ChildWarnings: map[string][]string{},
			},
		},
	}
	expectedProblems = []ConfigurationProblem{
		{
			Object:  invalidMinion1,
			IsError: true,
			Reason:  "Rejected",
			Message: `[spec.rules[1].host: Duplicate value: "example.com", spec.rules: Too many: 2: must have at most 1 items]`,
		},
	}

	changes, problems = configuration.AddOrUpdateIngress(invalidMinion1)
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("AddOrUpdateIngress() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("AddOrUpdateIngress() returned unexpected result (-want +got):\n%s", diff)
	}

	// Restore minion-1

	expectedChanges = []ResourceChange{
		{
			Op: AddOrUpdate,
			Resource: &FullIngress{
				Ingress: master,
				ValidHosts: map[string]bool{
					"foo.example.com": true,
				},
				IsMaster: true,
				Minions: []*FullMinion{
					{
						Ingress: updatedMinion1,
						ValidPaths: map[string]bool{
							"/path-1": true,
						},
					},
					{
						Ingress: minion2,
						ValidPaths: map[string]bool{
							"/path-2": true,
						},
					},
				},
				ChildWarnings: map[string][]string{},
			},
		},
	}
	expectedProblems = nil

	changes, problems = configuration.AddOrUpdateIngress(updatedMinion1)
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("AddOrUpdateIngress() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("AddOrUpdateIngress() returned unexpected result (-want +got):\n%s", diff)
	}

	// Update host of minion-2

	updatedMinion2 := minion2.DeepCopy()
	updatedMinion2.Generation++
	updatedMinion2.Spec.Rules[0].Host = "bar.example.com"

	expectedChanges = []ResourceChange{
		{
			Op: AddOrUpdate,
			Resource: &FullIngress{
				Ingress: master,
				ValidHosts: map[string]bool{
					"foo.example.com": true,
				},
				IsMaster: true,
				Minions: []*FullMinion{
					{
						Ingress: updatedMinion1,
						ValidPaths: map[string]bool{
							"/path-1": true,
						},
					},
				},
				ChildWarnings: map[string][]string{},
			},
		},
	}
	expectedProblems = []ConfigurationProblem{
		{
			Object:  updatedMinion2,
			Reason:  "NoIngressMasterFound",
			Message: "Ingress master is invalid or doesn't exist",
		},
	}

	changes, problems = configuration.AddOrUpdateIngress(updatedMinion2)
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("AddOrUpdateIngress() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("AddOrUpdateIngress() returned unexpected result (-want +got):\n%s", diff)
	}

	// Update host of master

	updatedMaster := master.DeepCopy()
	updatedMaster.Generation++
	updatedMaster.Spec.Rules[0].Host = "bar.example.com"

	expectedChanges = []ResourceChange{
		{
			Op: AddOrUpdate,
			Resource: &FullIngress{
				Ingress: updatedMaster,
				ValidHosts: map[string]bool{
					"bar.example.com": true,
				},
				IsMaster: true,
				Minions: []*FullMinion{
					{
						Ingress: updatedMinion2,
						ValidPaths: map[string]bool{
							"/path-2": true,
						},
					},
				},
				ChildWarnings: map[string][]string{},
			},
		},
	}
	expectedProblems = []ConfigurationProblem{
		{
			Object:  updatedMinion1,
			Reason:  "NoIngressMasterFound",
			Message: "Ingress master is invalid or doesn't exist",
		},
	}

	changes, problems = configuration.AddOrUpdateIngress(updatedMaster)
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("AddOrUpdateIngress() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("AddOrUpdateIngress() returned unexpected result (-want +got):\n%s", diff)
	}

	// Restore host
	expectedChanges = []ResourceChange{
		{
			Op: AddOrUpdate,
			Resource: &FullIngress{
				Ingress: master,
				ValidHosts: map[string]bool{
					"foo.example.com": true,
				},
				IsMaster: true,
				Minions: []*FullMinion{
					{
						Ingress: updatedMinion1,
						ValidPaths: map[string]bool{
							"/path-1": true,
						},
					},
				},
				ChildWarnings: map[string][]string{},
			},
		},
	}
	expectedProblems = []ConfigurationProblem{
		{
			Object:  updatedMinion2,
			Reason:  "NoIngressMasterFound",
			Message: "Ingress master is invalid or doesn't exist",
		},
	}

	changes, problems = configuration.AddOrUpdateIngress(master)
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("AddOrUpdateIngress() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("AddOrUpdateIngress() returned unexpected result (-want +got):\n%s", diff)
	}

	// Restore host of minion-2

	expectedChanges = []ResourceChange{
		{
			Op: AddOrUpdate,
			Resource: &FullIngress{
				Ingress: master,
				ValidHosts: map[string]bool{
					"foo.example.com": true,
				},
				IsMaster: true,
				Minions: []*FullMinion{
					{
						Ingress: updatedMinion1,
						ValidPaths: map[string]bool{
							"/path-1": true,
						},
					},
					{
						Ingress: minion2,
						ValidPaths: map[string]bool{
							"/path-2": true,
						},
					},
				},
				ChildWarnings: map[string][]string{},
			},
		},
	}
	expectedProblems = nil

	changes, problems = configuration.AddOrUpdateIngress(minion2)
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("AddOrUpdateIngress() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("AddOrUpdateIngress() returned unexpected result (-want +got):\n%s", diff)
	}

	// Remove minion-1

	expectedChanges = []ResourceChange{
		{
			Op: AddOrUpdate,
			Resource: &FullIngress{
				Ingress: master,
				ValidHosts: map[string]bool{
					"foo.example.com": true,
				},
				IsMaster: true,
				Minions: []*FullMinion{
					{
						Ingress: minion2,
						ValidPaths: map[string]bool{
							"/path-2": true,
						},
					},
				},
				ChildWarnings: map[string][]string{},
			},
		},
	}
	expectedProblems = nil

	changes, problems = configuration.DeleteIngress("default/ingress-minion-1")
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("DeleteIngress() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("DeleteIngress() returned unexpected result (-want +got):\n%s", diff)
	}

	// Remove master

	expectedChanges = []ResourceChange{
		{
			Op: Delete,
			Resource: &FullIngress{
				Ingress: master,
				ValidHosts: map[string]bool{
					"foo.example.com": true,
				},
				IsMaster: true,
				Minions: []*FullMinion{
					{
						Ingress: minion2,
						ValidPaths: map[string]bool{
							"/path-2": true,
						},
					},
				},
				ChildWarnings: map[string][]string{},
			},
		},
	}
	expectedProblems = []ConfigurationProblem{
		{
			Object:  minion2,
			Reason:  "NoIngressMasterFound",
			Message: "Ingress master is invalid or doesn't exist",
		},
	}

	changes, problems = configuration.DeleteIngress("default/ingress-master")
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("DeleteIngress() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("DeleteIngress() returned unexpected result (-want +got):\n%s", diff)
	}

	// Remove minion-2

	expectedChanges = nil
	expectedProblems = nil

	changes, problems = configuration.DeleteIngress("default/ingress-minion-2")
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("DeleteIngress() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("DeleteIngress() returned unexpected result (-want +got):\n%s", diff)
	}
}

func TestMinionPathCollisions(t *testing.T) {
	configuration := createTestConfiguration()

	// Add master

	master := createTestIngressMaster("ingress-master", "foo.example.com")
	expectedChanges := []ResourceChange{
		{
			Op: AddOrUpdate,
			Resource: &FullIngress{
				Ingress: master,
				ValidHosts: map[string]bool{
					"foo.example.com": true,
				},
				IsMaster:      true,
				ChildWarnings: map[string][]string{},
			},
		},
	}
	var expectedProblems []ConfigurationProblem

	changes, problems := configuration.AddOrUpdateIngress(master)
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("AddOrUpdateIngress() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("AddOrUpdateIngress() returned unexpected result (-want +got):\n%s", diff)
	}

	// Add  minion-1

	minion1 := createTestIngressMinion("ingress-minion-1", "foo.example.com", "/")
	expectedChanges = []ResourceChange{
		{
			Op: AddOrUpdate,
			Resource: &FullIngress{
				Ingress: master,
				ValidHosts: map[string]bool{
					"foo.example.com": true,
				},
				IsMaster: true,
				Minions: []*FullMinion{
					{
						Ingress: minion1,
						ValidPaths: map[string]bool{
							"/": true,
						},
					},
				},
				ChildWarnings: map[string][]string{},
			},
		},
	}
	expectedProblems = nil

	changes, problems = configuration.AddOrUpdateIngress(minion1)
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("AddOrUpdateIngress() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("AddOrUpdateIngress() returned unexpected result (-want +got):\n%s", diff)
	}

	// Add minion-2

	minion2 := createTestIngressMinion("ingress-minion-2", "foo.example.com", "/")
	expectedChanges = []ResourceChange{
		{
			Op: AddOrUpdate,
			Resource: &FullIngress{
				Ingress: master,
				ValidHosts: map[string]bool{
					"foo.example.com": true,
				},
				IsMaster: true,
				Minions: []*FullMinion{
					{
						Ingress: minion1,
						ValidPaths: map[string]bool{
							"/": true,
						},
					},
					{
						Ingress:    minion2,
						ValidPaths: map[string]bool{},
					},
				},
				ChildWarnings: map[string][]string{
					"default/ingress-minion-2": {
						"path / is taken by another resource",
					},
				},
			},
		},
	}
	expectedProblems = nil

	changes, problems = configuration.AddOrUpdateIngress(minion2)
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("AddOrUpdateIngress() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("AddOrUpdateIngress() returned unexpected result (-want +got):\n%s", diff)
	}

	// Delete minion-1

	expectedChanges = []ResourceChange{
		{
			Op: AddOrUpdate,
			Resource: &FullIngress{
				Ingress: master,
				ValidHosts: map[string]bool{
					"foo.example.com": true,
				},
				IsMaster: true,
				Minions: []*FullMinion{
					{
						Ingress: minion2,
						ValidPaths: map[string]bool{
							"/": true,
						},
					},
				},
				ChildWarnings: map[string][]string{},
			},
		},
	}
	expectedProblems = nil

	changes, problems = configuration.DeleteIngress("default/ingress-minion-1")
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("DeleteIngress() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("DeleteIngress() returned unexpected result (-want +got):\n%s", diff)
	}
}

func TestAddIngressWithIncorrectClass(t *testing.T) {
	configuration := createTestConfiguration()

	// Add Ingress with incorrect class

	ing := createTestIngress("regular-ingress", "foo.example.com")
	ing.Annotations["kubernetes.io/ingress.class"] = "someproxy"

	var expectedChanges []ResourceChange
	var expectedProblems []ConfigurationProblem

	changes, problems := configuration.AddOrUpdateIngress(ing)
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("AddOrUpdateIngress() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("AddOrUpdateIngress() returned unexpected result (-want +got):\n%s", diff)
	}

	// Make the class correct

	updatedIng := ing.DeepCopy()
	updatedIng.Annotations["kubernetes.io/ingress.class"] = "nginx"

	expectedChanges = []ResourceChange{
		{
			Op: AddOrUpdate,
			Resource: &FullIngress{
				Ingress: updatedIng,
				ValidHosts: map[string]bool{
					"foo.example.com": true,
				},
				ChildWarnings: map[string][]string{},
			},
		},
	}
	expectedProblems = nil

	changes, problems = configuration.AddOrUpdateIngress(updatedIng)
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("AddOrUpdateIngress() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("AddOrUpdateIngress() returned unexpected result (-want +got):\n%s", diff)
	}

	// Make the class incorrect

	expectedChanges = []ResourceChange{
		{
			Op: Delete,
			Resource: &FullIngress{
				Ingress: updatedIng,
				ValidHosts: map[string]bool{
					"foo.example.com": true,
				},
				ChildWarnings: map[string][]string{},
			},
		},
	}
	expectedProblems = nil

	changes, problems = configuration.AddOrUpdateIngress(ing)
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("AddOrUpdateIngress() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("AddOrUpdateIngress() returned unexpected result (-want +got):\n%s", diff)
	}
}

func TestAddVirtualServer(t *testing.T) {
	configuration := createTestConfiguration()

	// no problems are expected for all cases
	var expectedProblems []ConfigurationProblem

	// Add a VirtualServer

	vs := createTestVirtualServer("virtualserver", "foo.example.com")
	expectedChanges := []ResourceChange{
		{
			Op: AddOrUpdate,
			Resource: &FullVirtualServer{
				VirtualServer: vs,
			},
		},
	}

	changes, problems := configuration.AddOrUpdateVirtualServer(vs)
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("AddOrUpdateVirtualServer() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("AddOrUpdateVirtualServer() returned unexpected result (-want +got):\n%s", diff)
	}

	// Update VirtualServer

	updatedVS := vs.DeepCopy()
	updatedVS.Generation++
	updatedVS.Spec.ServerSnippets = "# snippet"

	expectedChanges = []ResourceChange{
		{
			Op: AddOrUpdate,
			Resource: &FullVirtualServer{
				VirtualServer: updatedVS,
			},
		},
	}

	changes, problems = configuration.AddOrUpdateVirtualServer(updatedVS)
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("AddOrUpdateVirtualServer() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("AddOrUpdateVirtualServer() returned unexpected result (-want +got):\n%s", diff)
	}

	// Make VirtualServer invalid

	invalidVS := updatedVS.DeepCopy()
	invalidVS.Generation++
	invalidVS.Spec.Host = ""

	expectedChanges = []ResourceChange{
		{
			Op: Delete,
			Resource: &FullVirtualServer{
				VirtualServer: updatedVS,
			},
			Error: "spec.host: Required value",
		},
	}

	changes, problems = configuration.AddOrUpdateVirtualServer(invalidVS)
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("AddOrUpdateVirtualServer() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("AddOrUpdateVirtualServer() returned unexpected result (-want +got):\n%s", diff)
	}

	// Restore VirtualServer

	expectedChanges = []ResourceChange{
		{
			Op: AddOrUpdate,
			Resource: &FullVirtualServer{
				VirtualServer: updatedVS,
			},
		},
	}

	changes, problems = configuration.AddOrUpdateVirtualServer(updatedVS)
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("AddOrUpdateVirtualServer() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("AddOrUpdateVirtualServer() returned unexpected result (-want +got):\n%s", diff)
	}

	// Update VirtualServer host

	updatedHostVS := updatedVS.DeepCopy()
	updatedHostVS.Generation++
	updatedHostVS.Spec.Host = "bar.example.com"

	expectedChanges = []ResourceChange{
		{
			Op: AddOrUpdate,
			Resource: &FullVirtualServer{
				VirtualServer: updatedHostVS,
			},
		},
	}

	changes, problems = configuration.AddOrUpdateVirtualServer(updatedHostVS)
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("AddOrUpdateVirtualServer() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("AddOrUpdateVirtualServer() returned unexpected result (-want +got):\n%s", diff)
	}

	// Delete VirtualServer

	expectedChanges = []ResourceChange{
		{
			Op: Delete,
			Resource: &FullVirtualServer{
				VirtualServer: updatedHostVS,
			},
		},
	}

	changes, problems = configuration.DeleteVirtualServer("default/virtualserver")
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("DeleteVirtualServer() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("DeleteVirtualServer() returned unexpected result (-want +got):\n%s", diff)
	}
}

func TestAddInvalidVirtualServer(t *testing.T) {
	configuration := createTestConfiguration()

	vs := createTestVirtualServer("virtualserver", "")

	var expectedChanges []ResourceChange
	expectedProblems := []ConfigurationProblem{
		{
			Object:  vs,
			IsError: true,
			Reason:  "Rejected",
			Message: "spec.host: Required value",
		},
	}

	changes, problems := configuration.AddOrUpdateVirtualServer(vs)
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("AddOrUpdateVirtualServer() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("AddOrUpdateVirtualServer() returned unexpected result (-want +got):\n%s", diff)
	}
}

func TestAddInvalidVirtualServerWithIncorrectClass(t *testing.T) {
	configuration := createTestConfiguration()

	// Add VirtualServer with incorrect class

	vs := createTestVirtualServer("virtualserver", "example.com")
	vs.Spec.IngressClass = "someproxy"

	var expectedChanges []ResourceChange
	var expectedProblems []ConfigurationProblem

	changes, problems := configuration.AddOrUpdateVirtualServer(vs)
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("AddOrUpdateVirtualServer() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("AddOrUpdateVirtualServer() returned unexpected result (-want +got):\n%s", diff)
	}

	// Make the class correct

	updatedVS := vs.DeepCopy()
	updatedVS.Generation++
	updatedVS.Spec.IngressClass = "nginx"

	expectedChanges = []ResourceChange{
		{
			Op: AddOrUpdate,
			Resource: &FullVirtualServer{
				VirtualServer: updatedVS,
			},
		},
	}
	expectedProblems = nil

	changes, problems = configuration.AddOrUpdateVirtualServer(updatedVS)
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("AddOrUpdateVirtualServer() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("AddOrUpdateVirtualServer() returned unexpected result (-want +got):\n%s", diff)
	}

	// Make the class incorrect

	expectedChanges = []ResourceChange{
		{
			Op: Delete,
			Resource: &FullVirtualServer{
				VirtualServer: updatedVS,
			},
		},
	}
	expectedProblems = nil

	changes, problems = configuration.AddOrUpdateVirtualServer(vs)
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("AddOrUpdateVirtualServer() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("AddOrUpdateVirtualServer() returned unexpected result (-want +got):\n%s", diff)
	}
}

func TestDeleteNonExistingVirtualServer(t *testing.T) {
	configuration := createTestConfiguration()

	var expectedChanges []ResourceChange
	var expectedProblems []ConfigurationProblem

	changes, problems := configuration.DeleteVirtualServer("default/virtualserver")
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("DeleteVirtualServer() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("DeleteVirtualServer() returned unexpected result (-want +got):\n%s", diff)
	}
}

func TestAddVirtualServerWithVirtualServerRoutes(t *testing.T) {
	configuration := createTestConfiguration()

	// Add VirtualServerRoute-1

	vsr1 := createTestVirtualServerRoute("virtualserverroute-1", "foo.example.com", "/first")
	var expectedChanges []ResourceChange
	expectedProblems := []ConfigurationProblem{
		{
			Object:  vsr1,
			Reason:  "NoVirtualServerFound",
			Message: "VirtualServer is invalid or doesn't exist",
		},
	}

	changes, problems := configuration.AddOrUpdateVirtualServerRoute(vsr1)
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("AddOrUpdateVirtualServerRoute() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("AddOrUpdateVirtualServerRoute() returned unexpected result (-want +got):\n%s", diff)
	}

	// Add VirtualServer

	vs := createTestVirtualServerWithRoutes(
		"virtualserver",
		"foo.example.com",
		[]conf_v1.Route{
			{
				Path:  "/first",
				Route: "virtualserverroute-1",
			},
			{
				Path:  "/second",
				Route: "virtualserverroute-2",
			},
		})
	expectedChanges = []ResourceChange{
		{
			Op: AddOrUpdate,
			Resource: &FullVirtualServer{
				VirtualServer:       vs,
				VirtualServerRoutes: []*conf_v1.VirtualServerRoute{vsr1},
				Warnings:            []string{"VirtualServerRoute default/virtualserverroute-2 doesn't exist or invalid"},
			},
		},
	}
	expectedProblems = nil

	changes, problems = configuration.AddOrUpdateVirtualServer(vs)
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("AddOrUpdateVirtualServer() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("AddOrUpdateVirtualServer() returned unexpected result (-want +got):\n%s", diff)
	}

	vsr2 := createTestVirtualServerRoute("virtualserverroute-2", "foo.example.com", "/second")

	// Add VirtualServerRoute-2

	expectedChanges = []ResourceChange{
		{
			Op: AddOrUpdate,
			Resource: &FullVirtualServer{
				VirtualServer:       vs,
				VirtualServerRoutes: []*conf_v1.VirtualServerRoute{vsr1, vsr2},
			},
		},
	}
	expectedProblems = nil

	changes, problems = configuration.AddOrUpdateVirtualServerRoute(vsr2)
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("AddOrUpdateVirtualServerRoute() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("AddOrUpdateVirtualServerRoute() returned unexpected result (-want +got):\n%s", diff)
	}

	// Update VirtualServerRoute-1

	updatedVSR1 := vsr1.DeepCopy()
	updatedVSR1.Generation++
	updatedVSR1.Spec.Subroutes[0].LocationSnippets = "# snippet"
	expectedChanges = []ResourceChange{
		{
			Op: AddOrUpdate,
			Resource: &FullVirtualServer{
				VirtualServer:       vs,
				VirtualServerRoutes: []*conf_v1.VirtualServerRoute{updatedVSR1, vsr2},
			},
		},
	}
	expectedProblems = nil

	changes, problems = configuration.AddOrUpdateVirtualServerRoute(updatedVSR1)
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("AddOrUpdateVirtualServerRoute() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("AddOrUpdateVirtualServerRoute() returned unexpected result (-want +got):\n%s", diff)
	}

	// Make VirtualServerRoute-1 invalid

	invalidVSR1 := updatedVSR1.DeepCopy()
	invalidVSR1.Generation++
	invalidVSR1.Spec.Host = ""
	expectedChanges = []ResourceChange{
		{
			Op: AddOrUpdate,
			Resource: &FullVirtualServer{
				VirtualServer:       vs,
				VirtualServerRoutes: []*conf_v1.VirtualServerRoute{vsr2},
				Warnings:            []string{"VirtualServerRoute default/virtualserverroute-1 doesn't exist or invalid"},
			},
		},
	}
	expectedProblems = []ConfigurationProblem{
		{
			Object:  invalidVSR1,
			IsError: true,
			Reason:  "Rejected",
			Message: "spec.host: Required value",
		},
	}

	changes, problems = configuration.AddOrUpdateVirtualServerRoute(invalidVSR1)
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("AddOrUpdateVirtualServerRoute() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("AddOrUpdateVirtualServerRoute() returned unexpected result (-want +got):\n%s", diff)
	}

	// Restore VirtualServerRoute-1

	expectedChanges = []ResourceChange{
		{
			Op: AddOrUpdate,
			Resource: &FullVirtualServer{
				VirtualServer:       vs,
				VirtualServerRoutes: []*conf_v1.VirtualServerRoute{vsr1, vsr2},
			},
		},
	}
	expectedProblems = nil

	changes, problems = configuration.AddOrUpdateVirtualServerRoute(vsr1)
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("AddOrUpdateVirtualServerRoute() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("AddOrUpdateVirtualServerRoute() returned unexpected result (-want +got):\n%s", diff)
	}

	// Make VirtualServerRoute-1 invalid for VirtualServer

	invalidForVSVSR1 := vsr1.DeepCopy()
	invalidForVSVSR1.Generation++
	invalidForVSVSR1.Spec.Subroutes[0].Path = "/"
	expectedChanges = []ResourceChange{
		{
			Op: AddOrUpdate,
			Resource: &FullVirtualServer{
				VirtualServer:       vs,
				VirtualServerRoutes: []*conf_v1.VirtualServerRoute{vsr2},
				Warnings:            []string{"VirtualServerRoute default/virtualserverroute-1 is invalid: spec.subroutes[0]: Invalid value: \"/\": must start with '/first'"},
			},
		},
	}
	expectedProblems = []ConfigurationProblem{
		{
			Object:  invalidForVSVSR1,
			Reason:  "Ignored",
			Message: "VirtualServer default/virtualserver ignores VirtualServerRoute",
		},
	}

	changes, problems = configuration.AddOrUpdateVirtualServerRoute(invalidForVSVSR1)
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("AddOrUpdateVirtualServerRoute() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("AddOrUpdateVirtualServerRoute() returned unexpected result (-want +got):\n%s", diff)
	}

	// Restore VirtualServerRoute-1

	expectedChanges = []ResourceChange{
		{
			Op: AddOrUpdate,
			Resource: &FullVirtualServer{
				VirtualServer:       vs,
				VirtualServerRoutes: []*conf_v1.VirtualServerRoute{vsr1, vsr2},
			},
		},
	}
	expectedProblems = nil

	changes, problems = configuration.AddOrUpdateVirtualServerRoute(vsr1)
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("AddOrUpdateVirtualServerRoute() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("AddOrUpdateVirtualServerRoute() returned unexpected result (-want +got):\n%s", diff)
	}

	// Update host of VirtualServerRoute-2

	updatedVSR2 := vsr2.DeepCopy()
	updatedVSR2.Generation++
	updatedVSR2.Spec.Host = "bar.example.com"
	expectedChanges = []ResourceChange{
		{
			Op: AddOrUpdate,
			Resource: &FullVirtualServer{
				VirtualServer:       vs,
				VirtualServerRoutes: []*conf_v1.VirtualServerRoute{vsr1},
				Warnings:            []string{"VirtualServerRoute default/virtualserverroute-2 is invalid: spec.host: Invalid value: \"bar.example.com\": must be equal to 'foo.example.com'"},
			},
		},
	}
	expectedProblems = []ConfigurationProblem{
		{
			Object:  updatedVSR2,
			Reason:  "NoVirtualServerFound",
			Message: "VirtualServer is invalid or doesn't exist",
		},
	}

	changes, problems = configuration.AddOrUpdateVirtualServerRoute(updatedVSR2)
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("AddOrUpdateVirtualServerRoute() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("AddOrUpdateVirtualServerRoute() returned unexpected result (-want +got):\n%s", diff)
	}

	// Update host of VirtualServer

	updatedVS := vs.DeepCopy()
	updatedVS.Generation++
	updatedVS.Spec.Host = "bar.example.com"
	expectedChanges = []ResourceChange{
		{
			Op: AddOrUpdate,
			Resource: &FullVirtualServer{
				VirtualServer:       updatedVS,
				VirtualServerRoutes: []*conf_v1.VirtualServerRoute{updatedVSR2},
				Warnings:            []string{"VirtualServerRoute default/virtualserverroute-1 is invalid: spec.host: Invalid value: \"foo.example.com\": must be equal to 'bar.example.com'"},
			},
		},
	}
	expectedProblems = []ConfigurationProblem{
		{
			Object:  vsr1,
			Reason:  "NoVirtualServerFound",
			Message: "VirtualServer is invalid or doesn't exist",
		},
	}

	changes, problems = configuration.AddOrUpdateVirtualServer(updatedVS)
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("AddOrUpdateVirtualServer() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("AddOrUpdateVirtualServer() returned unexpected result (-want +got):\n%s", diff)
	}

	// Restore host of VirtualServer

	expectedChanges = []ResourceChange{
		{
			Op: AddOrUpdate,
			Resource: &FullVirtualServer{
				VirtualServer:       vs,
				VirtualServerRoutes: []*conf_v1.VirtualServerRoute{vsr1},
				Warnings:            []string{"VirtualServerRoute default/virtualserverroute-2 is invalid: spec.host: Invalid value: \"bar.example.com\": must be equal to 'foo.example.com'"},
			},
		},
	}
	expectedProblems = []ConfigurationProblem{
		{
			Object:  updatedVSR2,
			Reason:  "NoVirtualServerFound",
			Message: "VirtualServer is invalid or doesn't exist",
		},
	}

	changes, problems = configuration.AddOrUpdateVirtualServer(vs)
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("AddOrUpdateVirtualServer() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("AddOrUpdateVirtualServer() returned unexpected result (-want +got):\n%s", diff)
	}

	// Restore host of VirtualServerRoute-2

	expectedChanges = []ResourceChange{
		{
			Op: AddOrUpdate,
			Resource: &FullVirtualServer{
				VirtualServer:       vs,
				VirtualServerRoutes: []*conf_v1.VirtualServerRoute{vsr1, vsr2},
			},
		},
	}
	expectedProblems = nil

	changes, problems = configuration.AddOrUpdateVirtualServerRoute(vsr2)
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("AddOrUpdateVirtualServerRoute() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("AddOrUpdateVirtualServerRoute() returned unexpected result (-want +got):\n%s", diff)
	}

	// Remove VirtualServerRoute-1

	expectedChanges = []ResourceChange{
		{
			Op: AddOrUpdate,
			Resource: &FullVirtualServer{
				VirtualServer:       vs,
				VirtualServerRoutes: []*conf_v1.VirtualServerRoute{vsr2},
				Warnings:            []string{"VirtualServerRoute default/virtualserverroute-1 doesn't exist or invalid"},
			},
		},
	}
	expectedProblems = nil

	changes, problems = configuration.DeleteVirtualServerRoute("default/virtualserverroute-1")
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("DeleteVirtualServerRoute() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("DeleteVirtualServerRoute() returned unexpected result (-want +got):\n%s", diff)
	}

	// Remove VirtualServer

	expectedChanges = []ResourceChange{
		{
			Op: Delete,
			Resource: &FullVirtualServer{
				VirtualServer:       vs,
				VirtualServerRoutes: []*conf_v1.VirtualServerRoute{vsr2},
				Warnings:            []string{"VirtualServerRoute default/virtualserverroute-1 doesn't exist or invalid"},
			},
		},
	}
	expectedProblems = []ConfigurationProblem{
		{
			Object:  vsr2,
			Reason:  "NoVirtualServerFound",
			Message: "VirtualServer is invalid or doesn't exist",
		},
	}

	changes, problems = configuration.DeleteVirtualServer("default/virtualserver")
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("DeleteVirtualServer() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("DeleteVirtualServer() returned unexpected result (-want +got):\n%s", diff)
	}

	// Remove VirtualServerRoute-2

	expectedChanges = nil
	expectedProblems = nil

	changes, problems = configuration.DeleteVirtualServerRoute("default/virtualserverroute-2")
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("DeleteVirtualServerRoute() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("DeleteVirtualServerRoute() returned unexpected result (-want +got):\n%s", diff)
	}
}

func TestAddInvalidVirtualServerRoute(t *testing.T) {
	configuration := createTestConfiguration()

	vsr := createTestVirtualServerRoute("virtualserverroute", "", "/")

	var expectedChanges []ResourceChange
	expectedProblems := []ConfigurationProblem{
		{
			Object:  vsr,
			IsError: true,
			Reason:  "Rejected",
			Message: "spec.host: Required value",
		},
	}

	changes, problems := configuration.AddOrUpdateVirtualServerRoute(vsr)
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("AddOrUpdateVirtualServerRoute() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("AddOrUpdateVirtualServerRoute() returned unexpected result (-want +got):\n%s", diff)
	}
}

func TestAddVirtualServerWithIncorrectClass(t *testing.T) {
	configuration := createTestConfiguration()

	vsr := createTestVirtualServerRoute("virtualserver", "foo.example.com", "/")
	vsr.Spec.IngressClass = "someproxy"

	var expectedChanges []ResourceChange
	var expectedProblems []ConfigurationProblem

	changes, problems := configuration.AddOrUpdateVirtualServerRoute(vsr)
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("AddOrUpdateVirtualServerRoute() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("AddOrUpdateVirtualServerRoute() returned unexpected result (-want +got):\n%s", diff)
	}
}

func TestDeleteNonExistingVirtualServerRoute(t *testing.T) {
	configuration := createTestConfiguration()

	var expectedChanges []ResourceChange
	var expectedProblems []ConfigurationProblem

	changes, problems := configuration.DeleteVirtualServerRoute("default/virtualserverroute")
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("DeleteVirtualServerRoute() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("DeleteVirtualServerRoute() returned unexpected result (-want +got):\n%s", diff)
	}
}

func TestHostCollisions(t *testing.T) {
	configuration := createTestConfiguration()

	var expectedProblems []ConfigurationProblem

	masterIng := createTestIngressMaster("master-ingress", "foo.example.com")
	regularIng := createTestIngress("regular-ingress", "foo.example.com", "bar.example.com")
	vs := createTestVirtualServer("virtualserver", "foo.example.com")
	regularIng2 := createTestIngress("regular-ingress-2", "foo.example.com")

	// Add VirtualServer

	expectedChanges := []ResourceChange{
		{
			Op: AddOrUpdate,
			Resource: &FullVirtualServer{
				VirtualServer: vs,
			},
		},
	}
	expectedProblems = nil

	changes, problems := configuration.AddOrUpdateVirtualServer(vs)
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("AddOrUpdateVirtualServer() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("AddOrUpdateVirtualServer() returned unexpected result (-want +got):\n%s", diff)
	}

	// Add regular Ingress

	expectedChanges = []ResourceChange{
		{
			Op: Delete,
			Resource: &FullVirtualServer{
				VirtualServer: vs,
				Warnings:      []string{"host foo.example.com is taken by another resource"},
			},
		},
		{
			Op: AddOrUpdate,
			Resource: &FullIngress{
				Ingress:       regularIng,
				ValidHosts:    map[string]bool{"foo.example.com": true, "bar.example.com": true},
				ChildWarnings: map[string][]string{},
			},
		},
	}
	expectedProblems = []ConfigurationProblem{
		{
			Object:  vs,
			IsError: false,
			Reason:  "Rejected",
			Message: "Host is taken by another resource",
		},
	}

	changes, problems = configuration.AddOrUpdateIngress(regularIng)
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("AddOrUpdateIngress() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("AddOrUpdateIngress() returned unexpected result (-want +got):\n%s", diff)
	}

	// Add master Ingress

	expectedChanges = []ResourceChange{
		{
			Op: AddOrUpdate,
			Resource: &FullIngress{
				Ingress:       regularIng,
				ValidHosts:    map[string]bool{"bar.example.com": true},
				Warnings:      []string{"host foo.example.com is taken by another resource"},
				ChildWarnings: map[string][]string{},
			},
		},
		{
			Op: AddOrUpdate,
			Resource: &FullIngress{
				Ingress:       masterIng,
				IsMaster:      true,
				ValidHosts:    map[string]bool{"foo.example.com": true},
				ChildWarnings: map[string][]string{},
			},
		},
	}
	expectedProblems = nil

	changes, problems = configuration.AddOrUpdateIngress(masterIng)
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("AddOrUpdateIngress() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("AddOrUpdateIngress() returned unexpected result (-want +got):\n%s", diff)
	}

	// Add regular Ingress-2

	expectedChanges = nil
	expectedProblems = []ConfigurationProblem{
		{
			Object:  regularIng2,
			IsError: false,
			Reason:  "Rejected",
			Message: "All hosts are taken by other resources",
		},
	}

	changes, problems = configuration.AddOrUpdateIngress(regularIng2)
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("AddOrUpdateIngress() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("AddOrUpdateIngress() returned unexpected result (-want +got):\n%s", diff)
	}

	// Delete regular Ingress-2
	expectedChanges = nil
	expectedProblems = nil

	changes, problems = configuration.DeleteIngress("default/regular-ingress-2")
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("DeleteIngress() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("DeleteIngress() returned unexpected result (-want +got):\n%s", diff)
	}

	// Delete master Ingress

	expectedChanges = []ResourceChange{
		{
			Op: Delete,
			Resource: &FullIngress{
				Ingress:       masterIng,
				IsMaster:      true,
				ValidHosts:    map[string]bool{"foo.example.com": true},
				ChildWarnings: map[string][]string{},
			},
		},
		{
			Op: AddOrUpdate,
			Resource: &FullIngress{
				Ingress:       regularIng,
				ValidHosts:    map[string]bool{"foo.example.com": true, "bar.example.com": true},
				ChildWarnings: map[string][]string{},
			},
		},
	}
	expectedProblems = nil

	changes, problems = configuration.DeleteIngress("default/master-ingress")
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("DeleteIngress() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("DeleteIngress() returned unexpected result (-want +got):\n%s", diff)
	}

	// Delete regular Ingress

	expectedChanges = []ResourceChange{
		{
			Op: Delete,
			Resource: &FullIngress{
				Ingress:       regularIng,
				ValidHosts:    map[string]bool{"foo.example.com": true, "bar.example.com": true},
				ChildWarnings: map[string][]string{},
			},
		},
		{
			Op: AddOrUpdate,
			Resource: &FullVirtualServer{
				VirtualServer: vs,
			},
		},
	}
	expectedProblems = nil

	changes, problems = configuration.DeleteIngress("default/regular-ingress")
	if diff := cmp.Diff(expectedChanges, changes); diff != "" {
		t.Errorf("DeleteIngress() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedProblems, problems); diff != "" {
		t.Errorf("DeleteIngress() returned unexpected result (-want +got):\n%s", diff)
	}
}

func createTestIngressMaster(name string, host string) *networking.Ingress {
	ing := createTestIngress(name, host)
	ing.Annotations["nginx.org/mergeable-ingress-type"] = "master"
	return ing
}

func createTestIngressMinion(name string, host string, path string) *networking.Ingress {
	ing := createTestIngress(name, host)
	ing.Spec.Rules[0].IngressRuleValue = networking.IngressRuleValue{
		HTTP: &networking.HTTPIngressRuleValue{
			Paths: []networking.HTTPIngressPath{
				{
					Path: path,
				},
			},
		},
	}

	ing.Annotations["nginx.org/mergeable-ingress-type"] = "minion"

	return ing
}

func createTestIngress(name string, hosts ...string) *networking.Ingress {
	var rules []networking.IngressRule

	for _, h := range hosts {
		rules = append(rules, networking.IngressRule{
			Host:             h,
			IngressRuleValue: networking.IngressRuleValue{},
		})
	}

	return &networking.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         "default",
			CreationTimestamp: metav1.Now(),
			Annotations: map[string]string{
				"kubernetes.io/ingress.class": "nginx",
			},
		},
		Spec: networking.IngressSpec{
			Rules: rules,
		},
	}
}

func createTestVirtualServer(name string, host string) *conf_v1.VirtualServer {
	return &conf_v1.VirtualServer{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         "default",
			Name:              name,
			CreationTimestamp: metav1.Now(),
		},
		Spec: conf_v1.VirtualServerSpec{
			IngressClass: "nginx",
			Host:         host,
		},
	}
}

func createTestVirtualServerWithRoutes(name string, host string, routes []conf_v1.Route) *conf_v1.VirtualServer {
	vs := createTestVirtualServer(name, host)
	vs.Spec.Routes = routes
	return vs
}

func createTestVirtualServerRoute(name string, host string, path string) *conf_v1.VirtualServerRoute {
	return &conf_v1.VirtualServerRoute{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      name,
		},
		Spec: conf_v1.VirtualServerRouteSpec{
			IngressClass: "nginx",
			Host:         host,
			Subroutes: []conf_v1.Route{
				{
					Path: path,
					Action: &conf_v1.Action{
						Return: &conf_v1.ActionReturn{
							Body: "vsr",
						},
					},
				},
			},
		},
	}
}

func TestChooseObjectMetaWinner(t *testing.T) {
	now := metav1.Now()
	afterNow := metav1.NewTime(now.Add(1 * time.Second))

	tests := []struct {
		meta1    *metav1.ObjectMeta
		meta2    *metav1.ObjectMeta
		msg      string
		expected bool
	}{
		{
			meta1: &metav1.ObjectMeta{
				UID:               "a",
				CreationTimestamp: now,
			},
			meta2: &metav1.ObjectMeta{
				UID:               "b",
				CreationTimestamp: afterNow,
			},
			msg:      "first is older",
			expected: true,
		},
		{
			meta1: &metav1.ObjectMeta{
				UID:               "a",
				CreationTimestamp: afterNow,
			},
			meta2: &metav1.ObjectMeta{
				UID:               "b",
				CreationTimestamp: now,
			},
			msg:      "second is older",
			expected: false,
		},
		{
			meta1: &metav1.ObjectMeta{
				UID:               "a",
				CreationTimestamp: now,
			},
			meta2: &metav1.ObjectMeta{
				UID:               "b",
				CreationTimestamp: now,
			},
			msg:      "both not older, but second wins",
			expected: false,
		},
	}

	for _, test := range tests {
		result := chooseObjectMetaWinner(test.meta1, test.meta2)
		if result != test.expected {
			t.Errorf("chooseObjectMetaWinner() returned %v but expected %v for the case %s", result, test.expected, test.msg)
		}
	}
}

func TestSquashResourceChanges(t *testing.T) {
	fullIngress := &FullIngress{
		Ingress: createTestIngress("test", "foo.example.com"),
	}

	fullVS := &FullVirtualServer{
		VirtualServer: createTestVirtualServer("test", "bar.example.com"),
	}

	tests := []struct {
		changes  []ResourceChange
		expected []ResourceChange
		msg      string
	}{
		{
			changes: []ResourceChange{
				{
					Op:       Delete,
					Resource: fullIngress,
				},
				{
					Op:       Delete,
					Resource: fullIngress,
				},
			},
			expected: []ResourceChange{
				{
					Op:       Delete,
					Resource: fullIngress,
				},
			},
			msg: "squash deletes",
		},
		{
			changes: []ResourceChange{
				{
					Op:       AddOrUpdate,
					Resource: fullIngress,
				},
				{
					Op:       AddOrUpdate,
					Resource: fullIngress,
				},
			},
			expected: []ResourceChange{
				{
					Op:       AddOrUpdate,
					Resource: fullIngress,
				},
			},
			msg: "squash updates",
		},
		{
			changes: []ResourceChange{
				{
					Op:       Delete,
					Resource: fullIngress,
				},
				{
					Op:       AddOrUpdate,
					Resource: fullIngress,
				},
			},
			expected: []ResourceChange{
				{
					Op:       AddOrUpdate,
					Resource: fullIngress,
				},
			},
			msg: "squash update and delete",
		},
		{
			changes: []ResourceChange{
				{
					Op:       Delete,
					Resource: fullVS,
				},
				{
					Op:       AddOrUpdate,
					Resource: fullIngress,
				},
			},
			expected: []ResourceChange{
				{
					Op:       Delete,
					Resource: fullVS,
				},
				{
					Op:       AddOrUpdate,
					Resource: fullIngress,
				},
			},
			msg: "preserve the order",
		},
		{
			changes: []ResourceChange{
				{
					Op:       Delete,
					Resource: fullIngress,
				},
				{
					Op:       AddOrUpdate,
					Resource: fullVS,
				},
			},
			expected: []ResourceChange{
				{
					Op:       Delete,
					Resource: fullIngress,
				},
				{
					Op:       AddOrUpdate,
					Resource: fullVS,
				},
			},
			msg: "do not squash different resource with same ns/name",
		},
		{
			changes: []ResourceChange{
				{
					Op:       Delete,
					Resource: fullIngress,
				},
				{
					Op:       AddOrUpdate,
					Resource: fullIngress,
				},
				{
					Op:       Delete,
					Resource: fullVS,
				},
			},
			expected: []ResourceChange{
				{
					Op:       Delete,
					Resource: fullVS,
				},
				{
					Op:       AddOrUpdate,
					Resource: fullIngress,
				},
			},
			msg: "squashed delete and update must follow delete",
		},
	}

	for _, test := range tests {
		result := squashResourceChanges(test.changes)
		if diff := cmp.Diff(test.expected, result); diff != "" {
			t.Errorf("squashResourceChanges() returned unexpected result for the case of %s (-want +got):\n%s", test.msg, diff)
		}
	}
}

type testReferenceChecker struct {
	resourceName            string
	resourceNamespace       string
	onlyIngresses           bool
	onlyMinions             bool
	onlyVirtualServers      bool
	onlyVirtualServerRoutes bool
}

func (rc *testReferenceChecker) IsReferencedByIngress(namespace string, name string, ing *networking.Ingress) bool {
	return rc.onlyIngresses && namespace == rc.resourceNamespace && name == rc.resourceName
}

func (rc *testReferenceChecker) IsReferencedByMinion(namespace string, name string, ing *networking.Ingress) bool {
	return rc.onlyMinions && namespace == rc.resourceNamespace && name == rc.resourceName
}

func (rc *testReferenceChecker) IsReferencedByVirtualServer(namespace string, name string, vs *conf_v1.VirtualServer) bool {
	return rc.onlyVirtualServers && namespace == rc.resourceNamespace && name == rc.resourceName
}

func (rc *testReferenceChecker) IsReferencedByVirtualServerRoute(namespace string, name string, vsr *conf_v1.VirtualServerRoute) bool {
	return rc.onlyVirtualServerRoutes && namespace == rc.resourceNamespace && name == rc.resourceName
}

func TestFindResourcesForResourceReference(t *testing.T) {
	regularIng := createTestIngress("regular-ingress", "foo.example.com")
	master := createTestIngressMaster("master-ingress", "bar.example.com")
	minion := createTestIngressMinion("minion-ingress", "bar.example.com", "/")
	vs := createTestVirtualServer("virtualserver-1", "qwe.example.com")
	vsWithVSR := createTestVirtualServerWithRoutes(
		"virtualserver-2",
		"asd.example.com",
		[]conf_v1.Route{
			{
				Path:  "/",
				Route: "virtualserverroute",
			},
		})
	vsr := createTestVirtualServerRoute("virtualserverroute", "asd.example.com", "/")

	configuration := createTestConfiguration()

	configuration.AddOrUpdateIngress(regularIng)
	configuration.AddOrUpdateIngress(master)
	configuration.AddOrUpdateIngress(minion)
	configuration.AddOrUpdateVirtualServer(vs)
	configuration.AddOrUpdateVirtualServer(vsWithVSR)
	configuration.AddOrUpdateVirtualServerRoute(vsr)

	tests := []struct {
		rc       resourceReferenceChecker
		expected []Resource
		msg      string
	}{
		{
			rc: &testReferenceChecker{
				resourceNamespace: "default",
				resourceName:      "test",
				onlyIngresses:     true,
			},
			expected: []Resource{
				configuration.hosts["bar.example.com"],
				configuration.hosts["foo.example.com"],
			},
			msg: "only Ingresses",
		},
		{
			rc: &testReferenceChecker{
				resourceNamespace: "default",
				resourceName:      "test",
				onlyMinions:       true,
			},
			expected: []Resource{
				configuration.hosts["bar.example.com"],
			},
			msg: "only Minions",
		},
		{
			rc: &testReferenceChecker{
				resourceNamespace:  "default",
				resourceName:       "test",
				onlyVirtualServers: true,
			},
			expected: []Resource{
				configuration.hosts["asd.example.com"],
				configuration.hosts["qwe.example.com"],
			},
			msg: "only VirtualServers",
		},
		{
			rc: &testReferenceChecker{
				resourceNamespace:       "default",
				resourceName:            "test",
				onlyVirtualServerRoutes: true,
			},
			expected: []Resource{
				configuration.hosts["asd.example.com"],
			},
			msg: "only VirtualServerRoutes",
		},
	}

	for _, test := range tests {
		result := configuration.findResourcesForResourceReference("default", "test", test.rc)
		if diff := cmp.Diff(test.expected, result); diff != "" {
			t.Errorf("findResourcesForResourceReference() returned unexpected result for the case of %s (-want +got):\n%s", test.msg, diff)
		}

		var noResources []Resource

		result = configuration.findResourcesForResourceReference("default", "wrong", test.rc)
		if diff := cmp.Diff(noResources, result); diff != "" {
			t.Errorf("findResourcesForResourceReference() returned unexpected result for the case of %s and wrong name (-want +got):\n%s", test.msg, diff)
		}

		result = configuration.findResourcesForResourceReference("wrong", "test", test.rc)
		if diff := cmp.Diff(noResources, result); diff != "" {
			t.Errorf("findResourcesForResourceReference() returned unexpected result for the case of %s and wrong namespace (-want +got):\n%s", test.msg, diff)
		}
	}
}

func TestGetResources(t *testing.T) {
	ing := createTestIngress("ingress", "foo.example.com", "bar.example.com")
	vs := createTestVirtualServer("virtualserver", "qwe.example.com")

	configuration := createTestConfiguration()
	configuration.AddOrUpdateIngress(ing)
	configuration.AddOrUpdateVirtualServer(vs)

	expected := []Resource{
		configuration.hosts["foo.example.com"],
		configuration.hosts["qwe.example.com"],
	}

	result := configuration.GetResources()
	if diff := cmp.Diff(expected, result); diff != "" {
		t.Errorf("GetResources() returned unexpected result (-want +got):\n%s", diff)
	}

	expected = []Resource{
		configuration.hosts["foo.example.com"],
	}

	result = configuration.GetResourcesWithFilter(resourceFilter{Ingresses: true})
	if diff := cmp.Diff(expected, result); diff != "" {
		t.Errorf("GetResources() returned unexpected result (-want +got):\n%s", diff)
	}

	expected = []Resource{
		configuration.hosts["qwe.example.com"],
	}

	result = configuration.GetResourcesWithFilter(resourceFilter{VirtualServers: true})
	if diff := cmp.Diff(expected, result); diff != "" {
		t.Errorf("GetResources() returned unexpected result (-want +got):\n%s", diff)
	}
}

func TestIsEqualForFullIngresses(t *testing.T) {
	regularIng := createTestIngress("regular-ingress", "foo.example.com")

	fullIngWithInvalidHost := NewRegularFullIngress(regularIng)
	fullIngWithInvalidHost.ValidHosts["foo.example.com"] = false

	fullIngWithUpdatedIsMaster := NewRegularFullIngress(regularIng)
	fullIngWithUpdatedIsMaster.IsMaster = true

	regularIngWithUpdatedGen := regularIng.DeepCopy()
	regularIngWithUpdatedGen.Generation++

	regularIngWithUpdatedAnnotations := regularIng.DeepCopy()
	regularIngWithUpdatedAnnotations.Annotations["new"] = "value"

	masterIng := createTestIngressMaster("master-ingress", "bar.example.com")
	minionIng := createTestIngressMinion("minion-ingress", "bar.example.com", "/")

	minionIngWithUpdatedGen := minionIng.DeepCopy()
	minionIngWithUpdatedGen.Generation++

	minionIngWithUpdatedAnnotations := minionIng.DeepCopy()
	minionIngWithUpdatedAnnotations.Annotations["new"] = "value"

	tests := []struct {
		fullIng1 *FullIngress
		fullIng2 *FullIngress
		expected bool
		msg      string
	}{
		{
			fullIng1: NewRegularFullIngress(regularIng),
			fullIng2: NewRegularFullIngress(regularIng),
			expected: true,
			msg:      "equal regular ingresses",
		},
		{
			fullIng1: NewRegularFullIngress(regularIng),
			fullIng2: fullIngWithInvalidHost,
			expected: false,
			msg:      "regular ingresses with different valid hosts",
		},
		{
			fullIng1: NewRegularFullIngress(regularIng),
			fullIng2: fullIngWithUpdatedIsMaster,
			expected: false,
			msg:      "regular ingresses with different IsMaster value",
		},
		{
			fullIng1: NewRegularFullIngress(regularIng),
			fullIng2: NewRegularFullIngress(regularIngWithUpdatedGen),
			expected: false,
			msg:      "regular ingresses with different generation",
		},
		{
			fullIng1: NewRegularFullIngress(regularIng),
			fullIng2: NewRegularFullIngress(regularIngWithUpdatedAnnotations),
			expected: false,
			msg:      "regular ingresses with different annotations",
		},
		{
			fullIng1: NewMasterFullIngress(masterIng, []*FullMinion{NewFullMinion(minionIng)}, map[string][]string{}),
			fullIng2: NewMasterFullIngress(masterIng, []*FullMinion{NewFullMinion(minionIng)}, map[string][]string{}),
			expected: true,
			msg:      "equal master ingresses",
		},
		{
			fullIng1: NewMasterFullIngress(masterIng, []*FullMinion{NewFullMinion(minionIng)}, map[string][]string{}),
			fullIng2: NewMasterFullIngress(masterIng, []*FullMinion{}, map[string][]string{}),
			expected: false,
			msg:      "masters with different number of minions",
		},
		{
			fullIng1: NewMasterFullIngress(masterIng, []*FullMinion{NewFullMinion(minionIng)}, map[string][]string{}),
			fullIng2: NewMasterFullIngress(masterIng, []*FullMinion{NewFullMinion(minionIngWithUpdatedGen)}, map[string][]string{}),
			expected: false,
			msg:      "masters with minions with different generation",
		},
		{
			fullIng1: NewMasterFullIngress(masterIng, []*FullMinion{NewFullMinion(minionIng)}, map[string][]string{}),
			fullIng2: NewMasterFullIngress(masterIng, []*FullMinion{NewFullMinion(minionIngWithUpdatedAnnotations)}, map[string][]string{}),
			expected: false,
			msg:      "masters with minions with different annotations",
		},
	}

	for _, test := range tests {
		result := test.fullIng1.IsEqual(test.fullIng2)
		if result != test.expected {
			t.Errorf("IsEqual() returned %v but expected %v for the case of %s", result, test.expected, test.msg)
		}
	}
}

func TestIsEqualForVirtualServers(t *testing.T) {
	vs := createTestVirtualServerWithRoutes(
		"virtualserver",
		"foo.example.com",
		[]conf_v1.Route{
			{
				Path:  "/",
				Route: "virtualserverroute",
			},
		})
	vsr := createTestVirtualServerRoute("virtualserverroute", "foo.example.com", "/")

	vsWithUpdatedGen := vs.DeepCopy()
	vsWithUpdatedGen.Generation++

	vsrWithUpdatedGen := vsr.DeepCopy()
	vsrWithUpdatedGen.Generation++

	tests := []struct {
		fullVS1  *FullVirtualServer
		fullVS2  *FullVirtualServer
		expected bool
		msg      string
	}{
		{
			fullVS1:  NewFullVirtualServer(vs, []*conf_v1.VirtualServerRoute{vsr}, []string{}),
			fullVS2:  NewFullVirtualServer(vs, []*conf_v1.VirtualServerRoute{vsr}, []string{}),
			expected: true,
			msg:      "equal virtual servers",
		},
		{
			fullVS1:  NewFullVirtualServer(vs, []*conf_v1.VirtualServerRoute{vsr}, []string{}),
			fullVS2:  NewFullVirtualServer(vsWithUpdatedGen, []*conf_v1.VirtualServerRoute{vsr}, []string{}),
			expected: false,
			msg:      "virtual servers with different generation",
		},
		{
			fullVS1:  NewFullVirtualServer(vs, []*conf_v1.VirtualServerRoute{vsr}, []string{}),
			fullVS2:  NewFullVirtualServer(vs, []*conf_v1.VirtualServerRoute{}, []string{}),
			expected: false,
			msg:      "virtual servers with different number of virtual server routes",
		},
		{
			fullVS1:  NewFullVirtualServer(vs, []*conf_v1.VirtualServerRoute{vsr}, []string{}),
			fullVS2:  NewFullVirtualServer(vs, []*conf_v1.VirtualServerRoute{vsrWithUpdatedGen}, []string{}),
			expected: false,
			msg:      "virtual servers with virtual server routes with different generation",
		},
	}

	for _, test := range tests {
		result := test.fullVS1.IsEqual(test.fullVS2)
		if result != test.expected {
			t.Errorf("IsEqual() returned %v but expected %v for the case of %s", result, test.expected, test.msg)
		}
	}
}

func TestIsEqualForDifferentResources(t *testing.T) {
	fullIng := NewRegularFullIngress(createTestIngress("ingress", "foo.example.com"))
	fullVS := NewFullVirtualServer(createTestVirtualServer("virtualserver", "bar.example.com"), []*conf_v1.VirtualServerRoute{}, []string{})

	result := fullIng.IsEqual(fullVS)
	if result != false {
		t.Error("IsEqual() for different resources returned true but expected false")
	}
}

func TestCompareConfigurationProblems(t *testing.T) {
	tests := []struct {
		problem1 *ConfigurationProblem
		problem2 *ConfigurationProblem
		expected bool
		msg      string
	}{
		{
			problem1: &ConfigurationProblem{
				IsError: false,
				Reason:  "reason",
				Message: "message",
			},
			problem2: &ConfigurationProblem{
				IsError: false,
				Reason:  "reason",
				Message: "message",
			},
			expected: true,
			msg:      "equal problems",
		},
		{
			problem1: &ConfigurationProblem{
				Object:  createTestIngress("ingress-1", "foo.example.com"),
				IsError: false,
				Reason:  "reason",
				Message: "message",
			},
			problem2: &ConfigurationProblem{
				Object:  createTestIngress("ingress-2", "bar.example.com"),
				IsError: false,
				Reason:  "reason",
				Message: "message",
			},
			expected: true,
			msg:      "equal problems although objects are different",
		},
		{
			problem1: &ConfigurationProblem{
				IsError: true,
				Reason:  "reason",
				Message: "message",
			},
			problem2: &ConfigurationProblem{
				IsError: false,
				Reason:  "reason",
				Message: "message",
			},
			expected: false,
			msg:      "different isError",
		},
		{
			problem1: &ConfigurationProblem{
				IsError: false,
				Reason:  "reason",
				Message: "message",
			},
			problem2: &ConfigurationProblem{
				IsError: false,
				Reason:  "another reason",
				Message: "message",
			},
			expected: false,
			msg:      "different Reason",
		},
		{
			problem1: &ConfigurationProblem{
				IsError: false,
				Reason:  "reason",
				Message: "message",
			},
			problem2: &ConfigurationProblem{
				IsError: false,
				Reason:  "reason",
				Message: "another message",
			},
			expected: false,
			msg:      "different Message",
		},
	}

	for _, test := range tests {
		result := compareConfigurationProblems(test.problem1, test.problem2)
		if result != test.expected {
			t.Errorf("compareConfigurationProblems() returned %v but expected %v for the case of %s", result, test.expected, test.msg)
		}
	}
}
