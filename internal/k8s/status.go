package k8s

import (
	"context"
	"fmt"
	"net"
	"reflect"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"github.com/nginxinc/kubernetes-ingress/internal/configs"
	conf_v1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"
	v1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"
	k8s_nginx "github.com/nginxinc/kubernetes-ingress/pkg/client/clientset/versioned"
	api_v1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typednetworking "k8s.io/client-go/kubernetes/typed/networking/v1beta1"

	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

// statusUpdater reports Ingress, VirtualServer and VirtualServerRoute status information via the kubernetes
// API. For external information, it primarily reports the IP or host of the LoadBalancer Service exposing the
// Ingress Controller, or an external IP specified in the ConfigMap.
type statusUpdater struct {
	client                   kubernetes.Interface
	namespace                string
	externalServiceName      string
	externalStatusAddress    string
	externalServiceAddresses []string
	externalServicePorts     string
	externalEndpoints        []v1.ExternalEndpoint
	status                   []api_v1.LoadBalancerIngress
	keyFunc                  func(obj interface{}) (string, error)
	ingLister                *storeToIngressLister
	confClient               k8s_nginx.Interface
}

// UpdateManagedAndMergeableIngresses handles the full return format of LoadBalancerController.getManagedIngresses
func (su *statusUpdater) UpdateManagedAndMergeableIngresses(managedIngresses []networking.Ingress, mergableIngExes map[string]*configs.MergeableIngresses) error {
	ings := []networking.Ingress{}
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
	ings := []networking.Ingress{}
	ingExes := []*configs.IngressEx{}

	ingExes = append(ingExes, mergableIngresses.Master)
	ingExes = append(ingExes, mergableIngresses.Minions...)

	for _, ingEx := range ingExes {
		ings = append(ings, *ingEx.Ingress)
	}
	return su.BulkUpdateIngressStatus(ings)
}

// ClearIngressStatus clears the Ingress status.
func (su *statusUpdater) ClearIngressStatus(ing networking.Ingress) error {
	return su.updateIngressWithStatus(ing, []api_v1.LoadBalancerIngress{})
}

// UpdateIngressStatus updates the status on the selected Ingress.
func (su *statusUpdater) UpdateIngressStatus(ing networking.Ingress) error {
	return su.updateIngressWithStatus(ing, su.status)
}

// updateIngressWithStatus sets the provided status on the selected Ingress.
func (su *statusUpdater) updateIngressWithStatus(ing networking.Ingress, status []api_v1.LoadBalancerIngress) error {
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
	clientIngress := su.client.NetworkingV1beta1().Ingresses(ingCopy.Namespace)
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
func (su *statusUpdater) BulkUpdateIngressStatus(ings []networking.Ingress) error {
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
func (su *statusUpdater) retryStatusUpdate(clientIngress typednetworking.IngressInterface, ingCopy *networking.Ingress) error {
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

var intPorts = [2]int32{80, 443}
var stringPorts = [2]string{"http", "https"}

func isRequiredPort(port intstr.IntOrString) bool {
	if port.Type == intstr.Int {
		for _, p := range intPorts {
			if p == port.IntVal {
				return true
			}
		}
	} else if port.Type == intstr.String {
		for _, p := range stringPorts {
			if p == port.StrVal {
				return true
			}
		}
	}

	return false
}

func getExternalServicePorts(svc *api_v1.Service) string {
	var ports []string
	if svc == nil {
		return ""
	}

	for _, port := range svc.Spec.Ports {
		if isRequiredPort(port.TargetPort) {
			ports = append(ports, strconv.Itoa(int(port.Port)))
		}
	}

	return fmt.Sprintf("[%v]", strings.Join(ports, ","))
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
			su.externalEndpoints = su.generateExternalEndpointsFromStatus(su.status)
			return
		}
	}
	ips := []string{}
	ips = append(ips, su.externalStatusAddress)
	su.saveStatus(ips)
	su.externalEndpoints = su.generateExternalEndpointsFromStatus(su.status)
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
	ports := getExternalServicePorts(svc)
	su.externalServicePorts = ports
	if su.externalStatusAddress != "" {
		glog.V(3).Info("skipping external service address/ports - external-status-address is set and takes precedence")
		return
	}
	su.saveStatus(ips)
	su.externalEndpoints = su.generateExternalEndpointsFromStatus(su.status)
}

func (su *statusUpdater) retryUpdateVirtualServerStatus(vsCopy *conf_v1.VirtualServer) error {
	vs, err := su.confClient.K8sV1().VirtualServers(vsCopy.Namespace).Get(context.TODO(), vsCopy.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	vs.Status = vsCopy.Status
	_, err = su.confClient.K8sV1().VirtualServers(vs.Namespace).UpdateStatus(context.TODO(), vs, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	return nil
}

func (su *statusUpdater) retryUpdateVirtualServerRouteStatus(vsrCopy *conf_v1.VirtualServerRoute) error {
	vsr, err := su.confClient.K8sV1().VirtualServerRoutes(vsrCopy.Namespace).Get(context.TODO(), vsrCopy.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	vsr.Status = vsrCopy.Status
	_, err = su.confClient.K8sV1().VirtualServerRoutes(vsr.Namespace).UpdateStatus(context.TODO(), vsr, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	return nil
}

func hasVsStatusChanged(vs *conf_v1.VirtualServer, state string, reason string, message string) bool {
	if vs.Status.State != state {
		return true
	}

	if vs.Status.Reason != reason {
		return true
	}

	if vs.Status.Message != message {
		return true
	}

	return false
}

// UpdateVirtualServerStatus updates the status of a VirtualServer.
func (su *statusUpdater) UpdateVirtualServerStatus(vs *conf_v1.VirtualServer, state string, reason string, message string) error {
	if !hasVsStatusChanged(vs, state, reason, message) {
		return nil
	}

	vsCopy := vs.DeepCopy()
	vsCopy.Status.State = state
	vsCopy.Status.Reason = reason
	vsCopy.Status.Message = message
	vsCopy.Status.ExternalEndpoints = su.externalEndpoints

	_, err := su.confClient.K8sV1().VirtualServers(vsCopy.Namespace).UpdateStatus(context.TODO(), vsCopy, metav1.UpdateOptions{})
	if err != nil {
		glog.V(3).Infof("error setting VirtualServer %v/%v status, retrying: %v", vsCopy.Namespace, vsCopy.Name, err)
		return su.retryUpdateVirtualServerStatus(vsCopy)
	}
	return err
}

func hasVsrStatusChanged(vsr *conf_v1.VirtualServerRoute, state string, reason string, message string, referencedByString string) bool {
	if vsr.Status.State != state {
		return true
	}

	if vsr.Status.Reason != reason {
		return true
	}

	if vsr.Status.Message != message {
		return true
	}

	if referencedByString != "" && vsr.Status.ReferencedBy != referencedByString {
		return true
	}

	return false
}

// UpdateVirtualServerRouteStatusWithReferencedBy updates the status of a VirtualServerRoute, including the referencedBy field.
func (su *statusUpdater) UpdateVirtualServerRouteStatusWithReferencedBy(vsr *conf_v1.VirtualServerRoute, state string, reason string, message string, referencedBy []*v1.VirtualServer) error {
	var referencedByString string
	if len(referencedBy) != 0 {
		vs := referencedBy[0]
		referencedByString = fmt.Sprintf("%v/%v", vs.Namespace, vs.Name)
	}

	vsrCopy := vsr.DeepCopy()
	vsrCopy.Status.State = state
	vsrCopy.Status.Reason = reason
	vsrCopy.Status.Message = message
	vsrCopy.Status.ReferencedBy = referencedByString
	vsrCopy.Status.ExternalEndpoints = su.externalEndpoints

	_, err := su.confClient.K8sV1().VirtualServerRoutes(vsrCopy.Namespace).UpdateStatus(context.TODO(), vsrCopy, metav1.UpdateOptions{})
	if err != nil {
		glog.V(3).Infof("error setting VirtualServerRoute %v/%v status, retrying: %v", vsrCopy.Namespace, vsrCopy.Name, err)
		return su.retryUpdateVirtualServerRouteStatus(vsrCopy)
	}
	return err
}

// UpdateVirtualServerRouteStatus updates the status of a VirtualServerRoute.
// This method does not clear or update the referencedBy field of the status.
// If you need to update the referencedBy field, use UpdateVirtualServerRouteStatusWithReferencedBy instead.
func (su *statusUpdater) UpdateVirtualServerRouteStatus(vsr *conf_v1.VirtualServerRoute, state string, reason string, message string) error {

	if !hasVsrStatusChanged(vsr, state, reason, message, "") {
		return nil
	}

	vsrCopy := vsr.DeepCopy()
	vsrCopy.Status.State = state
	vsrCopy.Status.Reason = reason
	vsrCopy.Status.Message = message
	vsrCopy.Status.ExternalEndpoints = su.externalEndpoints

	_, err := su.confClient.K8sV1().VirtualServerRoutes(vsrCopy.Namespace).UpdateStatus(context.TODO(), vsrCopy, metav1.UpdateOptions{})
	if err != nil {
		glog.V(3).Infof("error setting VirtualServerRoute %v/%v status, retrying: %v", vsrCopy.Namespace, vsrCopy.Name, err)
		return su.retryUpdateVirtualServerRouteStatus(vsrCopy)
	}
	return err
}

func (su *statusUpdater) updateVirtualServerExternalEndpoints(vs *conf_v1.VirtualServer) error {
	vsCopy := vs.DeepCopy()
	vsCopy.Status.ExternalEndpoints = su.externalEndpoints

	_, err := su.confClient.K8sV1().VirtualServers(vsCopy.Namespace).UpdateStatus(context.TODO(), vsCopy, metav1.UpdateOptions{})
	if err != nil {
		glog.V(3).Infof("error setting VirtualServer %v/%v status, retrying: %v", vsCopy.Namespace, vsCopy.Name, err)
		return su.retryUpdateVirtualServerStatus(vsCopy)
	}
	return err
}

func (su *statusUpdater) updateVirtualServerRouteExternalEndpoints(vsr *conf_v1.VirtualServerRoute) error {
	vsrCopy := vsr.DeepCopy()
	vsrCopy.Status.ExternalEndpoints = su.externalEndpoints

	_, err := su.confClient.K8sV1().VirtualServerRoutes(vsrCopy.Namespace).UpdateStatus(context.TODO(), vsrCopy, metav1.UpdateOptions{})
	if err != nil {
		glog.V(3).Infof("error setting VirtualServerRoute %v/%v status, retrying: %v", vsrCopy.Namespace, vsrCopy.Name, err)
		return su.retryUpdateVirtualServerRouteStatus(vsrCopy)
	}
	return err
}

// UpdateVsVsrExternalEndpoints updates all the external endpoints for the given VirtualServer and VirtualServerRoutes statuses.
func (su *statusUpdater) UpdateVsVsrExternalEndpoints(vss []*conf_v1.VirtualServer, vsrs []*conf_v1.VirtualServerRoute) error {
	if len(vss) < 1 && len(vsrs) < 1 {
		glog.V(3).Info("no VirtualServers or VirtualServerRoutes to update")
		return nil
	}

	var errorMsg string
	for _, vs := range vss {
		err := su.updateVirtualServerExternalEndpoints(vs)
		if err != nil {
			errorMsg = "not all VirtualServers updated"
		}
	}

	for _, vsr := range vsrs {
		err := su.updateVirtualServerRouteExternalEndpoints(vsr)
		if err != nil {
			if errorMsg != "" {
				errorMsg += ", "
			}
			errorMsg += "not all VirtualServerRoutes updated"
		}
	}

	if errorMsg != "" {
		return fmt.Errorf(errorMsg)
	}

	return nil
}

func (su *statusUpdater) generateExternalEndpointsFromStatus(status []api_v1.LoadBalancerIngress) []conf_v1.ExternalEndpoint {
	var externalEndpoints []conf_v1.ExternalEndpoint
	for _, lb := range status {
		endpoint := conf_v1.ExternalEndpoint{IP: lb.IP, Ports: su.externalServicePorts}
		externalEndpoints = append(externalEndpoints, endpoint)
	}

	return externalEndpoints
}
