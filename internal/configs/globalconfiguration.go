package configs

import conf_v1alpha1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1alpha1"

func ParseGlobalConfiguration(gc *conf_v1alpha1.GlobalConfiguration, tlsPassthrough bool) *GlobalConfigParams {
	gcfgParams := NewDefaultGlobalConfigParams()
	if tlsPassthrough {
		gcfgParams = NewGlobalConfigParamsWithTLSPassthrough()
	}

	for _, l := range gc.Spec.Listeners {
		gcfgParams.Listeners[l.Name] = Listener{
			Port:     l.Port,
			Protocol: l.Protocol,
		}
	}

	return gcfgParams
}
