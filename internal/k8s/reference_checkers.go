package k8s

import (
	"github.com/nginxinc/kubernetes-ingress/internal/configs"
	v1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"
	networking "k8s.io/api/networking/v1beta1"
)

type resourceReferenceChecker interface {
	IsReferencedByIngress(namespace string, name string, ing *networking.Ingress) bool
	IsReferencedByMinion(namespace string, name string, ing *networking.Ingress) bool
	IsReferencedByVirtualServer(namespace string, name string, vs *v1.VirtualServer) bool
	IsReferencedByVirtualServerRoute(namespace string, name string, vsr *v1.VirtualServerRoute) bool
}

type secretReferenceChecker struct {
	isPlus bool
}

func newSecretReferenceChecker(isPlus bool) *secretReferenceChecker {
	return &secretReferenceChecker{isPlus}
}

func (rc *secretReferenceChecker) IsReferencedByIngress(secretNamespace string, secretName string, ing *networking.Ingress) bool {
	if ing.Namespace != secretNamespace {
		return false
	}

	for _, tls := range ing.Spec.TLS {
		if tls.SecretName == secretName {
			return true
		}
	}

	if rc.isPlus {
		if jwtKey, exists := ing.Annotations[configs.JWTKeyAnnotation]; exists {
			if jwtKey == secretName {
				return true
			}
		}
	}

	return false
}

func (rc *secretReferenceChecker) IsReferencedByMinion(secretNamespace string, secretName string, ing *networking.Ingress) bool {
	if ing.Namespace != secretNamespace {
		return false
	}

	if rc.isPlus {
		if jwtKey, exists := ing.Annotations[configs.JWTKeyAnnotation]; exists {
			if jwtKey == secretName {
				return true
			}
		}
	}

	return false
}

func (rc *secretReferenceChecker) IsReferencedByVirtualServer(secretNamespace string, secretName string, vs *v1.VirtualServer) bool {
	if vs.Namespace != secretNamespace {
		return false
	}

	if vs.Spec.TLS != nil && vs.Spec.TLS.Secret == secretName {
		return true
	}

	return false
}

func (rc *secretReferenceChecker) IsReferencedByVirtualServerRoute(secretNamespace string, secretName string, vsr *v1.VirtualServerRoute) bool {
	return false
}

type serviceReferenceChecker struct{}

func newServiceReferenceChecker() *serviceReferenceChecker {
	return &serviceReferenceChecker{}
}

func (rc *serviceReferenceChecker) IsReferencedByIngress(svcNamespace string, svcName string, ing *networking.Ingress) bool {
	if ing.Namespace != svcNamespace {
		return false
	}

	if ing.Spec.Backend != nil {
		if ing.Spec.Backend.ServiceName == svcName {
			return true
		}
	}
	for _, rules := range ing.Spec.Rules {
		if rules.IngressRuleValue.HTTP == nil {
			continue
		}
		for _, p := range rules.IngressRuleValue.HTTP.Paths {
			if p.Backend.ServiceName == svcName {
				return true
			}
		}
	}

	return false
}

func (rc *serviceReferenceChecker) IsReferencedByMinion(svcNamespace string, svcName string, ing *networking.Ingress) bool {
	return rc.IsReferencedByIngress(svcNamespace, svcName, ing)
}

func (rc *serviceReferenceChecker) IsReferencedByVirtualServer(svcNamespace string, svcName string, vs *v1.VirtualServer) bool {
	if vs.Namespace != svcNamespace {
		return false
	}

	for _, u := range vs.Spec.Upstreams {
		if u.Service == svcName {
			return true
		}
	}

	return false
}

func (rc *serviceReferenceChecker) IsReferencedByVirtualServerRoute(svcNamespace string, svcName string, vsr *v1.VirtualServerRoute) bool {
	if vsr.Namespace != svcNamespace {
		return false
	}

	for _, u := range vsr.Spec.Upstreams {
		if u.Service == svcName {
			return true
		}
	}

	return false
}

type policyReferenceChecker struct {
}

func newPolicyReferenceChecker() *policyReferenceChecker {
	return &policyReferenceChecker{}
}

func (rc *policyReferenceChecker) IsReferencedByIngress(policyNamespace string, policyName string, ing *networking.Ingress) bool {
	return false
}

func (rc *policyReferenceChecker) IsReferencedByMinion(policyNamespace string, policyName string, ing *networking.Ingress) bool {
	return false
}

func (rc *policyReferenceChecker) IsReferencedByVirtualServer(policyNamespace string, policyName string, vs *v1.VirtualServer) bool {
	if isPolicyReferenced(vs.Spec.Policies, vs.Namespace, policyNamespace, policyName) {
		return true
	}

	for _, r := range vs.Spec.Routes {
		if isPolicyReferenced(r.Policies, vs.Namespace, policyNamespace, policyName) {
			return true
		}
	}

	return false
}

func (rc *policyReferenceChecker) IsReferencedByVirtualServerRoute(policyNamespace string, policyName string, vsr *v1.VirtualServerRoute) bool {
	for _, r := range vsr.Spec.Subroutes {
		if isPolicyReferenced(r.Policies, vsr.Namespace, policyNamespace, policyName) {
			return true
		}
	}

	return false
}

// appProtectResourceReferenceChecker is a reference checker for AppProtect related resources.
// Only Regular/Master Ingress can reference those resources.
type appProtectResourceReferenceChecker struct {
	annotation string
}

func newAppProtectResourceReferenceChecker(annotation string) *appProtectResourceReferenceChecker {
	return &appProtectResourceReferenceChecker{annotation}
}

func (rc *appProtectResourceReferenceChecker) IsReferencedByIngress(namespace string, name string, ing *networking.Ingress) bool {
	if pol, exists := ing.Annotations[rc.annotation]; exists {
		if pol == namespace+"/"+name || (namespace == ing.Namespace && pol == name) {
			return true
		}
	}

	return false
}

func (rc *appProtectResourceReferenceChecker) IsReferencedByMinion(namespace string, name string, ing *networking.Ingress) bool {
	return false
}

func (rc *appProtectResourceReferenceChecker) IsReferencedByVirtualServer(namespace string, name string, vs *v1.VirtualServer) bool {
	return false
}

func (rc *appProtectResourceReferenceChecker) IsReferencedByVirtualServerRoute(namespace string, name string, vsr *v1.VirtualServerRoute) bool {
	return false
}

func isPolicyReferenced(policies []v1.PolicyReference, resourceNamespace string, policyNamespace string, policyName string) bool {
	for _, p := range policies {
		namespace := p.Namespace
		if namespace == "" {
			namespace = resourceNamespace
		}

		if p.Name == policyName && namespace == policyNamespace {
			return true
		}
	}

	return false
}
