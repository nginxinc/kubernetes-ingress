package configs

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/nginxinc/kubernetes-ingress/internal/configs/version2"

	"github.com/golang/glog"
	"github.com/nginxinc/kubernetes-ingress/internal/configs/version1"
	"github.com/nginxinc/kubernetes-ingress/internal/nginx"
	conf_v1alpha1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1alpha1"
	api_v1 "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const pemFileNameForMissingTLSSecret = "/etc/nginx/secrets/default"
const pemFileNameForWildcardTLSSecret = "/etc/nginx/secrets/wildcard"

// DefaultServerSecretName is the filename of the Secret with a TLS cert and a key for the default server.
const DefaultServerSecretName = "default"

// WildcardSecretName is the filename of the Secret with a TLS cert and a key for the ingress resources with TLS termination enabled but not secret defined.
const WildcardSecretName = "wildcard"

// JWTKeyKey is the key of the data field of a Secret where the JWK must be stored.
const JWTKeyKey = "jwk"

// Configurator configures NGINX.
type Configurator struct {
	nginxManager       nginx.Manager
	staticCfgParams    *StaticConfigParams
	cfgParams          *ConfigParams
	templateExecutor   *version1.TemplateExecutor
	templateExecutorV2 *version2.TemplateExecutor
	ingresses          map[string]*IngressEx
	minions            map[string]map[string]bool
	virtualServers     map[string]*VirtualServerEx
	isWildcardEnabled  bool
	isPlus             bool
}

// NewConfigurator creates a new Configurator.
func NewConfigurator(nginxManager nginx.Manager, staticCfgParams *StaticConfigParams, config *ConfigParams, templateExecutor *version1.TemplateExecutor,
	templateExecutorV2 *version2.TemplateExecutor, isPlus bool, isWildcardEnabled bool) *Configurator {
	cnf := Configurator{
		nginxManager:       nginxManager,
		staticCfgParams:    staticCfgParams,
		cfgParams:          config,
		ingresses:          make(map[string]*IngressEx),
		virtualServers:     make(map[string]*VirtualServerEx),
		templateExecutor:   templateExecutor,
		templateExecutorV2: templateExecutorV2,
		minions:            make(map[string]map[string]bool),
		isPlus:             isPlus,
		isWildcardEnabled:  isWildcardEnabled,
	}
	return &cnf
}

// AddOrUpdateDHParam creates a dhparam file with the content of the string.
func (cnf *Configurator) AddOrUpdateDHParam(content string) (string, error) {
	return cnf.nginxManager.CreateDHParam(content)
}

// AddOrUpdateIngress adds or updates NGINX configuration for the Ingress resource.
func (cnf *Configurator) AddOrUpdateIngress(ingEx *IngressEx) error {
	if err := cnf.addOrUpdateIngress(ingEx); err != nil {
		return fmt.Errorf("Error adding or updating ingress %v/%v: %v", ingEx.Ingress.Namespace, ingEx.Ingress.Name, err)
	}

	if err := cnf.nginxManager.Reload(); err != nil {
		return fmt.Errorf("Error reloading NGINX for %v/%v: %v", ingEx.Ingress.Namespace, ingEx.Ingress.Name, err)
	}

	return nil
}

func (cnf *Configurator) addOrUpdateIngress(ingEx *IngressEx) error {
	pems := cnf.updateTLSSecrets(ingEx)
	jwtKeyFileName := cnf.updateJWKSecret(ingEx)

	isMinion := false
	nginxCfg := generateNginxCfg(ingEx, pems, isMinion, cnf.cfgParams, cnf.isPlus, cnf.IsResolverConfigured(), jwtKeyFileName)

	name := objectMetaToFileName(&ingEx.Ingress.ObjectMeta)
	content, err := cnf.templateExecutor.ExecuteIngressConfigTemplate(&nginxCfg)
	if err != nil {
		return fmt.Errorf("Error generating Ingress Config %v: %v", name, err)
	}
	cnf.nginxManager.CreateConfig(name, content)

	cnf.ingresses[name] = ingEx

	return nil
}

// AddOrUpdateMergeableIngress adds or updates NGINX configuration for the Ingress resources with Mergeable Types.
func (cnf *Configurator) AddOrUpdateMergeableIngress(mergeableIngs *MergeableIngresses) error {
	if err := cnf.addOrUpdateMergeableIngress(mergeableIngs); err != nil {
		return fmt.Errorf("Error when adding or updating ingress %v/%v: %v", mergeableIngs.Master.Ingress.Namespace, mergeableIngs.Master.Ingress.Name, err)
	}

	if err := cnf.nginxManager.Reload(); err != nil {
		return fmt.Errorf("Error reloading NGINX for %v/%v: %v", mergeableIngs.Master.Ingress.Namespace, mergeableIngs.Master.Ingress.Name, err)
	}

	return nil
}

func (cnf *Configurator) addOrUpdateMergeableIngress(mergeableIngs *MergeableIngresses) error {
	masterPems := cnf.updateTLSSecrets(mergeableIngs.Master)
	masterJwtKeyFileName := cnf.updateJWKSecret(mergeableIngs.Master)
	minionJwtKeyFileNames := make(map[string]string)
	for _, minion := range mergeableIngs.Minions {
		minionName := objectMetaToFileName(&minion.Ingress.ObjectMeta)
		minionJwtKeyFileNames[minionName] = cnf.updateJWKSecret(minion)
	}

	nginxCfg := generateNginxCfgForMergeableIngresses(mergeableIngs, masterPems, masterJwtKeyFileName, minionJwtKeyFileNames, cnf.cfgParams, cnf.isPlus, cnf.IsResolverConfigured())

	name := objectMetaToFileName(&mergeableIngs.Master.Ingress.ObjectMeta)
	content, err := cnf.templateExecutor.ExecuteIngressConfigTemplate(&nginxCfg)
	if err != nil {
		return fmt.Errorf("Error generating Ingress Config %v: %v", name, err)
	}
	cnf.nginxManager.CreateConfig(name, content)

	cnf.ingresses[name] = mergeableIngs.Master
	cnf.minions[name] = make(map[string]bool)
	for _, minion := range mergeableIngs.Minions {
		minionName := objectMetaToFileName(&minion.Ingress.ObjectMeta)
		cnf.minions[name][minionName] = true
	}

	return nil
}

// AddOrUpdateVirtualServer adds or updates NGINX configuration for the VirtualServer resource.
func (cnf *Configurator) AddOrUpdateVirtualServer(virtualServerEx *VirtualServerEx) (Warnings, error) {
	warnings, err := cnf.addOrUpdateVirtualServer(virtualServerEx)
	if err != nil {
		return warnings, fmt.Errorf("Error adding or updating VirtualServer %v/%v: %v", virtualServerEx.VirtualServer.Namespace, virtualServerEx.VirtualServer.Name, err)
	}

	if err := cnf.nginxManager.Reload(); err != nil {
		return warnings, fmt.Errorf("Error reloading NGINX for VirtualServer %v/%v: %v", virtualServerEx.VirtualServer.Namespace, virtualServerEx.VirtualServer.Name, err)
	}

	return warnings, nil
}

func (cnf *Configurator) addOrUpdateOpenTracingTracerConfig(content string) error {
	err := cnf.nginxManager.CreateOpenTracingTracerConfig(content)
	return err
}

func (cnf *Configurator) addOrUpdateVirtualServer(virtualServerEx *VirtualServerEx) (Warnings, error) {
	tlsPemFileName := ""
	if virtualServerEx.TLSSecret != nil {
		tlsPemFileName = cnf.addOrUpdateTLSSecret(virtualServerEx.TLSSecret)
	}
	vsc := newVirtualServerConfigurator(cnf.cfgParams, cnf.isPlus, cnf.IsResolverConfigured())
	vsCfg, warnings := vsc.GenerateVirtualServerConfig(virtualServerEx, tlsPemFileName)

	name := getFileNameForVirtualServer(virtualServerEx.VirtualServer)
	content, err := cnf.templateExecutorV2.ExecuteVirtualServerTemplate(&vsCfg)
	if err != nil {
		return warnings, fmt.Errorf("Error generating VirtualServer config: %v: %v", name, err)
	}
	cnf.nginxManager.CreateConfig(name, content)

	cnf.virtualServers[name] = virtualServerEx

	return warnings, nil
}

func (cnf *Configurator) updateTLSSecrets(ingEx *IngressEx) map[string]string {
	pems := make(map[string]string)

	for _, tls := range ingEx.Ingress.Spec.TLS {
		secretName := tls.SecretName

		pemFileName := pemFileNameForMissingTLSSecret
		if secretName == "" && cnf.isWildcardEnabled {
			pemFileName = pemFileNameForWildcardTLSSecret
		} else if secret, exists := ingEx.TLSSecrets[secretName]; exists {
			pemFileName = cnf.addOrUpdateTLSSecret(secret)
		}

		for _, host := range tls.Hosts {
			pems[host] = pemFileName
		}
		if len(tls.Hosts) == 0 {
			pems[emptyHost] = pemFileName
		}
	}

	return pems
}

func (cnf *Configurator) updateJWKSecret(ingEx *IngressEx) string {
	if !cnf.isPlus || ingEx.JWTKey.Name == "" {
		return ""
	}

	if ingEx.JWTKey.Secret != nil {
		cnf.addOrUpdateJWKSecret(ingEx.JWTKey.Secret)
	}

	return cnf.nginxManager.GetFilenameForSecret(ingEx.Ingress.Namespace + "-" + ingEx.JWTKey.Name)
}

func (cnf *Configurator) addOrUpdateJWKSecret(secret *api_v1.Secret) string {
	name := objectMetaToFileName(&secret.ObjectMeta)
	data := secret.Data[JWTKeyKey]
	return cnf.nginxManager.CreateSecret(name, data, nginx.JWKSecretFileMode)
}

func (cnf *Configurator) AddOrUpdateJWKSecret(secret *api_v1.Secret) {
	cnf.addOrUpdateJWKSecret(secret)
}

// AddOrUpdateTLSSecret adds or updates a file with the content of the TLS secret.
func (cnf *Configurator) AddOrUpdateTLSSecret(secret *api_v1.Secret, ingExes []IngressEx, mergeableIngresses []MergeableIngresses, virtualServerExes []*VirtualServerEx) error {
	cnf.addOrUpdateTLSSecret(secret)
	for i := range ingExes {
		err := cnf.addOrUpdateIngress(&ingExes[i])
		if err != nil {
			return fmt.Errorf("Error adding or updating ingress %v/%v: %v", ingExes[i].Ingress.Namespace, ingExes[i].Ingress.Name, err)
		}
	}

	for i := range mergeableIngresses {
		err := cnf.addOrUpdateMergeableIngress(&mergeableIngresses[i])
		if err != nil {
			return fmt.Errorf("Error adding or updating mergeableIngress %v/%v: %v", mergeableIngresses[i].Master.Ingress.Namespace, mergeableIngresses[i].Master.Ingress.Name, err)
		}
	}

	for _, vsEx := range virtualServerExes {
		// It is safe to ignore warnings here as no new warnings should appear when adding or updating a secret
		_, err := cnf.addOrUpdateVirtualServer(vsEx)
		if err != nil {
			return fmt.Errorf("Error adding or updating VirtualServer %v/%v: %v", vsEx.VirtualServer.Namespace, vsEx.VirtualServer.Name, err)
		}
	}

	if err := cnf.nginxManager.Reload(); err != nil {
		return fmt.Errorf("Error when reloading NGINX when updating Secret: %v", err)
	}

	return nil
}

func (cnf *Configurator) addOrUpdateTLSSecret(secret *api_v1.Secret) string {
	name := objectMetaToFileName(&secret.ObjectMeta)
	data := GenerateCertAndKeyFileContent(secret)
	return cnf.nginxManager.CreateSecret(name, data, nginx.TLSSecretFileMode)
}

// AddOrUpdateSpecialTLSSecrets adds or updates a file with a TLS cert and a key from a Special TLS Secret (eg. DefaultServerSecret, WildcardTLSSecret).
func (cnf *Configurator) AddOrUpdateSpecialTLSSecrets(secret *api_v1.Secret, secretNames []string) error {
	data := GenerateCertAndKeyFileContent(secret)

	for _, secretName := range secretNames {
		cnf.nginxManager.CreateSecret(secretName, data, nginx.TLSSecretFileMode)
	}

	if err := cnf.nginxManager.Reload(); err != nil {
		return fmt.Errorf("Error when reloading NGINX when updating the special Secrets: %v", err)
	}

	return nil
}

// GenerateCertAndKeyFileContent generates a pem file content from the TLS secret.
func GenerateCertAndKeyFileContent(secret *api_v1.Secret) []byte {
	var res bytes.Buffer

	res.Write(secret.Data[api_v1.TLSCertKey])
	res.WriteString("\n")
	res.Write(secret.Data[api_v1.TLSPrivateKeyKey])

	return res.Bytes()
}

// DeleteSecret deletes the file associated with the secret and the configuration files for Ingress and VirtualServer resources.
// NGINX is reloaded only when the total number of the resources > 0.
func (cnf *Configurator) DeleteSecret(key string, ingExes []IngressEx, mergeableIngresses []MergeableIngresses, virtualServerExes []*VirtualServerEx) error {
	cnf.nginxManager.DeleteSecret(keyToFileName(key))

	for i := range ingExes {
		err := cnf.addOrUpdateIngress(&ingExes[i])
		if err != nil {
			return fmt.Errorf("Error adding or updating ingress %v/%v: %v", ingExes[i].Ingress.Namespace, ingExes[i].Ingress.Name, err)
		}
	}

	for i := range mergeableIngresses {
		err := cnf.addOrUpdateMergeableIngress(&mergeableIngresses[i])
		if err != nil {
			return fmt.Errorf("Error adding or updating mergeableIngress %v/%v: %v", mergeableIngresses[i].Master.Ingress.Namespace, mergeableIngresses[i].Master.Ingress.Name, err)
		}
	}

	for _, vsEx := range virtualServerExes {
		// It is safe to ignore warnings here as no new warnings should appear when deleting a secret
		_, err := cnf.addOrUpdateVirtualServer(vsEx)
		if err != nil {
			return fmt.Errorf("Error adding or updating VirtualServer %v/%v: %v", vsEx.VirtualServer.Namespace, vsEx.VirtualServer.Name, err)
		}
	}

	if len(ingExes)+len(mergeableIngresses)+len(virtualServerExes) > 0 {
		if err := cnf.nginxManager.Reload(); err != nil {
			return fmt.Errorf("Error when reloading NGINX when deleting Secret %v: %v", key, err)
		}
	}

	return nil
}

// DeleteIngress deletes NGINX configuration for the Ingress resource.
func (cnf *Configurator) DeleteIngress(key string) error {
	name := keyToFileName(key)
	cnf.nginxManager.DeleteConfig(name)

	delete(cnf.ingresses, name)
	delete(cnf.minions, name)

	if err := cnf.nginxManager.Reload(); err != nil {
		return fmt.Errorf("Error when removing ingress %v: %v", key, err)
	}

	return nil
}

// DeleteVirtualServer deletes NGINX configuration for the VirtualServer resource.
func (cnf *Configurator) DeleteVirtualServer(key string) error {
	name := getFileNameForVirtualServerFromKey(key)
	cnf.nginxManager.DeleteConfig(name)

	delete(cnf.virtualServers, name)

	if err := cnf.nginxManager.Reload(); err != nil {
		return fmt.Errorf("Error when removing VirtualServer %v: %v", key, err)
	}

	return nil
}

// UpdateEndpoints updates endpoints in NGINX configuration for the Ingress resources.
func (cnf *Configurator) UpdateEndpoints(ingExes []*IngressEx) error {
	reloadPlus := false

	for _, ingEx := range ingExes {
		err := cnf.addOrUpdateIngress(ingEx)
		if err != nil {
			return fmt.Errorf("Error adding or updating ingress %v/%v: %v", ingEx.Ingress.Namespace, ingEx.Ingress.Name, err)
		}

		if cnf.isPlus {
			err := cnf.updatePlusEndpoints(ingEx)
			if err != nil {
				glog.Warningf("Couldn't update the endpoints via the API: %v; reloading configuration instead", err)
				reloadPlus = true
			}
		}
	}

	if cnf.isPlus && !reloadPlus {
		glog.V(3).Info("No need to reload nginx")
		return nil
	}

	if err := cnf.nginxManager.Reload(); err != nil {
		return fmt.Errorf("Error reloading NGINX when updating endpoints: %v", err)
	}

	return nil
}

// UpdateEndpointsMergeableIngress updates endpoints in NGINX configuration for a mergeable Ingress resource.
func (cnf *Configurator) UpdateEndpointsMergeableIngress(mergeableIngresses []*MergeableIngresses) error {
	reloadPlus := false

	for i := range mergeableIngresses {
		err := cnf.addOrUpdateMergeableIngress(mergeableIngresses[i])
		if err != nil {
			return fmt.Errorf("Error adding or updating mergeableIngress %v/%v: %v", mergeableIngresses[i].Master.Ingress.Namespace, mergeableIngresses[i].Master.Ingress.Name, err)
		}

		if cnf.isPlus {
			for _, ing := range mergeableIngresses[i].Minions {
				err = cnf.updatePlusEndpoints(ing)
				if err != nil {
					glog.Warningf("Couldn't update the endpoints via the API: %v; reloading configuration instead", err)
					reloadPlus = true
				}
			}
		}
	}

	if cnf.isPlus && !reloadPlus {
		glog.V(3).Info("No need to reload nginx")
		return nil
	}

	if err := cnf.nginxManager.Reload(); err != nil {
		return fmt.Errorf("Error reloading NGINX when updating endpoints for %v: %v", mergeableIngresses, err)
	}

	return nil
}

// UpdateEndpointsForVirtualServers updates endpoints in NGINX configuration for the s resources.
func (cnf *Configurator) UpdateEndpointsForVirtualServers(virtualServerExes []*VirtualServerEx) error {
	reloadPlus := false

	for _, vs := range virtualServerExes {
		// It is safe to ignore warnings here as no new warnings should appear when updating Endpoints for VirtualServers
		_, err := cnf.addOrUpdateVirtualServer(vs)
		if err != nil {
			return fmt.Errorf("Error adding or updating VirtualServer %v/%v: %v", vs.VirtualServer.Namespace, vs.VirtualServer.Name, err)
		}

		if cnf.isPlus {
			err := cnf.updatePlusEndpointsForVirtualServer(vs)
			if err != nil {
				glog.Warningf("Couldn't update the endpoints via the API: %v; reloading configuration instead", err)
				reloadPlus = true
			}
		}
	}

	if cnf.isPlus && !reloadPlus {
		glog.V(3).Info("No need to reload nginx")
		return nil
	}

	if err := cnf.nginxManager.Reload(); err != nil {
		return fmt.Errorf("Error reloading NGINX when updating endpoints: %v", err)
	}

	return nil
}

func (cnf *Configurator) updatePlusEndpointsForVirtualServer(virtualServerEx *VirtualServerEx) error {
	upstreams := createUpstreamsForPlus(virtualServerEx, cnf.cfgParams)
	for _, upstream := range upstreams {
		serverCfg := createUpstreamServersConfigForPlus(upstream)

		endpoints := createEndpointsFromUpstream(upstream)

		err := cnf.nginxManager.UpdateServersInPlus(upstream.Name, endpoints, serverCfg)
		if err != nil {
			return fmt.Errorf("Couldn't update the endpoints for %v: %v", upstream.Name, err)
		}
	}

	return nil
}

func (cnf *Configurator) updatePlusEndpoints(ingEx *IngressEx) error {
	ingCfg := parseAnnotations(ingEx, cnf.cfgParams, cnf.isPlus)

	cfg := nginx.ServerConfig{
		MaxFails:    ingCfg.MaxFails,
		MaxConns:    ingCfg.MaxConns,
		FailTimeout: ingCfg.FailTimeout,
		SlowStart:   ingCfg.SlowStart,
	}

	if ingEx.Ingress.Spec.Backend != nil {
		endps, exists := ingEx.Endpoints[ingEx.Ingress.Spec.Backend.ServiceName+ingEx.Ingress.Spec.Backend.ServicePort.String()]
		if exists {
			if _, isExternalName := ingEx.ExternalNameSvcs[ingEx.Ingress.Spec.Backend.ServiceName]; isExternalName {
				glog.V(3).Infof("Service %s is Type ExternalName, skipping NGINX Plus endpoints update via API", ingEx.Ingress.Spec.Backend.ServiceName)
			} else {
				name := getNameForUpstream(ingEx.Ingress, emptyHost, ingEx.Ingress.Spec.Backend)
				err := cnf.nginxManager.UpdateServersInPlus(name, endps, cfg)
				if err != nil {
					return fmt.Errorf("Couldn't update the endpoints for %v: %v", name, err)
				}
			}
		}
	}

	for _, rule := range ingEx.Ingress.Spec.Rules {
		if rule.IngressRuleValue.HTTP == nil {
			continue
		}

		for _, path := range rule.HTTP.Paths {
			endps, exists := ingEx.Endpoints[path.Backend.ServiceName+path.Backend.ServicePort.String()]
			if exists {
				if _, isExternalName := ingEx.ExternalNameSvcs[path.Backend.ServiceName]; isExternalName {
					glog.V(3).Infof("Service %s is Type ExternalName, skipping NGINX Plus endpoints update via API", path.Backend.ServiceName)
					continue
				}

				name := getNameForUpstream(ingEx.Ingress, rule.Host, &path.Backend)
				err := cnf.nginxManager.UpdateServersInPlus(name, endps, cfg)
				if err != nil {
					return fmt.Errorf("Couldn't update the endpoints for %v: %v", name, err)
				}
			}
		}
	}

	return nil
}

// UpdateConfig updates NGINX configuration parameters.
func (cnf *Configurator) UpdateConfig(cfgParams *ConfigParams, ingExes []*IngressEx, mergeableIngs map[string]*MergeableIngresses, virtualServerExes []*VirtualServerEx) (Warnings, error) {
	cnf.cfgParams = cfgParams
	allWarnings := newWarnings()

	if cnf.cfgParams.MainServerSSLDHParamFileContent != nil {
		fileName, err := cnf.nginxManager.CreateDHParam(*cnf.cfgParams.MainServerSSLDHParamFileContent)
		if err != nil {
			return allWarnings, fmt.Errorf("Error when updating dhparams: %v", err)
		}
		cfgParams.MainServerSSLDHParam = fileName
	}

	if cfgParams.MainTemplate != nil {
		err := cnf.templateExecutor.UpdateMainTemplate(cfgParams.MainTemplate)
		if err != nil {
			return allWarnings, fmt.Errorf("Error when parsing the main template: %v", err)
		}
	}

	if cfgParams.IngressTemplate != nil {
		err := cnf.templateExecutor.UpdateIngressTemplate(cfgParams.IngressTemplate)
		if err != nil {
			return allWarnings, fmt.Errorf("Error when parsing the ingress template: %v", err)
		}
	}

	mainCfg := GenerateNginxMainConfig(cnf.staticCfgParams, cfgParams)
	mainCfgContent, err := cnf.templateExecutor.ExecuteMainConfigTemplate(mainCfg)
	if err != nil {
		return allWarnings, fmt.Errorf("Error when writing main Config")
	}
	cnf.nginxManager.CreateMainConfig(mainCfgContent)

	for _, ingEx := range ingExes {
		if err := cnf.addOrUpdateIngress(ingEx); err != nil {
			return allWarnings, err
		}
	}
	for _, mergeableIng := range mergeableIngs {
		if err := cnf.addOrUpdateMergeableIngress(mergeableIng); err != nil {
			return allWarnings, err
		}
	}
	for _, vsEx := range virtualServerExes {
		warnings, err := cnf.addOrUpdateVirtualServer(vsEx)
		if err != nil {
			return allWarnings, err
		}
		allWarnings.Add(warnings)
	}

	if mainCfg.OpenTracingLoadModule {
		if err := cnf.addOrUpdateOpenTracingTracerConfig(mainCfg.OpenTracingTracerConfig); err != nil {
			return allWarnings, fmt.Errorf("Error when updating OpenTracing tracer config: %v", err)
		}
	}

	cnf.nginxManager.SetOpenTracing(mainCfg.OpenTracingLoadModule)
	if err := cnf.nginxManager.Reload(); err != nil {
		return allWarnings, fmt.Errorf("Error when updating config from ConfigMap: %v", err)
	}

	return allWarnings, nil
}

func keyToFileName(key string) string {
	return strings.Replace(key, "/", "-", -1)
}

func objectMetaToFileName(meta *meta_v1.ObjectMeta) string {
	return meta.Namespace + "-" + meta.Name
}

func getFileNameForVirtualServer(virtualServer *conf_v1alpha1.VirtualServer) string {
	return fmt.Sprintf("vs_%s_%s", virtualServer.Namespace, virtualServer.Name)
}

func getFileNameForVirtualServerFromKey(key string) string {
	replaced := strings.Replace(key, "/", "_", -1)
	return fmt.Sprintf("vs_%s", replaced)
}

// HasIngress checks if the Ingress resource is present in NGINX configuration.
func (cnf *Configurator) HasIngress(ing *extensions.Ingress) bool {
	name := objectMetaToFileName(&ing.ObjectMeta)
	_, exists := cnf.ingresses[name]
	return exists
}

// HasMinion checks if the minion Ingress resource of the master is present in NGINX configuration.
func (cnf *Configurator) HasMinion(master *extensions.Ingress, minion *extensions.Ingress) bool {
	masterName := objectMetaToFileName(&master.ObjectMeta)

	if _, exists := cnf.minions[masterName]; !exists {
		return false
	}

	return cnf.minions[masterName][objectMetaToFileName(&minion.ObjectMeta)]
}

// IsResolverConfigured checks if a DNS resolver is present in NGINX configuration.
func (cnf *Configurator) IsResolverConfigured() bool {
	return len(cnf.cfgParams.ResolverAddresses) != 0
}

// GetIngressCounts returns the total count of Ingress resources that are handled by the Ingress Controller grouped by their type
func (cnf *Configurator) GetIngressCounts() map[string]int {
	counters := map[string]int{
		"master":  0,
		"regular": 0,
		"minion":  0,
	}

	// cnf.ingresses contains only master and regular Ingress Resources
	for _, ing := range cnf.ingresses {
		if ing.Ingress.Annotations["nginx.org/mergeable-ingress-type"] == "master" {
			counters["master"]++
		} else {
			counters["regular"]++
		}
	}

	for _, min := range cnf.minions {
		counters["minion"] += len(min)
	}

	return counters
}

// GetVirtualServerCounts returns the total count of VS/VSR resources that are handled by the Ingress Controller
func (cnf *Configurator) GetVirtualServerCounts() (vsCount int, vsrCount int) {
	vsCount = len(cnf.virtualServers)
	for _, vs := range cnf.virtualServers {
		vsrCount += len(vs.VirtualServerRoutes)
	}

	return vsCount, vsrCount
}
