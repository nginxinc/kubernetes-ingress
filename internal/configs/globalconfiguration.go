package configs

import conf_v1alpha1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1alpha1"

func ParseGlobalConfiguration(gc *conf_v1alpha1.GlobalConfiguration) *GlobalConfigParams {
	gcfgParams := NewDefaultGlobalConfigParams()

	for _, l := range gc.Spec.Listeners {
		gcfgParams.Listeners[l.Name] = Listener{
			Port:     l.Port,
			Protocol: l.Protocol,
		}
	}

	return gcfgParams
}
