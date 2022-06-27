package externaldns

import (
	"context"
	"errors"
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
func SyncFnFor(rec record.EventRecorder, client clientset.Interface, extdnsLister extdnslisters.DNSEndpointLister) SyncFn {
	return func(ctx context.Context, vs *vsapi.VirtualServer) error {
		// Do nothing if ExternalDNS is not present (nil) in VS or is not enabled.
		if !vs.Spec.ExternalDNS.Enable {
			return nil
		}

		if vs.Status.ExternalEndpoints == nil {
			glog.Error("Failed to determine external endpoints")
			rec.Eventf(vs, corev1.EventTypeWarning, reasonBadConfig, "Could not determine external endpoints")
			return fmt.Errorf("failed to determine external endpoints")
		}

		targets, err := getValidTargets(vs.Status.ExternalEndpoints)
		if err != nil {
			glog.Error("Invalid external enpoint")
			rec.Eventf(vs, corev1.EventTypeWarning, reasonBadConfig, "Invalid external endpoint")
			return err
		}

		newDNSEndpoint, updateDNSEndpoint, err := buildDNSEndpoint(extdnsLister, vs, targets)
		if err != nil {
			glog.Errorf("error message here %s", err)
			rec.Eventf(vs, corev1.EventTypeWarning, reasonBadConfig, "Incorrect DNSEndpoint config for VirtualServer resource: %s", err)
			return err
		}

		// Create new ExternaDNS endpoint
		if newDNSEndpoint != nil {
			_, err = client.ExternaldnsV1().DNSEndpoints(newDNSEndpoint.Namespace).Create(ctx, newDNSEndpoint, metav1.CreateOptions{})
			if err != nil {
				glog.Errorf("Error creating ExternalDNS for VirtualServer resource: %v", err)
				rec.Eventf(vs, corev1.EventTypeWarning, reasonBadConfig, "Error creating ExternalDNS for VirtualServer resource %s", err)
				return err
			}
			rec.Eventf(vs, corev1.EventTypeNormal, reasonCreateExternalDNS, "Successfully created ExternalDNS %s", newDNSEndpoint.Name)
		}

		// Update existing ExternalDNS endpoints
		if updateDNSEndpoint != nil {
			_, err = client.ExternaldnsV1().DNSEndpoints(updateDNSEndpoint.Namespace).Update(ctx, updateDNSEndpoint, metav1.UpdateOptions{})
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

		// think where to put the logic of listing and deleteing dnsendpoints
		unrequiredExtDNSNames := findDNSExternalEndpointsToBeRemoved(extdnsentries)

		for _, name := range unrequiredExtDNSNames {
			err := client.ExternaldnsV1().DNSEndpoints(vs.GetNamespace()).Delete(ctx, name, metav1.DeleteOptions{})
			if err != nil {
				glog.Errorf("Error deleting ExternalDNS for VirtualServer resource: %v", err)
				return err
			}
			rec.Eventf(vs, corev1.EventTypeNormal, reasonDeleteExternalDNS, "Successfully deleted unrequired ExternalDNS endpoint %q", name)
		}
		return nil
	}
}

func getValidTargets(endpoints []vsapi.ExternalEndpoint) (extdnsapi.Targets, error) {
	var targets extdnsapi.Targets
	for _, e := range endpoints {
		if e.IP != "" {
			if errMsg := validators.IsValidIP(e.IP); len(errMsg) > 0 {
				continue
			}
			targets = append(targets, e.IP)
		}
		if e.Hostname != "" {
			if errMsg := validation.IsQualifiedName(e.Hostname); len(errMsg) > 0 {
				continue
			}
			targets = append(targets, e.Hostname)
		}
	}
	if len(targets) == 0 {
		return targets, errors.New("valid targets not defined")
	}
	return targets, nil
}

func buildDNSEndpoint(extdnsLister extdnslisters.DNSEndpointLister, vs *vsapi.VirtualServer, targets extdnsapi.Targets) (newDNSEndpoint, updateDNSEndpoint *extdnsapi.DNSEndpoint, _ error) {
	existingDNSEndpoint, err := extdnsLister.DNSEndpoints(vs.Namespace).Get(vs.Spec.Host)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, nil, err
		}
	}
	var controllerGVK schema.GroupVersionKind = vsGVK

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
					RecordType:       buildRecordType(vs),
					RecordTTL:        buildTTL(vs),
					Labels:           buildLabels(vs),
					ProviderSpecific: buildProviderSpecificProperties(vs),
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

func buildTTL(vs *vsapi.VirtualServer) extdnsapi.TTL {
	return extdnsapi.TTL(vs.Spec.ExternalDNS.RecordTTL)
}

func buildRecordType(vs *vsapi.VirtualServer) string {
	if vs.Spec.ExternalDNS.RecordType == "" {
		return "A"
	}
	return vs.Spec.ExternalDNS.RecordType
}

func buildLabels(vs *vsapi.VirtualServer) extdnsapi.Labels {
	if vs.Spec.ExternalDNS.Labels == nil {
		return nil
	}
	labels := make(extdnsapi.Labels)
	for k, v := range vs.Spec.ExternalDNS.Labels {
		labels[k] = v
	}
	return labels
}

func buildProviderSpecificProperties(vs *vsapi.VirtualServer) extdnsapi.ProviderSpecific {
	if vs.Spec.ExternalDNS.ProviderSpecific == nil {
		return nil
	}
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
