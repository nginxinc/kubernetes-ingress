package configs

import (
	"bytes"
	"crypto"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/nginxinc/kubernetes-ingress/internal/k8s/secrets"
	"github.com/nginxinc/nginx-prometheus-exporter/collector"
	"github.com/spiffe/go-spiffe/workload"

	"github.com/nginxinc/kubernetes-ingress/internal/configs/version2"
	conf_v1alpha1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1alpha1"

	"github.com/golang/glog"
	api_v1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1beta1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/nginxinc/kubernetes-ingress/internal/configs/version1"
	"github.com/nginxinc/kubernetes-ingress/internal/nginx"
	conf_v1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	latCollector "github.com/nginxinc/kubernetes-ingress/internal/metrics/collectors"
)

const (
	pemFileNameForMissingTLSSecret  = "/etc/nginx/secrets/default"
	pemFileNameForWildcardTLSSecret = "/etc/nginx/secrets/wildcard"
	appProtectPolicyFolder          = "/etc/nginx/waf/nac-policies/"
	appProtectLogConfFolder         = "/etc/nginx/waf/nac-logconfs/"
	appProtectUserSigFolder         = "/etc/nginx/waf/nac-usersigs/"
	appProtectUserSigIndex          = "/etc/nginx/waf/nac-usersigs/index.conf"
)

// DefaultServerSecretName is the filename of the Secret with a TLS cert and a key for the default server.
const DefaultServerSecretName = "default"

// WildcardSecretName is the filename of the Secret with a TLS cert and a key for the ingress resources with TLS termination enabled but not secret defined.
const WildcardSecretName = "wildcard"

// JWTKeyKey is the key of the data field of a Secret where the JWK must be stored.
const JWTKeyKey = "jwk"

// CAKey is the key of the data field of a Secret where the cert must be stored.
const CAKey = "ca.crt"

// ClientSecretKey is the key of the data field of a Secret where the OIDC client secret must be stored.
const ClientSecretKey = "client-secret"

// SPIFFE filenames and modes
const (
	spiffeCertFileName   = "spiffe_cert.pem"
	spiffeKeyFileName    = "spiffe_key.pem"
	spiffeBundleFileName = "spiffe_rootca.pem"
	spiffeCertsFileMode  = os.FileMode(0644)
	spiffeKeyFileMode    = os.FileMode(0600)
)

type tlsPassthroughPair struct {
	Host       string
	UnixSocket string
}

// metricLabelsIndex keeps the relations between Ingress Controller resources and NGINX configuration.
// Used to be able to add Prometheus Metrics variable labels grouped by resource key.
type metricLabelsIndex struct {
	ingressUpstreams             map[string][]string
	virtualServerUpstreams       map[string][]string
	transportServerUpstreams     map[string][]string
	ingressServerZones           map[string][]string
	virtualServerServerZones     map[string][]string
	transportServerServerZones   map[string][]string
	ingressUpstreamPeers         map[string][]string
	virtualServerUpstreamPeers   map[string][]string
	transportServerUpstreamPeers map[string][]string
}

// Configurator configures NGINX.
type Configurator struct {
	nginxManager            nginx.Manager
	staticCfgParams         *StaticConfigParams
	cfgParams               *ConfigParams
	globalCfgParams         *GlobalConfigParams
	templateExecutor        *version1.TemplateExecutor
	templateExecutorV2      *version2.TemplateExecutor
	ingresses               map[string]*IngressEx
	minions                 map[string]map[string]bool
	virtualServers          map[string]*VirtualServerEx
	tlsPassthroughPairs     map[string]tlsPassthroughPair
	isWildcardEnabled       bool
	isPlus                  bool
	labelUpdater            collector.LabelUpdater
	metricLabelsIndex       *metricLabelsIndex
	isPrometheusEnabled     bool
	latencyCollector        latCollector.LatencyCollector
	isLatencyMetricsEnabled bool
}

// NewConfigurator creates a new Configurator.
func NewConfigurator(nginxManager nginx.Manager, staticCfgParams *StaticConfigParams, config *ConfigParams, globalCfgParams *GlobalConfigParams,
	templateExecutor *version1.TemplateExecutor, templateExecutorV2 *version2.TemplateExecutor, isPlus bool, isWildcardEnabled bool,
	labelUpdater collector.LabelUpdater, isPrometheusEnabled bool, latencyCollector latCollector.LatencyCollector, isLatencyMetricsEnabled bool) *Configurator {
	metricLabelsIndex := &metricLabelsIndex{
		ingressUpstreams:             make(map[string][]string),
		virtualServerUpstreams:       make(map[string][]string),
		transportServerUpstreams:     make(map[string][]string),
		ingressServerZones:           make(map[string][]string),
		virtualServerServerZones:     make(map[string][]string),
		transportServerServerZones:   make(map[string][]string),
		ingressUpstreamPeers:         make(map[string][]string),
		virtualServerUpstreamPeers:   make(map[string][]string),
		transportServerUpstreamPeers: make(map[string][]string),
	}

	cnf := Configurator{
		nginxManager:            nginxManager,
		staticCfgParams:         staticCfgParams,
		cfgParams:               config,
		globalCfgParams:         globalCfgParams,
		ingresses:               make(map[string]*IngressEx),
		virtualServers:          make(map[string]*VirtualServerEx),
		templateExecutor:        templateExecutor,
		templateExecutorV2:      templateExecutorV2,
		minions:                 make(map[string]map[string]bool),
		tlsPassthroughPairs:     make(map[string]tlsPassthroughPair),
		isPlus:                  isPlus,
		isWildcardEnabled:       isWildcardEnabled,
		labelUpdater:            labelUpdater,
		metricLabelsIndex:       metricLabelsIndex,
		isPrometheusEnabled:     isPrometheusEnabled,
		latencyCollector:        latencyCollector,
		isLatencyMetricsEnabled: isLatencyMetricsEnabled,
	}
	return &cnf
}

// AddOrUpdateDHParam creates a dhparam file with the content of the string.
func (cnf *Configurator) AddOrUpdateDHParam(content string) (string, error) {
	return cnf.nginxManager.CreateDHParam(content)
}

func findRemovedKeys(currentKeys []string, newKeys map[string]bool) []string {
	var removedKeys []string
	for _, name := range currentKeys {
		if _, exists := newKeys[name]; !exists {
			removedKeys = append(removedKeys, name)
		}
	}
	return removedKeys
}

func (cnf *Configurator) updateIngressMetricsLabels(ingEx *IngressEx, upstreams []version1.Upstream) {
	upstreamServerLabels := make(map[string][]string)
	newUpstreams := make(map[string]bool)
	var newUpstreamsNames []string

	upstreamServerPeerLabels := make(map[string][]string)
	newPeers := make(map[string]bool)
	var newPeersIPs []string

	for _, u := range upstreams {
		upstreamServerLabels[u.Name] = []string{u.UpstreamLabels.Service, u.UpstreamLabels.ResourceType, u.UpstreamLabels.ResourceName, u.UpstreamLabels.ResourceNamespace}
		newUpstreams[u.Name] = true
		newUpstreamsNames = append(newUpstreamsNames, u.Name)
		for _, server := range u.UpstreamServers {
			s := fmt.Sprintf("%v:%v", server.Address, server.Port)
			podInfo := ingEx.PodsByIP[s]
			labelKey := fmt.Sprintf("%v/%v", u.Name, s)
			upstreamServerPeerLabels[labelKey] = []string{podInfo.Name}
			if cnf.staticCfgParams.NginxServiceMesh {
				ownerLabelVal := fmt.Sprintf("%s/%s", podInfo.OwnerType, podInfo.OwnerName)
				upstreamServerPeerLabels[labelKey] = append(upstreamServerPeerLabels[labelKey], ownerLabelVal)
			}
			newPeers[labelKey] = true
			newPeersIPs = append(newPeersIPs, labelKey)
		}
	}

	key := fmt.Sprintf("%v/%v", ingEx.Ingress.Namespace, ingEx.Ingress.Name)
	removedUpstreams := findRemovedKeys(cnf.metricLabelsIndex.ingressUpstreams[key], newUpstreams)
	cnf.metricLabelsIndex.ingressUpstreams[key] = newUpstreamsNames
	cnf.latencyCollector.UpdateUpstreamServerLabels(upstreamServerLabels)
	cnf.latencyCollector.DeleteUpstreamServerLabels(removedUpstreams)

	removedPeers := findRemovedKeys(cnf.metricLabelsIndex.ingressUpstreamPeers[key], newPeers)
	cnf.metricLabelsIndex.ingressUpstreamPeers[key] = newPeersIPs
	cnf.latencyCollector.UpdateUpstreamServerPeerLabels(upstreamServerPeerLabels)
	cnf.latencyCollector.DeleteUpstreamServerPeerLabels(removedPeers)
	cnf.latencyCollector.DeleteMetrics(removedPeers)

	if cnf.isPlus {
		cnf.labelUpdater.UpdateUpstreamServerLabels(upstreamServerLabels)
		cnf.labelUpdater.DeleteUpstreamServerLabels(removedUpstreams)
		cnf.labelUpdater.UpdateUpstreamServerPeerLabels(upstreamServerPeerLabels)
		cnf.labelUpdater.DeleteUpstreamServerPeerLabels(removedPeers)
		serverZoneLabels := make(map[string][]string)
		newZones := make(map[string]bool)
		var newZonesNames []string
		for _, rule := range ingEx.Ingress.Spec.Rules {
			serverZoneLabels[rule.Host] = []string{"ingress", ingEx.Ingress.Name, ingEx.Ingress.Namespace}
			newZones[rule.Host] = true
			newZonesNames = append(newZonesNames, rule.Host)
		}

		removedZones := findRemovedKeys(cnf.metricLabelsIndex.ingressServerZones[key], newZones)
		cnf.metricLabelsIndex.ingressServerZones[key] = newZonesNames
		cnf.labelUpdater.UpdateServerZoneLabels(serverZoneLabels)
		cnf.labelUpdater.DeleteServerZoneLabels(removedZones)
	}
}

func (cnf *Configurator) deleteIngressMetricsLabels(key string) {
	cnf.latencyCollector.DeleteUpstreamServerLabels(cnf.metricLabelsIndex.ingressUpstreams[key])
	cnf.latencyCollector.DeleteUpstreamServerPeerLabels(cnf.metricLabelsIndex.ingressUpstreamPeers[key])
	cnf.latencyCollector.DeleteMetrics(cnf.metricLabelsIndex.ingressUpstreamPeers[key])

	if cnf.isPlus {
		cnf.labelUpdater.DeleteUpstreamServerLabels(cnf.metricLabelsIndex.ingressUpstreams[key])
		cnf.labelUpdater.DeleteServerZoneLabels(cnf.metricLabelsIndex.ingressServerZones[key])
		cnf.labelUpdater.DeleteUpstreamServerPeerLabels(cnf.metricLabelsIndex.ingressUpstreamPeers[key])
	}

	delete(cnf.metricLabelsIndex.ingressUpstreams, key)
	delete(cnf.metricLabelsIndex.ingressServerZones, key)
	delete(cnf.metricLabelsIndex.ingressUpstreamPeers, key)
}

// AddOrUpdateIngress adds or updates NGINX configuration for the Ingress resource.
func (cnf *Configurator) AddOrUpdateIngress(ingEx *IngressEx) (Warnings, error) {
	warnings, err := cnf.addOrUpdateIngress(ingEx)
	if err != nil {
		return warnings, fmt.Errorf("Error adding or updating ingress %v/%v: %v", ingEx.Ingress.Namespace, ingEx.Ingress.Name, err)
	}

	if err := cnf.nginxManager.Reload(nginx.ReloadForOtherUpdate); err != nil {
		return warnings, fmt.Errorf("Error reloading NGINX for %v/%v: %v", ingEx.Ingress.Namespace, ingEx.Ingress.Name, err)
	}

	return warnings, nil
}

func (cnf *Configurator) addOrUpdateIngress(ingEx *IngressEx) (Warnings, error) {
	apResources := cnf.updateApResources(ingEx)

	if jwtKey, exists := ingEx.Ingress.Annotations[JWTKeyAnnotation]; exists {
		// LocalSecretStore will not set Path if the secret is not on the filesystem.
		// However, NGINX configuration for an Ingress resource, to handle the case of a missing secret,
		// relies on the path to be always configured.
		ingEx.SecretRefs[jwtKey].Path = cnf.nginxManager.GetFilenameForSecret(ingEx.Ingress.Namespace + "-" + jwtKey)
	}

	isMinion := false
	nginxCfg, warnings := generateNginxCfg(ingEx, apResources, isMinion, cnf.cfgParams, cnf.isPlus, cnf.IsResolverConfigured(),
		cnf.staticCfgParams, cnf.isWildcardEnabled)
	name := objectMetaToFileName(&ingEx.Ingress.ObjectMeta)
	content, err := cnf.templateExecutor.ExecuteIngressConfigTemplate(&nginxCfg)
	if err != nil {
		return warnings, fmt.Errorf("Error generating Ingress Config %v: %v", name, err)
	}
	cnf.nginxManager.CreateConfig(name, content)

	cnf.ingresses[name] = ingEx
	if (cnf.isPlus && cnf.isPrometheusEnabled) || cnf.isLatencyMetricsEnabled {
		cnf.updateIngressMetricsLabels(ingEx, nginxCfg.Upstreams)
	}
	return warnings, nil
}

// AddOrUpdateMergeableIngress adds or updates NGINX configuration for the Ingress resources with Mergeable Types.
func (cnf *Configurator) AddOrUpdateMergeableIngress(mergeableIngs *MergeableIngresses) (Warnings, error) {
	warnings, err := cnf.addOrUpdateMergeableIngress(mergeableIngs)
	if err != nil {
		return warnings, fmt.Errorf("Error when adding or updating ingress %v/%v: %v", mergeableIngs.Master.Ingress.Namespace, mergeableIngs.Master.Ingress.Name, err)
	}

	if err := cnf.nginxManager.Reload(nginx.ReloadForOtherUpdate); err != nil {
		return warnings, fmt.Errorf("Error reloading NGINX for %v/%v: %v", mergeableIngs.Master.Ingress.Namespace, mergeableIngs.Master.Ingress.Name, err)
	}

	return warnings, nil
}

func (cnf *Configurator) addOrUpdateMergeableIngress(mergeableIngs *MergeableIngresses) (Warnings, error) {
	masterApResources := cnf.updateApResources(mergeableIngs.Master)

	// LocalSecretStore will not set Path if the secret is not on the filesystem.
	// However, NGINX configuration for an Ingress resource, to handle the case of a missing secret,
	// relies on the path to be always configured.
	if jwtKey, exists := mergeableIngs.Master.Ingress.Annotations[JWTKeyAnnotation]; exists {
		mergeableIngs.Master.SecretRefs[jwtKey].Path = cnf.nginxManager.GetFilenameForSecret(mergeableIngs.Master.Ingress.Namespace + "-" + jwtKey)
	}
	for _, minion := range mergeableIngs.Minions {
		if jwtKey, exists := minion.Ingress.Annotations[JWTKeyAnnotation]; exists {
			minion.SecretRefs[jwtKey].Path = cnf.nginxManager.GetFilenameForSecret(minion.Ingress.Namespace + "-" + jwtKey)
		}
	}

	nginxCfg, warnings := generateNginxCfgForMergeableIngresses(mergeableIngs, masterApResources, cnf.cfgParams, cnf.isPlus,
		cnf.IsResolverConfigured(), cnf.staticCfgParams, cnf.isWildcardEnabled)

	name := objectMetaToFileName(&mergeableIngs.Master.Ingress.ObjectMeta)
	content, err := cnf.templateExecutor.ExecuteIngressConfigTemplate(&nginxCfg)
	if err != nil {
		return warnings, fmt.Errorf("Error generating Ingress Config %v: %v", name, err)
	}
	cnf.nginxManager.CreateConfig(name, content)

	cnf.ingresses[name] = mergeableIngs.Master
	cnf.minions[name] = make(map[string]bool)
	for _, minion := range mergeableIngs.Minions {
		minionName := objectMetaToFileName(&minion.Ingress.ObjectMeta)
		cnf.minions[name][minionName] = true
	}
	if (cnf.isPlus && cnf.isPrometheusEnabled) || cnf.isLatencyMetricsEnabled {
		cnf.updateIngressMetricsLabels(mergeableIngs.Master, nginxCfg.Upstreams)
	}

	return warnings, nil
}

func (cnf *Configurator) updateVirtualServerMetricsLabels(virtualServerEx *VirtualServerEx, upstreams []version2.Upstream) {
	labels := make(map[string][]string)
	newUpstreams := make(map[string]bool)
	var newUpstreamsNames []string

	upstreamServerPeerLabels := make(map[string][]string)
	newPeers := make(map[string]bool)
	var newPeersIPs []string

	for _, u := range upstreams {
		labels[u.Name] = []string{u.UpstreamLabels.Service, u.UpstreamLabels.ResourceType, u.UpstreamLabels.ResourceName, u.UpstreamLabels.ResourceNamespace}
		newUpstreams[u.Name] = true
		newUpstreamsNames = append(newUpstreamsNames, u.Name)
		for _, server := range u.Servers {
			podInfo := virtualServerEx.PodsByIP[server.Address]
			labelKey := fmt.Sprintf("%v/%v", u.Name, server.Address)
			upstreamServerPeerLabels[labelKey] = []string{podInfo.Name}
			if cnf.staticCfgParams.NginxServiceMesh {
				ownerLabelVal := fmt.Sprintf("%s/%s", podInfo.OwnerType, podInfo.OwnerName)
				upstreamServerPeerLabels[labelKey] = append(upstreamServerPeerLabels[labelKey], ownerLabelVal)
			}
			newPeers[labelKey] = true
			newPeersIPs = append(newPeersIPs, labelKey)
		}
	}

	key := fmt.Sprintf("%v/%v", virtualServerEx.VirtualServer.Namespace, virtualServerEx.VirtualServer.Name)

	removedPeers := findRemovedKeys(cnf.metricLabelsIndex.virtualServerUpstreamPeers[key], newPeers)
	cnf.metricLabelsIndex.virtualServerUpstreamPeers[key] = newPeersIPs
	cnf.latencyCollector.UpdateUpstreamServerPeerLabels(upstreamServerPeerLabels)
	cnf.latencyCollector.DeleteUpstreamServerPeerLabels(removedPeers)

	removedUpstreams := findRemovedKeys(cnf.metricLabelsIndex.virtualServerUpstreams[key], newUpstreams)
	cnf.latencyCollector.UpdateUpstreamServerLabels(labels)
	cnf.metricLabelsIndex.virtualServerUpstreams[key] = newUpstreamsNames

	cnf.latencyCollector.DeleteUpstreamServerLabels(removedUpstreams)
	cnf.latencyCollector.DeleteMetrics(removedPeers)

	if cnf.isPlus {
		cnf.labelUpdater.UpdateUpstreamServerPeerLabels(upstreamServerPeerLabels)
		cnf.labelUpdater.DeleteUpstreamServerPeerLabels(removedPeers)
		cnf.labelUpdater.UpdateUpstreamServerLabels(labels)
		cnf.labelUpdater.DeleteUpstreamServerLabels(removedUpstreams)

		serverZoneLabels := make(map[string][]string)
		newZones := make(map[string]bool)
		newZonesNames := []string{virtualServerEx.VirtualServer.Spec.Host}

		serverZoneLabels[virtualServerEx.VirtualServer.Spec.Host] = []string{
			"virtualserver", virtualServerEx.VirtualServer.Name, virtualServerEx.VirtualServer.Namespace,
		}

		newZones[virtualServerEx.VirtualServer.Spec.Host] = true

		removedZones := findRemovedKeys(cnf.metricLabelsIndex.virtualServerServerZones[key], newZones)
		cnf.metricLabelsIndex.virtualServerServerZones[key] = newZonesNames
		cnf.labelUpdater.UpdateServerZoneLabels(serverZoneLabels)
		cnf.labelUpdater.DeleteServerZoneLabels(removedZones)
	}
}

func (cnf *Configurator) deleteVirtualServerMetricsLabels(key string) {
	cnf.latencyCollector.DeleteUpstreamServerLabels(cnf.metricLabelsIndex.virtualServerUpstreams[key])
	cnf.latencyCollector.DeleteUpstreamServerPeerLabels(cnf.metricLabelsIndex.virtualServerUpstreamPeers[key])
	cnf.latencyCollector.DeleteMetrics(cnf.metricLabelsIndex.virtualServerUpstreamPeers[key])

	if cnf.isPlus {
		cnf.labelUpdater.DeleteUpstreamServerLabels(cnf.metricLabelsIndex.virtualServerUpstreams[key])
		cnf.labelUpdater.DeleteServerZoneLabels(cnf.metricLabelsIndex.virtualServerServerZones[key])
		cnf.labelUpdater.DeleteUpstreamServerPeerLabels(cnf.metricLabelsIndex.virtualServerUpstreamPeers[key])
	}

	delete(cnf.metricLabelsIndex.virtualServerUpstreams, key)
	delete(cnf.metricLabelsIndex.virtualServerServerZones, key)
	delete(cnf.metricLabelsIndex.virtualServerUpstreamPeers, key)
}

// AddOrUpdateVirtualServer adds or updates NGINX configuration for the VirtualServer resource.
func (cnf *Configurator) AddOrUpdateVirtualServer(virtualServerEx *VirtualServerEx) (Warnings, error) {
	warnings, err := cnf.addOrUpdateVirtualServer(virtualServerEx)
	if err != nil {
		return warnings, fmt.Errorf("Error adding or updating VirtualServer %v/%v: %v", virtualServerEx.VirtualServer.Namespace, virtualServerEx.VirtualServer.Name, err)
	}

	if err := cnf.nginxManager.Reload(nginx.ReloadForOtherUpdate); err != nil {
		return warnings, fmt.Errorf("Error reloading NGINX for VirtualServer %v/%v: %v", virtualServerEx.VirtualServer.Namespace, virtualServerEx.VirtualServer.Name, err)
	}

	return warnings, nil
}

func (cnf *Configurator) addOrUpdateOpenTracingTracerConfig(content string) error {
	err := cnf.nginxManager.CreateOpenTracingTracerConfig(content)
	return err
}

func (cnf *Configurator) addOrUpdateVirtualServer(virtualServerEx *VirtualServerEx) (Warnings, error) {
	name := getFileNameForVirtualServer(virtualServerEx.VirtualServer)

	vsc := newVirtualServerConfigurator(cnf.cfgParams, cnf.isPlus, cnf.IsResolverConfigured(), cnf.staticCfgParams)
	vsCfg, warnings := vsc.GenerateVirtualServerConfig(virtualServerEx)
	content, err := cnf.templateExecutorV2.ExecuteVirtualServerTemplate(&vsCfg)
	if err != nil {
		return warnings, fmt.Errorf("Error generating VirtualServer config: %v: %v", name, err)
	}
	cnf.nginxManager.CreateConfig(name, content)

	cnf.virtualServers[name] = virtualServerEx

	if (cnf.isPlus && cnf.isPrometheusEnabled) || cnf.isLatencyMetricsEnabled {
		cnf.updateVirtualServerMetricsLabels(virtualServerEx, vsCfg.Upstreams)
	}
	return warnings, nil
}

// AddOrUpdateVirtualServers adds or updates NGINX configuration for multiple VirtualServer resources.
func (cnf *Configurator) AddOrUpdateVirtualServers(virtualServerExes []*VirtualServerEx) (Warnings, error) {
	allWarnings := newWarnings()

	for _, vsEx := range virtualServerExes {
		warnings, err := cnf.addOrUpdateVirtualServer(vsEx)
		if err != nil {
			return allWarnings, err
		}
		allWarnings.Add(warnings)
	}

	if err := cnf.nginxManager.Reload(nginx.ReloadForOtherUpdate); err != nil {
		return allWarnings, fmt.Errorf("Error when reloading NGINX when updating Policy: %v", err)
	}

	return allWarnings, nil
}

func (cnf *Configurator) updateTransportServerMetricsLabels(transportServerEx *TransportServerEx, upstreams []version2.StreamUpstream) {
	labels := make(map[string][]string)
	newUpstreams := make(map[string]bool)
	var newUpstreamsNames []string

	upstreamServerPeerLabels := make(map[string][]string)
	newPeers := make(map[string]bool)
	var newPeersIPs []string

	for _, u := range upstreams {
		labels[u.Name] = []string{u.UpstreamLabels.Service, u.UpstreamLabels.ResourceType, u.UpstreamLabels.ResourceName, u.UpstreamLabels.ResourceNamespace}
		newUpstreams[u.Name] = true
		newUpstreamsNames = append(newUpstreamsNames, u.Name)

		for _, server := range u.Servers {
			podName := transportServerEx.PodsByIP[server.Address]
			labelKey := fmt.Sprintf("%v/%v", u.Name, server.Address)
			upstreamServerPeerLabels[labelKey] = []string{podName}

			newPeers[labelKey] = true
			newPeersIPs = append(newPeersIPs, labelKey)
		}
	}

	key := fmt.Sprintf("%v/%v", transportServerEx.TransportServer.Namespace, transportServerEx.TransportServer.Name)

	removedPeers := findRemovedKeys(cnf.metricLabelsIndex.transportServerUpstreamPeers[key], newPeers)
	cnf.metricLabelsIndex.transportServerUpstreamPeers[key] = newPeersIPs

	removedUpstreams := findRemovedKeys(cnf.metricLabelsIndex.transportServerUpstreams[key], newUpstreams)
	cnf.metricLabelsIndex.transportServerUpstreams[key] = newUpstreamsNames
	cnf.labelUpdater.UpdateStreamUpstreamServerPeerLabels(upstreamServerPeerLabels)
	cnf.labelUpdater.DeleteStreamUpstreamServerPeerLabels(removedPeers)
	cnf.labelUpdater.UpdateStreamUpstreamServerLabels(labels)
	cnf.labelUpdater.DeleteStreamUpstreamServerLabels(removedUpstreams)

	streamServerZoneLabels := make(map[string][]string)
	newZones := make(map[string]bool)
	zoneName := transportServerEx.TransportServer.Spec.Listener.Name

	if transportServerEx.TransportServer.Spec.Host != "" {
		zoneName = transportServerEx.TransportServer.Spec.Host
	}

	newZonesNames := []string{zoneName}

	streamServerZoneLabels[zoneName] = []string{
		"transportserver", transportServerEx.TransportServer.Name, transportServerEx.TransportServer.Namespace,
	}

	newZones[zoneName] = true
	removedZones := findRemovedKeys(cnf.metricLabelsIndex.transportServerServerZones[key], newZones)
	cnf.metricLabelsIndex.transportServerServerZones[key] = newZonesNames
	cnf.labelUpdater.UpdateStreamServerZoneLabels(streamServerZoneLabels)
	cnf.labelUpdater.DeleteStreamServerZoneLabels(removedZones)
}

func (cnf *Configurator) deleteTransportServerMetricsLabels(key string) {
	cnf.labelUpdater.DeleteStreamUpstreamServerLabels(cnf.metricLabelsIndex.transportServerUpstreams[key])
	cnf.labelUpdater.DeleteStreamServerZoneLabels(cnf.metricLabelsIndex.transportServerServerZones[key])
	cnf.labelUpdater.DeleteStreamUpstreamServerPeerLabels(cnf.metricLabelsIndex.transportServerUpstreamPeers[key])

	delete(cnf.metricLabelsIndex.transportServerUpstreams, key)
	delete(cnf.metricLabelsIndex.transportServerServerZones, key)
	delete(cnf.metricLabelsIndex.transportServerUpstreamPeers, key)
}

// AddOrUpdateTransportServer adds or updates NGINX configuration for the TransportServer resource.
// It is a responsibility of the caller to check that the TransportServer references an existing listener.
func (cnf *Configurator) AddOrUpdateTransportServer(transportServerEx *TransportServerEx) error {
	err := cnf.addOrUpdateTransportServer(transportServerEx)
	if err != nil {
		return fmt.Errorf("Error adding or updating TransportServer %v/%v: %v", transportServerEx.TransportServer.Namespace, transportServerEx.TransportServer.Name, err)
	}

	if err := cnf.nginxManager.Reload(nginx.ReloadForOtherUpdate); err != nil {
		return fmt.Errorf("Error reloading NGINX for TransportServer %v/%v: %v", transportServerEx.TransportServer.Namespace, transportServerEx.TransportServer.Name, err)
	}

	return nil
}

func (cnf *Configurator) addOrUpdateTransportServer(transportServerEx *TransportServerEx) error {
	name := getFileNameForTransportServer(transportServerEx.TransportServer)

	listener := cnf.globalCfgParams.Listeners[transportServerEx.TransportServer.Spec.Listener.Name]
	tsCfg := generateTransportServerConfig(transportServerEx, listener.Port, cnf.isPlus)

	content, err := cnf.templateExecutorV2.ExecuteTransportServerTemplate(&tsCfg)
	if err != nil {
		return fmt.Errorf("Error generating TransportServer config %v: %v", name, err)
	}

	if cnf.isPlus && cnf.isPrometheusEnabled {
		cnf.updateTransportServerMetricsLabels(transportServerEx, tsCfg.Upstreams)
	}

	cnf.nginxManager.CreateStreamConfig(name, content)

	// update TLS Passthrough Hosts config in case we have a TLS Passthrough TransportServer
	// only TLS Passthrough TransportServers have non-empty hosts
	if transportServerEx.TransportServer.Spec.Host != "" {
		key := generateNamespaceNameKey(&transportServerEx.TransportServer.ObjectMeta)
		cnf.tlsPassthroughPairs[key] = tlsPassthroughPair{
			Host:       transportServerEx.TransportServer.Spec.Host,
			UnixSocket: generateUnixSocket(transportServerEx),
		}

		return cnf.updateTLSPassthroughHostsConfig()
	}

	return nil
}

// GetVirtualServerRoutesForVirtualServer returns the virtualServerRoutes that a virtualServer
// references, if that virtualServer exists
func (cnf *Configurator) GetVirtualServerRoutesForVirtualServer(key string) []*conf_v1.VirtualServerRoute {
	vsFileName := getFileNameForVirtualServerFromKey(key)
	if cnf.virtualServers[vsFileName] != nil {
		return cnf.virtualServers[vsFileName].VirtualServerRoutes
	}
	return nil
}

func (cnf *Configurator) updateTLSPassthroughHostsConfig() error {
	cfg, duplicatedHosts := generateTLSPassthroughHostsConfig(cnf.tlsPassthroughPairs)

	for _, host := range duplicatedHosts {
		glog.Warningf("host %s is used by more than one TransportServers", host)
	}

	content, err := cnf.templateExecutorV2.ExecuteTLSPassthroughHostsTemplate(cfg)
	if err != nil {
		return fmt.Errorf("Error generating config for TLS Passthrough Unix Sockets map: %v", err)
	}

	cnf.nginxManager.CreateTLSPassthroughHostsConfig(content)

	return nil
}

func generateTLSPassthroughHostsConfig(tlsPassthroughPairs map[string]tlsPassthroughPair) (*version2.TLSPassthroughHostsConfig, []string) {
	var keys []string

	for key := range tlsPassthroughPairs {
		keys = append(keys, key)
	}

	// we sort the keys of tlsPassthroughPairs so that we get the same result for the same input
	sort.Strings(keys)

	cfg := version2.TLSPassthroughHostsConfig{}
	var duplicatedHosts []string

	for _, key := range keys {
		pair := tlsPassthroughPairs[key]

		if _, exists := cfg[pair.Host]; exists {
			duplicatedHosts = append(duplicatedHosts, pair.Host)
		}

		cfg[pair.Host] = pair.UnixSocket
	}

	return &cfg, duplicatedHosts
}

func (cnf *Configurator) addOrUpdateCASecret(secret *api_v1.Secret) string {
	name := objectMetaToFileName(&secret.ObjectMeta)
	data := GenerateCAFileContent(secret)
	return cnf.nginxManager.CreateSecret(name, data, nginx.TLSSecretFileMode)
}

func (cnf *Configurator) addOrUpdateJWKSecret(secret *api_v1.Secret) string {
	name := objectMetaToFileName(&secret.ObjectMeta)
	data := secret.Data[JWTKeyKey]
	return cnf.nginxManager.CreateSecret(name, data, nginx.JWKSecretFileMode)
}

// AddOrUpdateResources adds or updates configuration for resources.
func (cnf *Configurator) AddOrUpdateResources(ingExes []*IngressEx, mergeableIngresses []*MergeableIngresses, virtualServerExes []*VirtualServerEx) (Warnings, error) {
	allWarnings := newWarnings()

	for _, ingEx := range ingExes {
		warnings, err := cnf.addOrUpdateIngress(ingEx)
		if err != nil {
			return allWarnings, fmt.Errorf("Error adding or updating ingress %v/%v: %v", ingEx.Ingress.Namespace, ingEx.Ingress.Name, err)
		}
		allWarnings.Add(warnings)
	}

	for _, m := range mergeableIngresses {
		warnings, err := cnf.addOrUpdateMergeableIngress(m)
		if err != nil {
			return allWarnings, fmt.Errorf("Error adding or updating mergeableIngress %v/%v: %v", m.Master.Ingress.Namespace, m.Master.Ingress.Name, err)
		}
		allWarnings.Add(warnings)
	}

	for _, vsEx := range virtualServerExes {
		warnings, err := cnf.addOrUpdateVirtualServer(vsEx)
		if err != nil {
			return allWarnings, fmt.Errorf("Error adding or updating VirtualServer %v/%v: %v", vsEx.VirtualServer.Namespace, vsEx.VirtualServer.Name, err)
		}
		allWarnings.Add(warnings)
	}

	if err := cnf.nginxManager.Reload(nginx.ReloadForOtherUpdate); err != nil {
		return allWarnings, fmt.Errorf("Error when reloading NGINX when updating resources: %v", err)
	}

	return allWarnings, nil
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

	if err := cnf.nginxManager.Reload(nginx.ReloadForOtherUpdate); err != nil {
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

// GenerateCAFileContent generates a pem file content from the TLS secret.
func GenerateCAFileContent(secret *api_v1.Secret) []byte {
	var res bytes.Buffer

	res.Write(secret.Data[CAKey])

	return res.Bytes()
}

// DeleteIngress deletes NGINX configuration for the Ingress resource.
func (cnf *Configurator) DeleteIngress(key string) error {
	name := keyToFileName(key)
	cnf.nginxManager.DeleteConfig(name)

	delete(cnf.ingresses, name)
	delete(cnf.minions, name)

	if (cnf.isPlus && cnf.isPrometheusEnabled) || cnf.isLatencyMetricsEnabled {
		cnf.deleteIngressMetricsLabels(key)
	}

	if err := cnf.nginxManager.Reload(nginx.ReloadForOtherUpdate); err != nil {
		return fmt.Errorf("Error when removing ingress %v: %v", key, err)
	}

	return nil
}

// DeleteVirtualServer deletes NGINX configuration for the VirtualServer resource.
func (cnf *Configurator) DeleteVirtualServer(key string) error {
	name := getFileNameForVirtualServerFromKey(key)
	cnf.nginxManager.DeleteConfig(name)

	delete(cnf.virtualServers, name)
	if (cnf.isPlus && cnf.isPrometheusEnabled) || cnf.isLatencyMetricsEnabled {
		cnf.deleteVirtualServerMetricsLabels(fmt.Sprintf(key))
	}

	if err := cnf.nginxManager.Reload(nginx.ReloadForOtherUpdate); err != nil {
		return fmt.Errorf("Error when removing VirtualServer %v: %v", key, err)
	}

	return nil
}

// DeleteTransportServer deletes NGINX configuration for the TransportServer resource.
func (cnf *Configurator) DeleteTransportServer(key string) error {
	if cnf.isPlus && cnf.isPrometheusEnabled {
		cnf.deleteTransportServerMetricsLabels(key)
	}

	err := cnf.deleteTransportServer(key)
	if err != nil {
		return fmt.Errorf("Error when removing TransportServer %v: %v", key, err)
	}

	err = cnf.nginxManager.Reload(nginx.ReloadForOtherUpdate)
	if err != nil {
		return fmt.Errorf("Error when removing TransportServer %v: %v", key, err)
	}

	return nil
}

func (cnf *Configurator) deleteTransportServer(key string) error {
	name := getFileNameForTransportServerFromKey(key)
	cnf.nginxManager.DeleteStreamConfig(name)

	// update TLS Passthrough Hosts config in case we have a TLS Passthrough TransportServer
	if _, exists := cnf.tlsPassthroughPairs[key]; exists {
		delete(cnf.tlsPassthroughPairs, key)

		return cnf.updateTLSPassthroughHostsConfig()
	}

	return nil
}

// UpdateEndpoints updates endpoints in NGINX configuration for the Ingress resources.
func (cnf *Configurator) UpdateEndpoints(ingExes []*IngressEx) error {
	reloadPlus := false

	for _, ingEx := range ingExes {
		// It is safe to ignore warnings here as no new warnings should appear when updating Endpoints for Ingresses
		_, err := cnf.addOrUpdateIngress(ingEx)
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

	if err := cnf.nginxManager.Reload(nginx.ReloadForEndpointsUpdate); err != nil {
		return fmt.Errorf("Error reloading NGINX when updating endpoints: %v", err)
	}

	return nil
}

// UpdateEndpointsMergeableIngress updates endpoints in NGINX configuration for a mergeable Ingress resource.
func (cnf *Configurator) UpdateEndpointsMergeableIngress(mergeableIngresses []*MergeableIngresses) error {
	reloadPlus := false

	for i := range mergeableIngresses {
		// It is safe to ignore warnings here as no new warnings should appear when updating Endpoints for Ingresses
		_, err := cnf.addOrUpdateMergeableIngress(mergeableIngresses[i])
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

	if err := cnf.nginxManager.Reload(nginx.ReloadForEndpointsUpdate); err != nil {
		return fmt.Errorf("Error reloading NGINX when updating endpoints for %v: %v", mergeableIngresses, err)
	}

	return nil
}

// UpdateEndpointsForVirtualServers updates endpoints in NGINX configuration for the VirtualServer resources.
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

	if err := cnf.nginxManager.Reload(nginx.ReloadForEndpointsUpdate); err != nil {
		return fmt.Errorf("Error reloading NGINX when updating endpoints: %v", err)
	}

	return nil
}

func (cnf *Configurator) updatePlusEndpointsForVirtualServer(virtualServerEx *VirtualServerEx) error {
	upstreams := createUpstreamsForPlus(virtualServerEx, cnf.cfgParams, cnf.staticCfgParams)
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

// UpdateEndpointsForTransportServers updates endpoints in NGINX configuration for the TransportServer resources.
func (cnf *Configurator) UpdateEndpointsForTransportServers(transportServerExes []*TransportServerEx) error {
	reloadPlus := false

	for _, tsEx := range transportServerExes {
		err := cnf.addOrUpdateTransportServer(tsEx)
		if err != nil {
			return fmt.Errorf("Error adding or updating TransportServer %v/%v: %v", tsEx.TransportServer.Namespace, tsEx.TransportServer.Name, err)
		}

		if cnf.isPlus {
			err := cnf.updatePlusEndpointsForTransportServer(tsEx)
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

	if err := cnf.nginxManager.Reload(nginx.ReloadForEndpointsUpdate); err != nil {
		return fmt.Errorf("Error reloading NGINX when updating endpoints: %v", err)
	}

	return nil
}

func (cnf *Configurator) updatePlusEndpointsForTransportServer(transportServerEx *TransportServerEx) error {
	upstreamNamer := newUpstreamNamerForTransportServer(transportServerEx.TransportServer)

	for _, u := range transportServerEx.TransportServer.Spec.Upstreams {
		name := upstreamNamer.GetNameForUpstream(u.Name)

		// subselector is not supported yet in TransportServer upstreams. That's why we pass "nil" here
		endpointsKey := GenerateEndpointsKey(transportServerEx.TransportServer.Namespace, u.Service, nil, uint16(u.Port))
		endpoints := transportServerEx.Endpoints[endpointsKey]

		err := cnf.nginxManager.UpdateStreamServersInPlus(name, endpoints)
		if err != nil {
			return fmt.Errorf("Couldn't update the endpoints for %v: %v", u.Name, err)
		}
	}

	return nil
}

func (cnf *Configurator) updatePlusEndpoints(ingEx *IngressEx) error {
	ingCfg := parseAnnotations(ingEx, cnf.cfgParams, cnf.isPlus, cnf.staticCfgParams.MainAppProtectLoadModule, cnf.staticCfgParams.EnableInternalRoutes)

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
func (cnf *Configurator) UpdateConfig(cfgParams *ConfigParams, ingExes []*IngressEx, mergeableIngs []*MergeableIngresses, virtualServerExes []*VirtualServerEx) (Warnings, error) {
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

	if cfgParams.VirtualServerTemplate != nil {
		err := cnf.templateExecutorV2.UpdateVirtualServerTemplate(cfgParams.VirtualServerTemplate)
		if err != nil {
			return allWarnings, fmt.Errorf("Error when parsing the VirtualServer template: %v", err)
		}
	}

	mainCfg := GenerateNginxMainConfig(cnf.staticCfgParams, cfgParams)
	mainCfgContent, err := cnf.templateExecutor.ExecuteMainConfigTemplate(mainCfg)
	if err != nil {
		return allWarnings, fmt.Errorf("Error when writing main Config")
	}
	cnf.nginxManager.CreateMainConfig(mainCfgContent)

	for _, ingEx := range ingExes {
		warnings, err := cnf.addOrUpdateIngress(ingEx)
		if err != nil {
			return allWarnings, err
		}
		allWarnings.Add(warnings)
	}
	for _, mergeableIng := range mergeableIngs {
		warnings, err := cnf.addOrUpdateMergeableIngress(mergeableIng)
		if err != nil {
			return allWarnings, err
		}
		allWarnings.Add(warnings)
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
	if err := cnf.nginxManager.Reload(nginx.ReloadForOtherUpdate); err != nil {
		return allWarnings, fmt.Errorf("Error when updating config from ConfigMap: %v", err)
	}

	return allWarnings, nil
}

// UpdateGlobalConfiguration updates NGINX config based on the changes to the GlobalConfiguration resource.
// Currently, changes to the GlobalConfiguration only affect TransportServer resources.
// As a result of the changes, the configuration for TransportServers is updated and some TransportServers
// might be removed from NGINX.
func (cnf *Configurator) UpdateGlobalConfiguration(globalConfiguration *conf_v1alpha1.GlobalConfiguration,
	transportServerExes []*TransportServerEx) (updatedTransportServerExes []*TransportServerEx, deletedTransportServerExes []*TransportServerEx, err error) {
	cnf.globalCfgParams = ParseGlobalConfiguration(globalConfiguration, cnf.staticCfgParams.TLSPassthrough)

	for _, tsEx := range transportServerExes {
		if cnf.CheckIfListenerExists(&tsEx.TransportServer.Spec.Listener) {
			updatedTransportServerExes = append(updatedTransportServerExes, tsEx)

			err := cnf.addOrUpdateTransportServer(tsEx)
			if err != nil {
				return updatedTransportServerExes, deletedTransportServerExes, fmt.Errorf("Error when updating global configuration: %v", err)
			}

		} else {
			deletedTransportServerExes = append(deletedTransportServerExes, tsEx)
			if err != nil {
				return updatedTransportServerExes, deletedTransportServerExes, fmt.Errorf("Error when updating global configuration: %v", err)
			}
		}
	}

	if err := cnf.nginxManager.Reload(nginx.ReloadForOtherUpdate); err != nil {
		return updatedTransportServerExes, deletedTransportServerExes, fmt.Errorf("Error when updating global configuration: %v", err)
	}

	return updatedTransportServerExes, deletedTransportServerExes, nil
}

func keyToFileName(key string) string {
	return strings.Replace(key, "/", "-", -1)
}

func objectMetaToFileName(meta *meta_v1.ObjectMeta) string {
	return meta.Namespace + "-" + meta.Name
}

func generateNamespaceNameKey(objectMeta *meta_v1.ObjectMeta) string {
	return fmt.Sprintf("%s/%s", objectMeta.Namespace, objectMeta.Name)
}

func getFileNameForVirtualServer(virtualServer *conf_v1.VirtualServer) string {
	return fmt.Sprintf("vs_%s_%s", virtualServer.Namespace, virtualServer.Name)
}

func getFileNameForTransportServer(transportServer *conf_v1alpha1.TransportServer) string {
	return fmt.Sprintf("ts_%s_%s", transportServer.Namespace, transportServer.Name)
}

func getFileNameForVirtualServerFromKey(key string) string {
	replaced := strings.Replace(key, "/", "_", -1)
	return fmt.Sprintf("vs_%s", replaced)
}

func getFileNameForTransportServerFromKey(key string) string {
	replaced := strings.Replace(key, "/", "_", -1)
	return fmt.Sprintf("ts_%s", replaced)
}

// HasIngress checks if the Ingress resource is present in NGINX configuration.
func (cnf *Configurator) HasIngress(ing *networking.Ingress) bool {
	name := objectMetaToFileName(&ing.ObjectMeta)
	_, exists := cnf.ingresses[name]
	return exists
}

// HasMinion checks if the minion Ingress resource of the master is present in NGINX configuration.
func (cnf *Configurator) HasMinion(master *networking.Ingress, minion *networking.Ingress) bool {
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

func (cnf *Configurator) CheckIfListenerExists(transportServerListener *conf_v1alpha1.TransportServerListener) bool {
	listener, exists := cnf.globalCfgParams.Listeners[transportServerListener.Name]

	if !exists {
		return false
	}

	return transportServerListener.Protocol == listener.Protocol
}

// AddOrUpdateSpiffeCerts writes Spiffe certs and keys to disk and reloads NGINX
func (cnf *Configurator) AddOrUpdateSpiffeCerts(svidResponse *workload.X509SVIDs) error {
	svid := svidResponse.Default()
	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(svid.PrivateKey.(crypto.PrivateKey))
	if err != nil {
		return fmt.Errorf("error when marshaling private key: %v", err)
	}

	cnf.nginxManager.CreateSecret(spiffeKeyFileName, createSpiffeKey(privateKeyBytes), spiffeKeyFileMode)
	cnf.nginxManager.CreateSecret(spiffeCertFileName, createSpiffeCert(svid.Certificates), spiffeCertsFileMode)
	cnf.nginxManager.CreateSecret(spiffeBundleFileName, createSpiffeCert(svid.TrustBundle), spiffeCertsFileMode)

	err = cnf.nginxManager.Reload(nginx.ReloadForOtherUpdate)
	if err != nil {
		return fmt.Errorf("error when reloading NGINX when updating the SPIFFE Certs: %v", err)
	}
	return nil
}

func createSpiffeKey(content []byte) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: content,
	})
}

func createSpiffeCert(certs []*x509.Certificate) []byte {
	pemData := make([]byte, 0, len(certs))
	for _, c := range certs {
		b := &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: c.Raw,
		}
		pemData = append(pemData, pem.EncodeToMemory(b)...)
	}
	return pemData
}

func (cnf *Configurator) updateApResources(ingEx *IngressEx) map[string]string {
	apRes := make(map[string]string)
	if ingEx.AppProtectPolicy != nil {
		policyFileName := appProtectPolicyFileNameFromIngEx(ingEx)
		policyContent := generateApResourceFileContent(ingEx.AppProtectPolicy)
		cnf.nginxManager.CreateAppProtectResourceFile(policyFileName, policyContent)
		apRes[appProtectPolicyKey] = policyFileName

	}

	if ingEx.AppProtectLogConf != nil {
		logConfFileName := appProtectLogConfFileNameFromIngEx(ingEx)
		logConfContent := generateApResourceFileContent(ingEx.AppProtectLogConf)
		cnf.nginxManager.CreateAppProtectResourceFile(logConfFileName, logConfContent)
		apRes[appProtectLogConfKey] = logConfFileName + " " + ingEx.AppProtectLogDst
	}

	return apRes
}

func appProtectPolicyFileNameFromIngEx(ingEx *IngressEx) string {
	return fmt.Sprintf("%s%s_%s", appProtectPolicyFolder, ingEx.AppProtectPolicy.GetNamespace(), ingEx.AppProtectPolicy.GetName())
}

func appProtectLogConfFileNameFromIngEx(ingEx *IngressEx) string {
	return fmt.Sprintf("%s%s_%s", appProtectLogConfFolder, ingEx.AppProtectLogConf.GetNamespace(), ingEx.AppProtectLogConf.GetName())
}

func appProtectUserSigFileNameFromUnstruct(unst *unstructured.Unstructured) string {
	return fmt.Sprintf("%s%s_%s", appProtectUserSigFolder, unst.GetNamespace(), unst.GetName())
}

func generateApResourceFileContent(apResource *unstructured.Unstructured) []byte {
	// Safe to ignore errors since validation already checked those
	spec, _, _ := unstructured.NestedMap(apResource.Object, "spec")
	data, _ := json.Marshal(spec)
	return data
}

// AddOrUpdateAppProtectResource updates Ingresses that use App Protect Resources
func (cnf *Configurator) AddOrUpdateAppProtectResource(resource *unstructured.Unstructured, ingExes []*IngressEx, mergeableIngresses []*MergeableIngresses) (Warnings, error) {
	allWarnings := newWarnings()

	for _, ingEx := range ingExes {
		warnings, err := cnf.addOrUpdateIngress(ingEx)
		if err != nil {
			return allWarnings, fmt.Errorf("Error adding or updating ingress %v/%v: %v", ingEx.Ingress.Namespace, ingEx.Ingress.Name, err)
		}
		allWarnings.Add(warnings)
	}

	for _, m := range mergeableIngresses {
		warnings, err := cnf.addOrUpdateMergeableIngress(m)
		if err != nil {
			return allWarnings, fmt.Errorf("Error adding or updating mergeableIngress %v/%v: %v", m.Master.Ingress.Namespace, m.Master.Ingress.Name, err)
		}
		allWarnings.Add(warnings)
	}
	if err := cnf.nginxManager.Reload(nginx.ReloadForOtherUpdate); err != nil {
		return allWarnings, fmt.Errorf("Error when reloading NGINX when updating %v: %v", resource.GetKind(), err)
	}

	return allWarnings, nil
}

// DeleteAppProtectPolicy updates Ingresses that use AP Policy after that policy is deleted
func (cnf *Configurator) DeleteAppProtectPolicy(polNamespaceName string, ingExes []*IngressEx, mergeableIngresses []*MergeableIngresses) (Warnings, error) {
	if len(ingExes) > 0 || len(mergeableIngresses) > 0 {
		fName := strings.Replace(polNamespaceName, "/", "_", 1)
		polFileName := appProtectPolicyFolder + fName
		cnf.nginxManager.DeleteAppProtectResourceFile(polFileName)
	}

	allWarnings := newWarnings()

	for _, ingEx := range ingExes {
		warnings, err := cnf.addOrUpdateIngress(ingEx)
		if err != nil {
			return allWarnings, fmt.Errorf("Error adding or updating ingress %v/%v: %v", ingEx.Ingress.Namespace, ingEx.Ingress.Name, err)
		}
		allWarnings.Add(warnings)
	}

	for _, m := range mergeableIngresses {
		warnings, err := cnf.addOrUpdateMergeableIngress(m)
		if err != nil {
			return allWarnings, fmt.Errorf("Error adding or updating mergeableIngress %v/%v: %v", m.Master.Ingress.Namespace, m.Master.Ingress.Name, err)
		}
		allWarnings.Add(warnings)
	}

	if err := cnf.nginxManager.Reload(nginx.ReloadForOtherUpdate); err != nil {
		return allWarnings, fmt.Errorf("Error when reloading NGINX when removing App Protect Policy: %v", err)
	}

	return allWarnings, nil
}

// DeleteAppProtectLogConf updates Ingresses that use AP Log Configuration after that policy is deleted
func (cnf *Configurator) DeleteAppProtectLogConf(logConfNamespaceName string, ingExes []*IngressEx, mergeableIngresses []*MergeableIngresses) (Warnings, error) {
	if len(ingExes) > 0 || len(mergeableIngresses) > 0 {
		fName := strings.Replace(logConfNamespaceName, "/", "_", 1)
		logConfFileName := appProtectLogConfFolder + fName
		cnf.nginxManager.DeleteAppProtectResourceFile(logConfFileName)
	}
	allWarnings := newWarnings()

	for _, ingEx := range ingExes {
		warnings, err := cnf.addOrUpdateIngress(ingEx)
		if err != nil {
			return allWarnings, fmt.Errorf("Error adding or updating ingress %v/%v: %v", ingEx.Ingress.Namespace, ingEx.Ingress.Name, err)
		}
		allWarnings.Add(warnings)
	}

	for _, m := range mergeableIngresses {
		warnings, err := cnf.addOrUpdateMergeableIngress(m)
		if err != nil {
			return allWarnings, fmt.Errorf("Error adding or updating mergeableIngress %v/%v: %v", m.Master.Ingress.Namespace, m.Master.Ingress.Name, err)
		}
		allWarnings.Add(warnings)
	}

	if err := cnf.nginxManager.Reload(nginx.ReloadForOtherUpdate); err != nil {
		return allWarnings, fmt.Errorf("Error when reloading NGINX when removing App Protect Log Configuration: %v", err)
	}

	return allWarnings, nil
}

// RefreshAppProtectUserSigs writes all valid uds files to fs and reloads
func (cnf *Configurator) RefreshAppProtectUserSigs(userSigs []*unstructured.Unstructured, delPols []string, ingExes []*IngressEx, mergeableIngresses []*MergeableIngresses) (Warnings, error) {
	allWarnings := newWarnings()
	for _, ingEx := range ingExes {
		warnings, err := cnf.addOrUpdateIngress(ingEx)
		if err != nil {
			return allWarnings, fmt.Errorf("Error adding or updating ingress %v/%v: %v", ingEx.Ingress.Namespace, ingEx.Ingress.Name, err)
		}
		allWarnings.Add(warnings)
	}

	for _, m := range mergeableIngresses {
		warnings, err := cnf.addOrUpdateMergeableIngress(m)
		if err != nil {
			return allWarnings, fmt.Errorf("Error adding or updating mergeableIngress %v/%v: %v", m.Master.Ingress.Namespace, m.Master.Ingress.Name, err)
		}
		allWarnings.Add(warnings)
	}

	for _, file := range delPols {
		cnf.nginxManager.DeleteAppProtectResourceFile(file)
	}

	var builder strings.Builder
	cnf.nginxManager.ClearAppProtectFolder(appProtectUserSigFolder)
	for _, sig := range userSigs {
		fName := appProtectUserSigFileNameFromUnstruct(sig)
		data := generateApResourceFileContent(sig)
		cnf.nginxManager.CreateAppProtectResourceFile(fName, data)
		fmt.Fprintf(&builder, "app_protect_user_defined_signatures %s;\n", fName)
	}
	cnf.nginxManager.CreateAppProtectResourceFile(appProtectUserSigIndex, []byte(builder.String()))
	return allWarnings, cnf.nginxManager.Reload(nginx.ReloadForOtherUpdate)
}

// AddInternalRouteConfig adds internal route server to NGINX Configuration and
// reloads NGINX
func (cnf *Configurator) AddInternalRouteConfig() error {
	cnf.staticCfgParams.EnableInternalRoutes = true
	cnf.staticCfgParams.PodName = os.Getenv("POD_NAME")
	mainCfg := GenerateNginxMainConfig(cnf.staticCfgParams, cnf.cfgParams)
	mainCfgContent, err := cnf.templateExecutor.ExecuteMainConfigTemplate(mainCfg)
	if err != nil {
		return fmt.Errorf("Error when writing main Config: %v", err)
	}
	cnf.nginxManager.CreateMainConfig(mainCfgContent)
	if err := cnf.nginxManager.Reload(nginx.ReloadForOtherUpdate); err != nil {
		return fmt.Errorf("Error when reloading nginx: %v", err)
	}
	return nil
}

// AddOrUpdateSecret adds or updates a secret.
func (cnf *Configurator) AddOrUpdateSecret(secret *api_v1.Secret) string {
	switch secret.Type {
	case secrets.SecretTypeCA:
		return cnf.addOrUpdateCASecret(secret)
	case secrets.SecretTypeJWK:
		return cnf.addOrUpdateJWKSecret(secret)
	case secrets.SecretTypeOIDC:
		// OIDC ClientSecret is not required on the filesystem, it is written directly to the config file.
		return ""
	default:
		return cnf.addOrUpdateTLSSecret(secret)
	}
}

// DeleteSecret deletes a secret.
func (cnf *Configurator) DeleteSecret(key string) {
	cnf.nginxManager.DeleteSecret(keyToFileName(key))
}
