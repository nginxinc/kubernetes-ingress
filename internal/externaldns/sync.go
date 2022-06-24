package externaldns

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang/glog"
	"github.com/google/go-cmp/cmp"
	vsapi "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"
	extdnsapi "github.com/nginxinc/kubernetes-ingress/pkg/apis/externaldns/v1"
	clientset "github.com/nginxinc/kubernetes-ingress/pkg/client/clientset/versioned"
	extdnslisters "github.com/nginxinc/kubernetes-ingress/pkg/client/listers/externaldns/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation"
	validators "k8s.io/apimachinery/pkg/util/validation"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
)

const (
	reasonBadConfig         = "BadConfig"
	reasonCreateExternalDNS = "CreateExternalDNS"
	reasonUpdateExternalDNS = "UpdateExternalDNS"
	reasonDeleteExternalDNS = "DeleteExternalDNS"
)

var vsGVK = vsapi.SchemeGroupVersion.WithKind("VirtualServer")

// SyncFn is the reconciliation function passed to externaldns controller.
type SyncFn func(context.Context, *vsapi.VirtualServer) error

// SyncFnFor knows how to reconcile VirtualServer ExternalDNS object.
func SyncFnFor(rec record.EventRecorder, extdnsClient clientset.Interface, extdnsLister extdnslisters.DNSEndpointLister) SyncFn {
	return func(ctx context.Context, vs *vsapi.VirtualServer) error {
		// Do nothing if ExternalDNS is not present in VS or is not enabled.
		if &vs.Spec.ExternalDNS == nil || !vs.Spec.ExternalDNS.Enable {
			return nil
		}

		// Logic covered in docs here:
		// https://docs.nginx.com/nginx-ingress-controller/configuration/global-configuration/reporting-resources-status/#virtualserver-and-virtualserverroute-resources
		if vs.Status.ExternalEndpoints == nil {
			glog.Error("Failed to determine external endpoints")
			rec.Eventf(vs, corev1.EventTypeWarning, reasonBadConfig, "Could not determine external endpoints")
			return fmt.Errorf("Failed to determine external endpoints")
		}

		/*
			1) retrieve hostname from VS
			2) retrieve external IP of the VS (enable VS status should be done at the moment of the start ingress) - check it and bail if not enabled

			3) ret. ExternalDNS data (record)

		*/

		// Step 1
		// get info about configured externaldns from the VS (?)

		// (validation needed at this point?) vsHost := vs.Spec.Host

		// verify if external endpoints are valid IP addresses here
		if err := validateExternalEndpoints(vs); err != nil {
			glog.Error("Invalid external enpoint")
			rec.Eventf(vs, corev1.EventTypeWarning, reasonBadConfig, "Invalid external endpoint")
			return err
		}

		// Step 2
		// build dnsendpoint top level struct from data retrieved from VS

		newDNSEndpoint, updateDNSEndpoint, err := buildDNSEndpoint(extdnsLister, vs)
		if err != nil {
			glog.Errorf("error message here %s", err)
			rec.Eventf(vs, corev1.EventTypeWarning, reasonBadConfig, "Incorrect DNSEndpoint config for VirtualServer resource: %s", err)
			return err
		}

		// Create new ExternaDNS endpoint
		if newDNSEndpoint != nil {
			_, err = extdnsClient.ExternaldnsV1().DNSEndpoints(newDNSEndpoint.Namespace).Create(ctx, newDNSEndpoint, metav1.CreateOptions{})
			if err != nil {
				glog.Errorf("Error creating ExternalDNS for VirtualServer resource: %v", err)
				rec.Eventf(vs, corev1.EventTypeWarning, reasonBadConfig, "Error creating ExternalDNS for VirtualServer resource %s", err)
				return err
			}
			rec.Eventf(vs, corev1.EventTypeNormal, reasonCreateExternalDNS, "Successfully created ExternalDNS %s", newDNSEndpoint.Name)
		}
		// Step 3
		// Update existing ExternalDNS endpoints
		if updateDNSEndpoint != nil {
			_, err = extdnsClient.ExternaldnsV1().DNSEndpoints(updateDNSEndpoint.Namespace).Update(ctx, updateDNSEndpoint, metav1.UpdateOptions{})
			if err != nil {
				glog.Errorf("Error updating ExternalDNS endpoint for VirtualServer resource: %v", err)
				rec.Eventf(vs, corev1.EventTypeWarning, reasonBadConfig, "Error updating ExternalDNS endpoint for VirtualServer resource: %s", err)
				return err
			}
			rec.Eventf(vs, corev1.EventTypeNormal, reasonUpdateExternalDNS, "Successfully updated ExternalDNS %q", updateDNSEndpoint.Name)
		}

		// Step 4
		// list dns entries
		extdnsentries, err := extdnsLister.DNSEndpoints(vs.GetNamespace()).List(labels.Everything())
		if err != nil {
			return err
		}
		unrequiredExtDNSNames := findDNSExternalEndpointsToBeRemoved(extdnsentries, vs)

		for _, name := range unrequiredExtDNSNames {
			err := extdnsClient.ExternaldnsV1().DNSEndpoints(vs.GetNamespace()).Delete(ctx, name, metav1.DeleteOptions{})
			if err != nil {
				glog.Errorf("Error deleting ExternalDNS for VirtualServer resource: %v", err)
				return err
			}
			rec.Eventf(vs, corev1.EventTypeNormal, reasonDeleteExternalDNS, "Successfully deleted unrequired ExternalDNS endpoint %q", name)
		}

		return nil
	}
}

func findDNSExternalEndpointsToBeRemoved(endpoints []*extdnsapi.DNSEndpoint, vs *vsapi.VirtualServer) []string {
	var toBeRemoved []string
	for _, e := range endpoints {
		if !metav1.IsControlledBy(e, vs) {
			continue
		}
		if !extDNSNameUsedIn(e.ObjectMeta.Name, *vs) {
			toBeRemoved = append(toBeRemoved, e.ObjectMeta.Name)
		}
	}
	return toBeRemoved
}

func extDNSNameUsedIn(endpointName string, vs vsapi.VirtualServer) bool {
	return endpointName == vs.Spec.Host
}

func validateExternalEndpoints(vs *vsapi.VirtualServer) error {
	for _, e := range vs.Status.ExternalEndpoints {
		if errMsg := validators.IsValidIP(e.IP); len(errMsg) > 0 {
			return fmt.Errorf("invalid external endpoint: %s, %s", e.IP, strings.Join(errMsg, ", "))
		}
	}
	return nil
}

func buildDNSEndpoint(extdnsLister extdnslisters.DNSEndpointLister, vs *vsapi.VirtualServer) (newDNSEndpoint, updateDNSEndpoint *extdnsapi.DNSEndpoint, _ error) {
	// Get existing DNSEndpoint
	existingDNSEndpoint, err := extdnsLister.DNSEndpoints(vs.Namespace).Get(vs.Spec.Host)
	if apierrors.IsNotFound(err) && err != nil {
		return nil, nil, err
	}

	var controllerGVK schema.GroupVersionKind = vsGVK

	if err := isFullyQualifiedDomainName(vs.Spec.Host); err != nil {
		return nil, nil, err
	}

	targets, err := buildTargets(vs)
	if err != nil {
		return nil, nil, err
	}

	labels := buildLabels(vs)
	providerSpecific := buildProviderSpecificProperties(vs)

	dnsEndpoint := &extdnsapi.DNSEndpoint{
		ObjectMeta: metav1.ObjectMeta{
			Name:            vs.Spec.Host,
			Namespace:       vs.Namespace,
			Labels:          vs.Labels,
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(vs, controllerGVK)},
		},
		Spec: extdnsapi.DNSEndpointSpec{
			Endpoints: []*extdnsapi.Endpoint{
				{
					DNSName:          vs.Spec.Host,
					Targets:          targets,
					RecordType:       vs.Spec.ExternalDNS.RecordType,
					RecordTTL:        extdnsapi.TTL(vs.Spec.ExternalDNS.RecordTTL),
					Labels:           labels,
					ProviderSpecific: providerSpecific,
				},
			},
		},
	}

	vs = vs.DeepCopy()

	if existingDNSEndpoint != nil {
		glog.V(3).Infof("DNDEndpoint already exist for this object, ensuring it is up to date")
		if metav1.GetControllerOf(existingDNSEndpoint) == nil {
			glog.V(3).Infof("DNSEndpoint has no owner. refusing to update non-owned resource")
			return nil, nil, nil
		}
		if !metav1.IsControlledBy(existingDNSEndpoint, vs) {
			glog.V(3).Infof("external DNS endpoint resource isnot owned by this object. refusing to update non-owned resource")
			return nil, nil, nil
		}
		if !extdnsendpointNeedsUpdate(existingDNSEndpoint, dnsEndpoint) {
			glog.V(3).Infof("external DNS resource is already up to date for object")
			return nil, nil, nil
		}

		updateDNSEndpoint := existingDNSEndpoint.DeepCopy()
		updateDNSEndpoint.Spec = dnsEndpoint.Spec
		updateDNSEndpoint.Labels = dnsEndpoint.Labels
		updateDNSEndpoint.Name = dnsEndpoint.Name
	} else {
		newDNSEndpoint = dnsEndpoint
	}
	return newDNSEndpoint, updateDNSEndpoint, nil
}

func buildTargets(vs *vsapi.VirtualServer) (extdnsapi.Targets, error) {
	var targets extdnsapi.Targets
	for _, target := range vs.Status.ExternalEndpoints {
		if errMsg := validators.IsValidIP(target.IP); len(errMsg) > 0 {
			return nil, fmt.Errorf("%s", strings.Join(errMsg, ", "))
		}
		targets = append(targets, target.IP)
	}
	return targets, nil
}

func buildLabels(vs *vsapi.VirtualServer) extdnsapi.Labels {
	labels := make(extdnsapi.Labels)
	for k, v := range vs.Spec.ExternalDNS.Labels {
		labels[k] = v
	}
	return labels
}

func buildProviderSpecificProperties(vs *vsapi.VirtualServer) extdnsapi.ProviderSpecific {
	var providerSpecific extdnsapi.ProviderSpecific
	for _, pspecific := range vs.Spec.ExternalDNS.ProviderSpecific {
		p := extdnsapi.ProviderSpecificProperty{
			Name:  pspecific.Name,
			Value: pspecific.Value,
		}
		providerSpecific = append(providerSpecific, p)
	}
	return providerSpecific
}

func extdnsendpointNeedsUpdate(dnsA, dnsB *extdnsapi.DNSEndpoint) bool {
	if !cmp.Equal(dnsA.ObjectMeta.Name, dnsB.ObjectMeta.Name) {
		return true
	}
	if !cmp.Equal(dnsA.Labels, dnsB.Labels) {
		return true
	}
	if !cmp.Equal(dnsA.Spec, dnsB.Spec) {
		return true
	}
	return false
}

// isFullyQualifiedDomainName checks if the domain name is fully qualified.
// It requires a minimum of 2 segments and accepts a trailing . as valid.
func isFullyQualifiedDomainName(name string) error {
	if len(name) == 0 {
		return fmt.Errorf("name not provided")
	}
	name = strings.TrimSuffix(name, ".")
	if issues := validation.IsDNS1123Subdomain(name); len(issues) > 0 {
		return fmt.Errorf("name %s is not valid subdomain, %s", name, strings.Join(issues, ", "))
	}
	if len(strings.Split(name, ".")) < 2 {
		return fmt.Errorf("name %s should be a domain with at least two segments separated by dots", name)
	}
	for _, label := range strings.Split(name, ".") {
		if issues := validation.IsDNS1123Label(label); len(issues) > 0 {
			return fmt.Errorf("label %s should conform to the definition of label in DNS (RFC1123), %s", label, strings.Join(issues, ", "))
		}
	}
	return nil
}
