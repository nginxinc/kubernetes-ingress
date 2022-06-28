package externaldns

import (
	"context"
	"errors"
	"fmt"

	"github.com/golang/glog"
	"github.com/google/go-cmp/cmp"
	vsapi "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"
	extdnsapi "github.com/nginxinc/kubernetes-ingress/pkg/apis/externaldns/v1"
	clientset "github.com/nginxinc/kubernetes-ingress/pkg/client/clientset/versioned"
	extdnslisters "github.com/nginxinc/kubernetes-ingress/pkg/client/listers/externaldns/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	validators "k8s.io/apimachinery/pkg/util/validation"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	netutils "k8s.io/utils/net"
)

const (
	reasonBadConfig         = "BadConfig"
	reasonCreateExternalDNS = "CreateExternalDNS"
	reasonUpdateExternalDNS = "UpdateExternalDNS"
	recordTypeA             = "A"
	recordTypeAAAA          = "AAAA"
	recordTypeCNAME         = "CNAME"
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

		targets, recordType, err := getValidTargets(vs.Status.ExternalEndpoints)
		if err != nil {
			glog.Error("Invalid external enpoint")
			rec.Eventf(vs, corev1.EventTypeWarning, reasonBadConfig, "Invalid external endpoint")
			return err
		}

		newDNSEndpoint, updateDNSEndpoint, err := buildDNSEndpoint(extdnsLister, vs, targets, recordType)
		if err != nil {
			glog.Errorf("error message here %s", err)
			rec.Eventf(vs, corev1.EventTypeWarning, reasonBadConfig, "Incorrect DNSEndpoint config for VirtualServer resource: %s", err)
			return err
		}

		// Create new ExternalDNS endpoint
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
		return nil
	}
}

func getValidTargets(endpoints []vsapi.ExternalEndpoint) (extdnsapi.Targets, string, error) {
	var targets extdnsapi.Targets
	var recordType string
	var recordA bool
	var recordCNAME bool
	var recordAAAA bool
	var err error
	glog.Infof("Going through endpoints %v", endpoints)
	for _, e := range endpoints {
		if e.IP != "" {
			glog.V(3).Infof("IP is defined: %v", e.IP)
			if errMsg := validators.IsValidIP(e.IP); len(errMsg) > 0 {
				continue
			}
			ip := netutils.ParseIPSloppy(e.IP)
			if ip.To4() != nil {
				recordA = true
			} else {
				recordAAAA = true
			}
			targets = append(targets, e.IP)
		} else if e.Hostname != "" {
			glog.V(3).Infof("Hostname is defined: %v", e.Hostname)
			targets = append(targets, e.Hostname)
			recordCNAME = true
		}
	}
	if len(targets) == 0 {
		return targets, recordType, errors.New("valid targets not defined")
	}
	if recordA {
		recordType = recordTypeA
	} else if recordAAAA {
		recordType = recordTypeAAAA
	} else if recordCNAME {
		recordType = recordTypeCNAME
	} else {
		err = errors.New("recordType could not be determined")
	}
	return targets, recordType, err
}

func buildDNSEndpoint(extdnsLister extdnslisters.DNSEndpointLister, vs *vsapi.VirtualServer, targets extdnsapi.Targets, recordType string) (newDNSEndpoint, updateDNSEndpoint *extdnsapi.DNSEndpoint, _ error) {
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
					RecordType:       buildRecordType(vs, recordType),
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

func buildRecordType(vs *vsapi.VirtualServer, recordType string) string {
	if vs.Spec.ExternalDNS.RecordType == "" {
		return recordType
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
