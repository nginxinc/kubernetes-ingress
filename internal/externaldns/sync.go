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
	"k8s.io/utils/net"

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

		// verify if external endpoints are valid IP addresses here
		if err := validateExternalEndpoints(vs); err != nil {
			glog.Error("Invalid external enpoint")
			rec.Eventf(vs, corev1.EventTypeWarning, reasonBadConfig, "Invalid external endpoint")
			return err
		}

		// check if it is IP or hostname or both

		// External IP or Hostname (LoadBalancer!!!)
		// implement validation for IP v4, v6, and hostname
		// if neither return error

		endpoints := vs.Status.ExternalEndpoints

		/*
			1) retrieve hostname from VS
			2) retrieve external IP of the VS (enable VS status should be done at the moment of the start ingress) - check it and bail if not enabled

			3) ret. ExternalDNS data (record)

		*/

		// Step 1
		// get info about configured externaldns from the VS (?)

		// (validation needed at this point?) vsHost := vs.Spec.Host
		vs.Status.ExternalEndpoints
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
			_, err = client.ExternaldnsV1().DNSEndpoints(newDNSEndpoint.Namespace).Create(ctx, newDNSEndpoint, metav1.CreateOptions{})
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

// GetEndpointAndRecordType takes an ExternalEndpoint and returns a string
// representing IP or Hostname, and a corresponding DNS record type.
// It returns an error if either IP or Hostname are invalid.
//
// NOTE: Check what validation we need to perform inside the func.
// And what should the func return if both IP and Hostname are present
// and valid (in vsapi.ExternalEndpoint). Current draft assumes that
// we pass vsapi.ExternalEndpoint with either IP or Hostname.
// Also, possible this func will be unexported when used as a part of
// larger func with comprehensive test coverage. As of now is exported
// to be possible to import it for testing.
func GetEndpointAndRecordType(vs vsapi.ExternalEndpoint) (string, string, error) {
	if vs.IP == "" && vs.Hostname == "" {
		return "", "", errors.New("ip and hostname not provided")
	}
	var endpoint, record string
	// ExternalEndpoint has an IP address
	if vs.IP != "" {
		if errMsg := validation.IsValidIP(vs.IP); len(errMsg) > 0 {
			return "", "", fmt.Errorf("invalid IP address %s, %s", vs.IP, strings.Join(errMsg, ", "))
		}
		if net.IsIPv6String(vs.IP) {
			endpoint, record = vs.IP, "AAAA"
		} else {
			endpoint, record = vs.IP, "A"
		}
		return endpoint, record, nil
	}

	// ExternalEnpoint has a hostname
	if vs.Hostname != "" {
		if errMsg := validation.IsQualifiedName(vs.Hostname); len(errMsg) > 0 {
			return "", "", fmt.Errorf("invalid hostname %s, %s", vs.Hostname, strings.Join(errMsg, ", "))
		}
		endpoint, record = vs.Hostname, "CNAME"
	}

	return endpoint, record, nil
}

// DNS endpoint reconciler

func findDNSExternalEndpointsToBeRemoved(endpoints []*extdnsapi.DNSEndpoint) []string {
	var toBeRemoved []string
	for _, e := range endpoints {
		owner := metav1.GetControllerOf(e)
		if owner == nil {
			continue
		}
		if owner.Kind != vsGVK.Kind {
			continue
		}

		// owner name to get
		vsThatOwnsEnpoint, err := vsLister.VirtualServers(e.Namespace).Get(owner.Name)
		if err != nil {
			// delete endpoint
			// add to the slist to be removed
		}
	}
	return toBeRemoved
}

func validateExternalEndpoints(vs *vsapi.VirtualServer) error {
	for _, e := range vs.Status.ExternalEndpoints {
		if e.IP == "" && e.Hostname == "" {
			return fmt.Error("missing hostname and IP address")
		}
		if e.IP != "" {
			if errMsg := validators.IsValidIP(e.IP); len(errMsg) > 0 {
				return fmt.Errorf("invalid external endpoint IP address %s, %s", e.IP, strings.Join(errMsg, ", "))
		}
		if e.Hostname != "" {
			if errMsg := validation.IsQualifiedName(e.Hostname); len(errMsg) > 0 {
				return fmt.Errorf("invalid external endpoint hostname %s, %s", e.Hostname, strings.Join(errMsg, ", "))
			}
		}
	}
	return nil
}

func buildDNSEndpoint(extdnsLister extdnslisters.DNSEndpointLister, vs *vsapi.VirtualServer) (newDNSEndpoint, updateDNSEndpoint *extdnsapi.DNSEndpoint, _ error) {
	// Get existing DNSEndpoint
	existingDNSEndpoint, err := extdnsLister.DNSEndpoints(vs.Namespace).Get(vs.Spec.Host)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, nil, err
		}
	}

	var controllerGVK schema.GroupVersionKind = vsGVK

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
					DNSName: vs.Spec.Host,
					Targets: targets,

					// RecordType to be computed based on hostname or IP ONLY
					//  if user did not specify it!!!
					RecordType: vs.Spec.ExternalDNS.RecordType,

					// Update unit tests for TTL
					// do not add this if recordttl is not provided (check for nil)
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
