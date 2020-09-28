package k8s

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/nginxinc/kubernetes-ingress/internal/configs"
	conf_v1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"
	"github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/validation"
	networking "k8s.io/api/networking/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const ingressKind = "Ingress"
const virtualServerKind = "VirtualServer"
const virtualServerRouteKind = "VirtualServerRoute"

// Operation defines an operation to perform for a resource.
type Operation int

const (
	// Delete the config of the resource
	Delete Operation = iota
	// AddOrUpdate the config of the resource
	AddOrUpdate
)

// Resource represents a configuration resource.
// A Resource can be a top level configuration object:
// - Regular or Master Ingress
// - VirtualServer
type Resource interface {
	GetObjectMeta() *metav1.ObjectMeta
	GetKeyWithKind() string
	AcquireHost(host string)
	ReleaseHost(host string)
	Wins(resource Resource) bool
	AddWarning(warning string)
	IsEqual(resource Resource) bool
}

func chooseObjectMetaWinner(meta1 *metav1.ObjectMeta, meta2 *metav1.ObjectMeta) bool {
	if meta1.CreationTimestamp.Equal(&meta2.CreationTimestamp) {
		return meta1.UID > meta2.UID
	}

	return meta1.CreationTimestamp.Before(&meta2.CreationTimestamp)
}

// ResourceChange represents a change of the resource that needs to be reflected in the NGINX config.
type ResourceChange struct {
	// Op is an operation that needs be performed on the resource.
	Op Operation
	// Resource is the target resource.
	Resource Resource
	// Error is the error associated with the resource.
	Error string
}

// ConfigurationProblem is a problem associated with a configuration object.
type ConfigurationProblem struct {
	// Object is a configuration object.
	Object runtime.Object
	// IsError tells if the problem is an error. If it is an error, then it is expected that the status of the object
	// will be updated to the state 'invalid'. Otherwise, the state will be 'warning'.
	IsError bool
	// Reason tells the reason. It matches the reason in the events/status of our configuration objects.
	Reason string
	// Messages gives the details about the problem. It matches the message in the events/status of our configuration objects.
	Message string
}

func compareConfigurationProblems(problem1 *ConfigurationProblem, problem2 *ConfigurationProblem) bool {
	return problem1.IsError == problem2.IsError &&
		problem1.Reason == problem2.Reason &&
		problem1.Message == problem2.Message
}

// FullIngress holds an Ingress resource with its minions. It implements the Resource interface.
type FullIngress struct {
	// Ingress holds a regular Ingress or a master Ingress.
	Ingress *networking.Ingress
	// IsMaster is true when the Ingress is a master.
	IsMaster bool
	// Minions contains minions if the Ingress is a master.
	Minions []*FullMinion
	// ValidHosts marks the hosts of the Ingress as valid (true) or invalid (false).
	// Regular Ingress resources can have multiple hosts. It is possible that some of the hosts are taken by other
	// resources. In that case, those hosts will be marked as invalid.
	ValidHosts map[string]bool
	// Warnings includes all the warnings for the resource.
	Warnings []string
	// ChildWarnings includes the warnings of the minions. The key is the namespace/name.
	ChildWarnings map[string][]string
}

// NewRegularFullIngress creates a FullIngress from a regular Ingress resource.
func NewRegularFullIngress(ing *networking.Ingress) *FullIngress {
	return &FullIngress{
		Ingress:       ing,
		IsMaster:      false,
		ValidHosts:    make(map[string]bool),
		ChildWarnings: make(map[string][]string),
	}
}

// NewMasterFullIngress creates a FullIngress from a master Ingress resource.
func NewMasterFullIngress(ing *networking.Ingress, minions []*FullMinion, childWarnings map[string][]string) *FullIngress {
	return &FullIngress{
		Ingress:       ing,
		IsMaster:      true,
		Minions:       minions,
		ValidHosts:    make(map[string]bool),
		ChildWarnings: childWarnings,
	}
}

// GetObjectMeta returns the resource ObjectMeta.
func (fi *FullIngress) GetObjectMeta() *metav1.ObjectMeta {
	return &fi.Ingress.ObjectMeta
}

// GetKeyWithKind returns the key of the resource with its kind. For example, Ingress/my-namespace/my-name.
func (fi *FullIngress) GetKeyWithKind() string {
	key := getResourceKey(&fi.Ingress.ObjectMeta)
	return fmt.Sprintf("%s/%s", ingressKind, key)
}

// AcquireHost acquires the host for this resource.
func (fi *FullIngress) AcquireHost(host string) {
	fi.ValidHosts[host] = true
}

// ReleaseHost releases the host.
func (fi *FullIngress) ReleaseHost(host string) {
	fi.ValidHosts[host] = false
}

// Wins tells if this resource wins over the specified resource.
func (fi *FullIngress) Wins(resource Resource) bool {
	return chooseObjectMetaWinner(fi.GetObjectMeta(), resource.GetObjectMeta())
}

// AddWarning adds a warning.
func (fi *FullIngress) AddWarning(warning string) {
	fi.Warnings = append(fi.Warnings, warning)
}

// IsEqual tests if the FullIngress is equal to the resource.
func (fi *FullIngress) IsEqual(resource Resource) bool {
	fullIngress, ok := resource.(*FullIngress)
	if !ok {
		return false
	}

	if !compareObjectMetasWithAnnotations(&fi.Ingress.ObjectMeta, &fullIngress.Ingress.ObjectMeta) {
		return false
	}

	if !reflect.DeepEqual(fi.ValidHosts, fullIngress.ValidHosts) {
		return false
	}

	if fi.IsMaster != fullIngress.IsMaster {
		return false
	}

	if len(fi.Minions) != len(fullIngress.Minions) {
		return false
	}

	for i := range fi.Minions {
		if !compareObjectMetasWithAnnotations(&fi.Minions[i].Ingress.ObjectMeta, &fullIngress.Minions[i].Ingress.ObjectMeta) {
			return false
		}
	}

	return true
}

// FullMinion holds a Minion resource.
type FullMinion struct {
	// Ingress is the Ingress behind a minion.
	Ingress *networking.Ingress
	// ValidPaths marks the paths of the Ingress as valid (true) or invalid (false).
	// Minion Ingress resources can have multiple paths. It is possible that some of the paths are taken by other
	// Minions. In that case, those paths will be marked as invalid.
	ValidPaths map[string]bool
}

// NewFullMinion creates a new FullMinion.
func NewFullMinion(ing *networking.Ingress) *FullMinion {
	return &FullMinion{
		Ingress:    ing,
		ValidPaths: make(map[string]bool),
	}
}

// FullVirtualServer holds a VirtualServer along with its VirtualServerRoutes.
type FullVirtualServer struct {
	VirtualServer       *conf_v1.VirtualServer
	VirtualServerRoutes []*conf_v1.VirtualServerRoute
	Warnings            []string
}

// NewFullVirtualServer creates a NewFullVirtualServer.
func NewFullVirtualServer(vs *conf_v1.VirtualServer, vsrs []*conf_v1.VirtualServerRoute, warnings []string) *FullVirtualServer {
	return &FullVirtualServer{
		VirtualServer:       vs,
		VirtualServerRoutes: vsrs,
		Warnings:            warnings,
	}
}

// GetObjectMeta returns the resource ObjectMeta.
func (fvs *FullVirtualServer) GetObjectMeta() *metav1.ObjectMeta {
	return &fvs.VirtualServer.ObjectMeta
}

// GetKeyWithKind returns the key of the resource with its kind. For example, VirtualServer/my-namespace/my-name.
func (fvs *FullVirtualServer) GetKeyWithKind() string {
	key := getResourceKey(&fvs.VirtualServer.ObjectMeta)
	return fmt.Sprintf("%s/%s", virtualServerKind, key)
}

// AcquireHost acquires the host for this resource.
func (fvs *FullVirtualServer) AcquireHost(host string) {
	// we do nothing because we don't need to track which host belongs to VirtualServer, in contrast with the Ingress resource.
}

// ReleaseHost releases the host.
func (fvs *FullVirtualServer) ReleaseHost(host string) {
	// we do nothing because we don't need to track which host belongs to VirtualServer, in contrast with the Ingress resource.
}

// Wins tells if this resource wins over the specified resource.
// It is used to determine which resource should win over a host.
func (fvs *FullVirtualServer) Wins(resource Resource) bool {
	return chooseObjectMetaWinner(fvs.GetObjectMeta(), resource.GetObjectMeta())
}

// AddWarning adds a warning.
func (fvs *FullVirtualServer) AddWarning(warning string) {
	fvs.Warnings = append(fvs.Warnings, warning)
}

// IsEqual tests if the FullVirtualServer is equal to the resource.
func (fvs *FullVirtualServer) IsEqual(resource Resource) bool {
	fullVS, ok := resource.(*FullVirtualServer)
	if !ok {
		return false
	}

	if !compareObjectMetas(&fvs.VirtualServer.ObjectMeta, &fullVS.VirtualServer.ObjectMeta) {
		return false
	}

	if len(fvs.VirtualServerRoutes) != len(fullVS.VirtualServerRoutes) {
		return false
	}

	for i := range fvs.VirtualServerRoutes {
		if !compareObjectMetas(&fvs.VirtualServerRoutes[i].ObjectMeta, &fullVS.VirtualServerRoutes[i].ObjectMeta) {
			return false
		}
	}

	return true
}

func compareObjectMetas(meta1 *metav1.ObjectMeta, meta2 *metav1.ObjectMeta) bool {
	return meta1.Namespace == meta2.Namespace &&
		meta1.Name == meta2.Name &&
		meta1.Generation == meta2.Generation
}

func compareObjectMetasWithAnnotations(meta1 *metav1.ObjectMeta, meta2 *metav1.ObjectMeta) bool {
	return compareObjectMetas(meta1, meta2) && reflect.DeepEqual(meta1.Annotations, meta2.Annotations)
}

// Configuration represents the configuration of the Ingress Controller - a collection of configuration objects
// (Ingresses, VirtualServers, VirtualServerRoutes) ready to be transformed into NGINX config.
// It holds the latest valid state of those objects.
// The IC needs to ensure that at any point in time the NGINX config on the filesystem reflects the state
// of the objects in the Configuration.
type Configuration struct {
	hosts map[string]Resource

	// only valid resources with the matching IngressClass are stored
	ingresses           map[string]*networking.Ingress
	virtualServers      map[string]*conf_v1.VirtualServer
	virtualServerRoutes map[string]*conf_v1.VirtualServerRoute

	problems map[string]ConfigurationProblem

	hasCorrectIngressClass func(interface{}) bool
	virtualServerValidator *validation.VirtualServerValidator

	secretReferenceChecker     *secretReferenceChecker
	serviceReferenceChecker    *serviceReferenceChecker
	policyReferenceChecker     *policyReferenceChecker
	appPolicyReferenceChecker  *appProtectResourceReferenceChecker
	appLogConfReferenceChecker *appProtectResourceReferenceChecker

	lock sync.RWMutex
}

// NewConfiguration creates a new Configuration.
func NewConfiguration(hasCorrectIngressClass func(interface{}) bool, isPlus bool, virtualServerValidator *validation.VirtualServerValidator) *Configuration {
	return &Configuration{
		hosts:                      make(map[string]Resource),
		ingresses:                  make(map[string]*networking.Ingress),
		virtualServers:             make(map[string]*conf_v1.VirtualServer),
		virtualServerRoutes:        make(map[string]*conf_v1.VirtualServerRoute),
		problems:                   make(map[string]ConfigurationProblem),
		hasCorrectIngressClass:     hasCorrectIngressClass,
		virtualServerValidator:     virtualServerValidator,
		secretReferenceChecker:     newSecretReferenceChecker(isPlus),
		serviceReferenceChecker:    newServiceReferenceChecker(),
		policyReferenceChecker:     newPolicyReferenceChecker(),
		appPolicyReferenceChecker:  newAppProtectResourceReferenceChecker(configs.AppProtectPolicyAnnotation),
		appLogConfReferenceChecker: newAppProtectResourceReferenceChecker(configs.AppProtectLogConfAnnotation),
	}
}

// AddOrUpdateIngress adds or updates the Ingress resource.
func (c *Configuration) AddOrUpdateIngress(ing *networking.Ingress) ([]ResourceChange, []ConfigurationProblem) {
	c.lock.Lock()
	defer c.lock.Unlock()

	key := getResourceKey(&ing.ObjectMeta)
	var validationError error

	if !c.hasCorrectIngressClass(ing) {
		delete(c.ingresses, key)
	} else {
		validationError = validateIngress(ing).ToAggregate()
		if validationError != nil {
			delete(c.ingresses, key)
		} else {
			c.ingresses[key] = ing
		}
	}

	changes, problems := c.rebuild()

	if validationError != nil {
		// If the invalid resource has any active hosts, rebuild will create a change
		// to remove the resource.
		// Here we add the validationErr to that change.
		keyWithKind := getResourceKeyWithKind(ingressKind, &ing.ObjectMeta)
		for i := range changes {
			k := changes[i].Resource.GetKeyWithKind()

			if k == keyWithKind {
				changes[i].Error = validationError.Error()
				return changes, problems
			}
		}

		// On the other hand, the invalid resource might not have any active hosts.
		// Or the resource was invalid before and is still invalid (in some different way).
		// In those cases,  rebuild will create no change for that resource.
		// To make sure the validationErr is reported to the user, we create a problem.
		p := ConfigurationProblem{
			Object:  ing,
			IsError: true,
			Reason:  "Rejected",
			Message: validationError.Error(),
		}
		problems = append(problems, p)
	}

	return changes, problems
}

// DeleteIngress deletes an Ingress resource by the key.
func (c *Configuration) DeleteIngress(key string) ([]ResourceChange, []ConfigurationProblem) {
	c.lock.Lock()
	defer c.lock.Unlock()

	_, exists := c.ingresses[key]
	if !exists {
		return nil, nil
	}

	delete(c.ingresses, key)

	return c.rebuild()
}

// AddOrUpdateVirtualServer adds or updates the VirtualServer resource.
func (c *Configuration) AddOrUpdateVirtualServer(vs *conf_v1.VirtualServer) ([]ResourceChange, []ConfigurationProblem) {
	c.lock.Lock()
	defer c.lock.Unlock()

	key := getResourceKey(&vs.ObjectMeta)
	var validationError error

	if !c.hasCorrectIngressClass(vs) {
		delete(c.virtualServers, key)
	} else {
		validationError = c.virtualServerValidator.ValidateVirtualServer(vs)
		if validationError != nil {
			delete(c.virtualServers, key)
		} else {
			c.virtualServers[key] = vs
		}
	}

	changes, problems := c.rebuild()

	if validationError != nil {
		// If the invalid resource has an active host, rebuild will create a change
		// to remove the resource.
		// Here we add the validationErr to that change.
		kind := getResourceKeyWithKind(virtualServerKind, &vs.ObjectMeta)
		for i := range changes {
			k := changes[i].Resource.GetKeyWithKind()

			if k == kind {
				changes[i].Error = validationError.Error()
				return changes, problems
			}
		}

		// On the other hand, the invalid resource might not have any active host.
		// Or the resource was invalid before and is still invalid (in some different way).
		// In those cases,  rebuild will create no change for that resource.
		// To make sure the validationErr is reported to the user, we create a problem.
		p := ConfigurationProblem{
			Object:  vs,
			IsError: true,
			Reason:  "Rejected",
			Message: fmt.Sprintf("VirtualServer %s was rejected with error: %s", getResourceKey(&vs.ObjectMeta), validationError.Error()),
		}
		problems = append(problems, p)
	}

	return changes, problems
}

// DeleteVirtualServer deletes a VirtualServerResource by the key.
func (c *Configuration) DeleteVirtualServer(key string) ([]ResourceChange, []ConfigurationProblem) {
	c.lock.Lock()
	defer c.lock.Unlock()

	_, exists := c.virtualServers[key]
	if !exists {
		return nil, nil
	}

	delete(c.virtualServers, key)

	return c.rebuild()
}

// AddOrUpdateVirtualServerRoute adds or updates the VirtualServerRoute.
func (c *Configuration) AddOrUpdateVirtualServerRoute(vsr *conf_v1.VirtualServerRoute) ([]ResourceChange, []ConfigurationProblem) {
	c.lock.Lock()
	defer c.lock.Unlock()

	key := getResourceKey(&vsr.ObjectMeta)
	var validationError error

	if !c.hasCorrectIngressClass(vsr) {
		delete(c.virtualServerRoutes, key)
	} else {
		validationError = c.virtualServerValidator.ValidateVirtualServerRoute(vsr)
		if validationError != nil {
			delete(c.virtualServerRoutes, key)
		} else {
			c.virtualServerRoutes[key] = vsr
		}
	}

	changes, problems := c.rebuild()

	if validationError != nil {
		p := ConfigurationProblem{
			Object:  vsr,
			IsError: true,
			Reason:  "Rejected",
			Message: fmt.Sprintf("VirtualServerRoute %s was rejected with error: %s", getResourceKey(&vsr.ObjectMeta), validationError.Error()),
		}
		problems = append(problems, p)
	}

	return changes, problems
}

// DeleteVirtualServerRoute deletes a VirtualServerRoute by the key.
func (c *Configuration) DeleteVirtualServerRoute(key string) ([]ResourceChange, []ConfigurationProblem) {
	c.lock.Lock()
	defer c.lock.Unlock()

	_, exists := c.virtualServerRoutes[key]
	if !exists {
		return nil, nil
	}

	delete(c.virtualServerRoutes, key)

	return c.rebuild()
}

// GetResources returns all configuration resources.
func (c *Configuration) GetResources() []Resource {
	return c.GetResourcesWithFilter(resourceFilter{
		Ingresses:      true,
		VirtualServers: true,
	})
}

type resourceFilter struct {
	Ingresses      bool
	VirtualServers bool
}

// GetResourcesWithFilter returns resources using the filter.
func (c *Configuration) GetResourcesWithFilter(filter resourceFilter) []Resource {
	c.lock.RLock()
	defer c.lock.RUnlock()

	resources := make(map[string]Resource)

	for _, r := range c.hosts {
		switch r.(type) {
		case *FullIngress:
			if filter.Ingresses {
				resources[r.GetKeyWithKind()] = r
			}
		case *FullVirtualServer:
			if filter.VirtualServers {
				resources[r.GetKeyWithKind()] = r
			}
		}
	}

	var result []Resource
	for _, key := range getSortedResourceKeys(resources) {
		result = append(result, resources[key])
	}

	return result
}

// FindResourcesForService finds resources that reference the specified service.
func (c *Configuration) FindResourcesForService(svcNamespace string, svcName string) []Resource {
	return c.findResourcesForResourceReference(svcNamespace, svcName, c.serviceReferenceChecker)
}

// FindResourcesForEndpoints finds resources that reference the specified endpoints.
func (c *Configuration) FindResourcesForEndpoints(endpointsNamespace string, endpointsName string) []Resource {
	// Resources reference not endpoints but the corresponding service, which has the same namespace and name
	return c.FindResourcesForService(endpointsNamespace, endpointsName)
}

// FindResourcesForSecret finds resources that reference the specified secret.
func (c *Configuration) FindResourcesForSecret(secretNamespace string, secretName string) []Resource {
	return c.findResourcesForResourceReference(secretNamespace, secretName, c.secretReferenceChecker)
}

// FindResourcesForPolicy finds resources that reference the specified policy.
func (c *Configuration) FindResourcesForPolicy(policyNamespace string, policyName string) []Resource {
	return c.findResourcesForResourceReference(policyNamespace, policyName, c.policyReferenceChecker)
}

// FindResourcesForAppProtectPolicy finds resources that reference the specified AppProtect policy.
func (c *Configuration) FindResourcesForAppProtectPolicy(policyNamespace string, policyName string) []Resource {
	return c.findResourcesForResourceReference(policyNamespace, policyName, c.appPolicyReferenceChecker)
}

// FindResourcesForAppProtectLogConf finds resources that reference the specified AppProtect LogConf.
func (c *Configuration) FindResourcesForAppProtectLogConf(logConfNamespace string, logConfName string) []Resource {
	return c.findResourcesForResourceReference(logConfNamespace, logConfName, c.appLogConfReferenceChecker)
}

func (c *Configuration) findResourcesForResourceReference(namespace string, name string, checker resourceReferenceChecker) []Resource {
	c.lock.RLock()
	defer c.lock.RUnlock()

	var result []Resource

	for _, h := range getSortedResourceKeys(c.hosts) {
		r := c.hosts[h]

		switch impl := r.(type) {
		case *FullIngress:
			if checker.IsReferencedByIngress(namespace, name, impl.Ingress) {
				result = append(result, r)
				continue
			}

			for _, fm := range impl.Minions {
				if checker.IsReferencedByMinion(namespace, name, fm.Ingress) {
					result = append(result, r)
					break
				}
			}
		case *FullVirtualServer:
			if checker.IsReferencedByVirtualServer(namespace, name, impl.VirtualServer) {
				result = append(result, r)
				continue
			}

			for _, vsr := range impl.VirtualServerRoutes {
				if checker.IsReferencedByVirtualServerRoute(namespace, name, vsr) {
					result = append(result, r)
					break
				}
			}
		}
	}

	return result
}

func getResourceKey(meta *metav1.ObjectMeta) string {
	return fmt.Sprintf("%s/%s", meta.Namespace, meta.Name)
}

// rebuild rebuilds the Configuration and returns the changes to it and the new problems.
func (c *Configuration) rebuild() ([]ResourceChange, []ConfigurationProblem) {
	newHosts, newResources := c.buildHostsAndResources()

	removedHosts, updatedHosts, addedHosts := detectChangesInHosts(c.hosts, newHosts)
	changes := c.createResourceChanges(removedHosts, updatedHosts, addedHosts, newHosts)

	// safe to update hosts
	c.hosts = newHosts

	changes = squashResourceChanges(changes)

	// Note that the change will not refer to the latest version, if the resource is being removed.
	// However, referring to the latest version is necessary so that the resource latest Warnings are reported and not lost.
	// So here we make sure that changes always refer to the latest version of resources.
	for i := range changes {
		key := changes[i].Resource.GetKeyWithKind()
		if r, exists := newResources[key]; exists {
			changes[i].Resource = r
		}
	}

	newProblems := make(map[string]ConfigurationProblem)

	c.addProblemsForResourcesWithoutActiveHost(newResources, newProblems)
	c.addProblemsForOrphanMinions(newProblems)
	c.addProblemsForOrphanOrIgnoredVsrs(newProblems)

	newOrUpdatedProblems := detectChangesInProblems(newProblems, c.problems)

	// safe to update problems
	c.problems = newProblems

	return changes, newOrUpdatedProblems
}

func detectChangesInProblems(newProblems map[string]ConfigurationProblem, oldProblems map[string]ConfigurationProblem) []ConfigurationProblem {
	var result []ConfigurationProblem

	for _, key := range getSortedProblemKeys(newProblems) {
		newP := newProblems[key]

		oldP, exists := oldProblems[key]
		if !exists {
			result = append(result, newP)
			continue
		}

		if !compareConfigurationProblems(&newP, &oldP) {
			result = append(result, newP)
		}
	}

	return result
}

func (c *Configuration) addProblemsForResourcesWithoutActiveHost(resources map[string]Resource, problems map[string]ConfigurationProblem) {
	for _, k := range getSortedResourceKeys(resources) {
		r := resources[k]

		switch impl := r.(type) {
		case *FullIngress:
			atLeastOneValidHost := false
			for _, v := range impl.ValidHosts {
				if v {
					atLeastOneValidHost = true
					break
				}
			}
			if !atLeastOneValidHost {
				p := ConfigurationProblem{
					Object:  impl.Ingress,
					IsError: false,
					Reason:  "Rejected",
					Message: "All hosts are taken by other resources",
				}
				problems[r.GetKeyWithKind()] = p
			}
		case *FullVirtualServer:
			res, exists := c.hosts[impl.VirtualServer.Spec.Host]

			if !exists {
				continue
			}

			if res.GetKeyWithKind() != r.GetKeyWithKind() {
				p := ConfigurationProblem{
					Object:  impl.VirtualServer,
					IsError: false,
					Reason:  "Rejected",
					Message: "Host is taken by another resource",
				}
				problems[r.GetKeyWithKind()] = p
			}
		}
	}
}

func (c *Configuration) addProblemsForOrphanMinions(problems map[string]ConfigurationProblem) {
	for _, key := range getSortedIngressKeys(c.ingresses) {
		ing := c.ingresses[key]

		if !isMinion(ing) {
			continue
		}

		r, exists := c.hosts[ing.Spec.Rules[0].Host]
		fullIngress, ok := r.(*FullIngress)

		if !exists || !ok || !fullIngress.IsMaster {
			p := ConfigurationProblem{
				Object:  ing,
				IsError: false,
				Reason:  "NoIngressMasterFound",
				Message: "Ingress master is invalid or doesn't exist",
			}
			k := getResourceKeyWithKind(ingressKind, &ing.ObjectMeta)
			problems[k] = p
		}
	}
}

func (c *Configuration) addProblemsForOrphanOrIgnoredVsrs(problems map[string]ConfigurationProblem) {
	for _, key := range getSortedVirtualServerRouteKeys(c.virtualServerRoutes) {
		vsr := c.virtualServerRoutes[key]

		r, exists := c.hosts[vsr.Spec.Host]
		fullVS, ok := r.(*FullVirtualServer)

		if !exists || !ok {
			p := ConfigurationProblem{
				Object:  vsr,
				IsError: false,
				Reason:  "NoVirtualServerFound",
				Message: "VirtualServer is invalid or doesn't exist",
			}
			k := getResourceKeyWithKind(virtualServerRouteKind, &vsr.ObjectMeta)
			problems[k] = p
			continue
		}

		found := false
		for _, v := range fullVS.VirtualServerRoutes {
			if vsr.Namespace == v.Namespace && vsr.Name == v.Name {
				found = true
				break
			}
		}

		if !found {
			p := ConfigurationProblem{
				Object:  vsr,
				IsError: false,
				Reason:  "Ignored",
				Message: fmt.Sprintf("VirtualServer %s ignores VirtualServerRoute", getResourceKey(&fullVS.VirtualServer.ObjectMeta)),
			}
			k := getResourceKeyWithKind(virtualServerRouteKind, &vsr.ObjectMeta)
			problems[k] = p
		}
	}
}

func getResourceKeyWithKind(kind string, objectMeta *metav1.ObjectMeta) string {
	return fmt.Sprintf("%s/%s/%s", kind, objectMeta.Namespace, objectMeta.Name)
}

func (c *Configuration) createResourceChanges(removedHosts []string, updatedHosts []string, addedHosts []string, newHosts map[string]Resource) []ResourceChange {
	var changes []ResourceChange
	var deleteChanges []ResourceChange

	for _, h := range removedHosts {
		change := ResourceChange{
			Op:       Delete,
			Resource: c.hosts[h],
		}
		deleteChanges = append(deleteChanges, change)
	}

	for _, h := range updatedHosts {
		if c.hosts[h].GetKeyWithKind() != newHosts[h].GetKeyWithKind() {
			deleteChange := ResourceChange{
				Op:       Delete,
				Resource: c.hosts[h],
			}
			deleteChanges = append(deleteChanges, deleteChange)
		}

		change := ResourceChange{
			Op:       AddOrUpdate,
			Resource: newHosts[h],
		}
		changes = append(changes, change)
	}

	for _, h := range addedHosts {
		change := ResourceChange{
			Op:       AddOrUpdate,
			Resource: newHosts[h],
		}
		changes = append(changes, change)
	}

	// We need to ensure that delete changes come first.
	// This way an addOrUpdate change, which might include a resource that uses the same host as a resource
	// in a delete change, will be processed only after the config of the delete change is removed.
	// That will prevent any host collisions in the NGINX config in the state between the changes.
	return append(deleteChanges, changes...)
}

func squashResourceChanges(changes []ResourceChange) []ResourceChange {
	// deletes for the same resource become a single delete
	// updates for the same resource become a single update
	// delete and update for the same resource become a single update

	var deletes []ResourceChange
	var updates []ResourceChange

	changesPerResource := make(map[string][]ResourceChange)

	for _, c := range changes {
		key := c.Resource.GetKeyWithKind()
		changesPerResource[key] = append(changesPerResource[key], c)
	}

	// we range over the changes again to preserver the original order
	for _, c := range changes {
		key := c.Resource.GetKeyWithKind()
		resChanges, exists := changesPerResource[key]

		if !exists {
			continue
		}

		// the last element will be an update (if it exists) or a delete
		squashedChanged := resChanges[len(resChanges)-1]
		if squashedChanged.Op == Delete {
			deletes = append(deletes, squashedChanged)
		} else {
			updates = append(updates, squashedChanged)
		}

		delete(changesPerResource, key)
	}

	// We need to ensure that delete changes come first.
	// This way an addOrUpdate change, which might include a resource that uses the same host as a resource
	// in a delete change, will be processed only after the config of the delete change is removed.
	// That will prevent any host collisions in the NGINX config in the state between the changes.
	return append(deletes, updates...)
}

func (c *Configuration) buildHostsAndResources() (newHosts map[string]Resource, newResources map[string]Resource) {
	newHosts = make(map[string]Resource)
	newResources = make(map[string]Resource)

	// Step 1 - Build hosts from Ingress resources

	for _, key := range getSortedIngressKeys(c.ingresses) {
		ing := c.ingresses[key]

		if isMinion(ing) {
			continue
		}

		var resource *FullIngress

		if isMaster(ing) {
			minions, childWarnings := c.buildFullMinions(ing.Spec.Rules[0].Host)
			resource = NewMasterFullIngress(ing, minions, childWarnings)
		} else {
			resource = NewRegularFullIngress(ing)
		}

		newResources[resource.GetKeyWithKind()] = resource

		for _, rule := range ing.Spec.Rules {
			holder, exists := newHosts[rule.Host]
			if !exists {
				newHosts[rule.Host] = resource
				resource.AcquireHost(rule.Host)
				continue
			}

			warning := fmt.Sprintf("host %s is taken by another resource", rule.Host)

			if !holder.Wins(resource) {
				holder.ReleaseHost(rule.Host)
				holder.AddWarning(warning)
				newHosts[rule.Host] = resource
				resource.AcquireHost(rule.Host)
			} else {
				resource.AddWarning(warning)
			}
		}
	}

	// Step 2 - Build hosts from VirtualServer resources

	for _, key := range getSortedVirtualServerKeys(c.virtualServers) {
		vs := c.virtualServers[key]

		vsrs, warnings := c.buildVirtualServerRoutes(vs)
		resource := NewFullVirtualServer(vs, vsrs, warnings)

		newResources[resource.GetKeyWithKind()] = resource

		holder, exists := newHosts[vs.Spec.Host]
		if !exists {
			newHosts[vs.Spec.Host] = resource
			resource.AcquireHost(vs.Spec.Host)
			continue
		}

		warning := fmt.Sprintf("host %s is taken by another resource", vs.Spec.Host)

		if !holder.Wins(resource) {
			holder.ReleaseHost(vs.Spec.Host)
			newHosts[vs.Spec.Host] = resource
			resource.AcquireHost(vs.Spec.Host)
			holder.AddWarning(warning)
		} else {
			resource.AddWarning(warning)
		}
	}

	return newHosts, newResources
}

func (c *Configuration) buildFullMinions(masterHost string) ([]*FullMinion, map[string][]string) {
	var fullMinions []*FullMinion
	childWarnings := make(map[string][]string)
	paths := make(map[string]*FullMinion)

	for _, minionKey := range getSortedIngressKeys(c.ingresses) {
		ingress := c.ingresses[minionKey]

		if !isMinion(ingress) {
			continue
		}

		if masterHost != ingress.Spec.Rules[0].Host {
			continue
		}

		fullMinion := NewFullMinion(ingress)

		for _, p := range ingress.Spec.Rules[0].HTTP.Paths {
			holder, exists := paths[p.Path]
			if !exists {
				paths[p.Path] = fullMinion
				fullMinion.ValidPaths[p.Path] = true
				continue
			}

			warning := fmt.Sprintf("path %s is taken by another resource", p.Path)

			if !chooseObjectMetaWinner(&holder.Ingress.ObjectMeta, &ingress.ObjectMeta) {
				paths[p.Path] = fullMinion
				fullMinion.ValidPaths[p.Path] = true

				holder.ValidPaths[p.Path] = false
				key := getResourceKey(&holder.Ingress.ObjectMeta)
				childWarnings[key] = append(childWarnings[key], warning)
			} else {
				key := getResourceKey(&fullMinion.Ingress.ObjectMeta)
				childWarnings[key] = append(childWarnings[key], warning)
			}
		}

		fullMinions = append(fullMinions, fullMinion)
	}

	return fullMinions, childWarnings
}

func (c *Configuration) buildVirtualServerRoutes(vs *conf_v1.VirtualServer) ([]*conf_v1.VirtualServerRoute, []string) {
	var vsrs []*conf_v1.VirtualServerRoute
	var warnings []string

	for _, r := range vs.Spec.Routes {
		if r.Route == "" {
			continue
		}

		vsrKey := r.Route

		// if route is defined without a namespace, use the namespace of VirtualServer.
		if !strings.Contains(r.Route, "/") {
			vsrKey = fmt.Sprintf("%s/%s", vs.Namespace, r.Route)
		}

		vsr, exists := c.virtualServerRoutes[vsrKey]
		if !exists {
			warning := fmt.Sprintf("VirtualServerRoute %s doesn't exist or invalid", vsrKey)
			warnings = append(warnings, warning)
			continue
		}

		err := c.virtualServerValidator.ValidateVirtualServerRouteForVirtualServer(vsr, vs.Spec.Host, r.Path)
		if err != nil {
			warning := fmt.Sprintf("VirtualServerRoute %s is invalid: %v", vsrKey, err)
			warnings = append(warnings, warning)
			continue
		}

		vsrs = append(vsrs, vsr)
	}

	return vsrs, warnings
}

func getSortedIngressKeys(m map[string]*networking.Ingress) []string {
	var keys []string

	for k := range m {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	return keys
}

func getSortedVirtualServerKeys(m map[string]*conf_v1.VirtualServer) []string {
	var keys []string

	for k := range m {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	return keys
}

func getSortedVirtualServerRouteKeys(m map[string]*conf_v1.VirtualServerRoute) []string {
	var keys []string

	for k := range m {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	return keys
}

func getSortedProblemKeys(m map[string]ConfigurationProblem) []string {
	var keys []string

	for k := range m {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	return keys
}

func getSortedResourceKeys(m map[string]Resource) []string {
	var keys []string

	for k := range m {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	return keys
}

func detectChangesInHosts(oldHosts map[string]Resource, newHosts map[string]Resource) (removedHosts []string, updatedHosts []string, addedHosts []string) {
	for _, h := range getSortedResourceKeys(oldHosts) {
		_, exists := newHosts[h]
		if !exists {
			removedHosts = append(removedHosts, h)
		}
	}

	for _, h := range getSortedResourceKeys(newHosts) {
		_, exists := oldHosts[h]
		if !exists {
			addedHosts = append(addedHosts, h)
		}
	}

	for _, h := range getSortedResourceKeys(newHosts) {
		oldR, exists := oldHosts[h]
		if !exists {
			continue
		}

		if !oldR.IsEqual(newHosts[h]) {
			updatedHosts = append(updatedHosts, h)
		}
	}

	return removedHosts, updatedHosts, addedHosts
}
