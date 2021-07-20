package api

// Apis provides access to the Kubernetes resources api.
type Apis struct {
	Ingresses               *Ingresses
	Endpoints               *Endpoints
	ConfigMaps              *ConfigMaps
	Pods                    *Pods
	TransportServers        *TransportServers
	Services                *Services
	VirtualServers          *VirtualServers
	VirtualServerRoutes     *VirtualServerRoutes
	AppProtectPolicies      *AppProtectPolicies
	AppProtectLogConfs      *AppProtectLogConfs
	AppProtectUserSigLister *AppProtectUserSigs
	GlobalConfigurations    *GlobalConfigurations
	Policies                *Policies
	IngressLinks            *IngressLinks
	Secrets                 *Secrets
}

func NewApis() *Apis {
	return &Apis{}
}
