package k8s

import (
	"context"
	"fmt"
	"net"
	"reflect"

	"github.com/golang/glog"
	"github.com/nginxinc/kubernetes-ingress/internal/configs"
	api_v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	extensionsv1beta1 "k8s.io/client-go/kubernetes/typed/extensions/v1beta1"
)

// statusUpdater reports Ingress status information via the kubernetes
// API, primarily the IP or host of the LoadBalancer Service exposing the
// Ingress Controller, or an external IP specified in the ConfigMap.
type statusUpdater struct {
	client                   kubernetes.Interface
	namespace                string
	externalServiceName      string
	externalStatusAddress    string
	externalServiceAddresses []string
	status                   []api_v1.LoadBalancerIngress
	keyFunc                  func(obj interface{}) (string, error)
	ingLister                *storeToIngressLister
}

// UpdateManagedAndMergeableIngresses handles the full return format of LoadBalancerController.getManagedIngresses
func (su *statusUpdater) UpdateManagedAndMergeableIngresses(managedIngresses []v1beta1.Ingress, mergableIngExes map[string]*configs.MergeableIngresses) error {
	ings := []v1beta1.Ingress{}
	ings = append(ings, managedIngresses...)
	for _, mergableIngEx := range mergableIngExes {
		for _, minion := range mergableIngEx.Minions {
			ings = append(ings, *minion.Ingress)
		}
	}
	return su.BulkUpdateIngressStatus(ings)
}

// UpdateMergableIngresses is a convience passthru to update Ingresses with our configs.MergableIngresses type
func (su *statusUpdater) UpdateMergableIngresses(mergableIngresses *configs.MergeableIngresses) error {
	ings := []v1beta1.Ingress{}
	ingExes := []*configs.IngressEx{}

	ingExes = append(ingExes, mergableIngresses.Master)
	ingExes = append(ingExes, mergableIngresses.Minions...)

	for _, ingEx := range ingExes {
		ings = append(ings, *ingEx.Ingress)
	}
	return su.BulkUpdateIngressStatus(ings)
}

// ClearIngressStatus clears the Ingress status.
func (su *statusUpdater) ClearIngressStatus(ing v1beta1.Ingress) error {
	return su.updateIngressWithStatus(ing, []api_v1.LoadBalancerIngress{})
}

// UpdateIngressStatus updates the status on the selected Ingress.
func (su *statusUpdater) UpdateIngressStatus(ing v1beta1.Ingress) error {
	return su.updateIngressWithStatus(ing, su.status)
}

// updateIngressWithStatus sets the provided status on the selected Ingress.
func (su *statusUpdater) updateIngressWithStatus(ing v1beta1.Ingress, status []api_v1.LoadBalancerIngress) error {
	if reflect.DeepEqual(ing.Status.LoadBalancer.Ingress, status) {
		return nil
	}

	// Get a pristine Ingress from the Store. Required because annotations can be modified
	// for mergable Ingress objects and the update status API call will update annotations, not just status.
	key, err := su.keyFunc(&ing)
	if err != nil {
		glog.V(3).Infof("error getting key for ing: %v", err)
		return err
	}
	ingCopy, exists, err := su.ingLister.GetByKeySafe(key)
	if err != nil {
		glog.V(3).Infof("error getting ing from Store by key: %v", err)
		return err
	}
	if !exists {
		glog.V(3).Infof("ing doesn't exist in Store")
		return nil
	}

	ingCopy.Status.LoadBalancer.Ingress = status
	clientIngress := su.client.ExtensionsV1beta1().Ingresses(ingCopy.Namespace)
	_, err = clientIngress.UpdateStatus(context.TODO(), ingCopy, metav1.UpdateOptions{})
	if err != nil {
		glog.V(3).Infof("error setting ingress status: %v", err)
		err = su.retryStatusUpdate(clientIngress, ingCopy)
		if err != nil {
			glog.V(3).Infof("error retrying status update: %v", err)
			return err
		}
	}
	glog.V(3).Infof("updated status for ing: %v %v", ing.Namespace, ing.Name)
	return nil
}

// BulkUpdateIngressStatus sets the status field on the selected Ingresses, specifically
// the External IP field.
func (su *statusUpdater) BulkUpdateIngressStatus(ings []v1beta1.Ingress) error {
	if len(ings) < 1 {
		glog.V(3).Info("no ingresses to update")
		return nil
	}
	failed := false
	for _, ing := range ings {
		err := su.updateIngressWithStatus(ing, su.status)
		if err != nil {
			failed = true
		}
	}
	if failed {
		return fmt.Errorf("not all Ingresses updated")
	}
	return nil
}

// retryStatusUpdate fetches a fresh copy of the Ingress from the k8s API, checks if it still needs to be
// updated, and then attempts to update. We often need to fetch fresh copies due to the
// k8s API using ResourceVersion to stop updates on stale items.
func (su *statusUpdater) retryStatusUpdate(clientIngress extensionsv1beta1.IngressInterface, ingCopy *v1beta1.Ingress) error {
	apiIng, err := clientIngress.Get(context.TODO(), ingCopy.Name, metav1.GetOptions{})
	if err != nil {
		glog.V(3).Infof("error getting ingress resource: %v", err)
		return err
	}
	if !reflect.DeepEqual(ingCopy.Status.LoadBalancer, apiIng.Status.LoadBalancer) {
		glog.V(3).Infof("retrying update status for ingress: %v, %v", ingCopy.Namespace, ingCopy.Name)
		apiIng.Status.LoadBalancer = ingCopy.Status.LoadBalancer
		_, err := clientIngress.UpdateStatus(context.TODO(), apiIng, metav1.UpdateOptions{})
		if err != nil {
			glog.V(3).Infof("update retry failed: %v", err)
		}
		return err
	}
	return nil
}

// saveStatus saves the string array of IPs or addresses that we will set as status
// on all the Ingresses that we manage.
func (su *statusUpdater) saveStatus(ips []string) {
	statusIngs := []api_v1.LoadBalancerIngress{}
	for _, ip := range ips {
		if net.ParseIP(ip) == nil {
			statusIngs = append(statusIngs, api_v1.LoadBalancerIngress{Hostname: ip})
		} else {
			statusIngs = append(statusIngs, api_v1.LoadBalancerIngress{IP: ip})
		}
	}
	su.status = statusIngs
}

func getExternalServiceAddress(svc *api_v1.Service) []string {
	addresses := []string{}
	if svc == nil {
		return addresses
	}

	if svc.Spec.Type == api_v1.ServiceTypeExternalName {
		addresses = append(addresses, svc.Spec.ExternalName)
		return addresses
	}

	for _, ip := range svc.Status.LoadBalancer.Ingress {
		if ip.IP == "" {
			addresses = append(addresses, ip.Hostname)
		} else {
			addresses = append(addresses, ip.IP)
		}
	}
	addresses = append(addresses, svc.Spec.ExternalIPs...)
	return addresses
}

// SaveStatusFromExternalStatus saves the status from a string.
// For use with the external-status-address ConfigMap setting.
// This method does not update ingress status - statusUpdater.UpdateIngressStatus must be called separately.
func (su *statusUpdater) SaveStatusFromExternalStatus(externalStatusAddress string) {
	su.externalStatusAddress = externalStatusAddress
	if externalStatusAddress == "" {
		// if external-status-address was removed from configMap, fall back on
		// external service if it exists
		if len(su.externalServiceAddresses) > 0 {
			su.saveStatus(su.externalServiceAddresses)
			return
		}
	}
	ips := []string{}
	ips = append(ips, su.externalStatusAddress)
	su.saveStatus(ips)
}

// ClearStatusFromExternalService clears the saved status from the External Service
func (su *statusUpdater) ClearStatusFromExternalService() {
	su.SaveStatusFromExternalService(nil)
}

// SaveStatusFromExternalService saves the external IP or address from the service.
// This method does not update ingress status - UpdateIngressStatus must be called separately.
func (su *statusUpdater) SaveStatusFromExternalService(svc *api_v1.Service) {
	ips := getExternalServiceAddress(svc)
	su.externalServiceAddresses = ips
	if su.externalStatusAddress != "" {
		glog.V(3).Info("skipping external service address - external-status-address is set and takes precedence")
		return
	}
	su.saveStatus(ips)
}
