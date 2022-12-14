package ingress

import (
	securityv1 "github.com/Layer7-Community/layer7-operator/api/v1"
	"github.com/Layer7-Community/layer7-operator/pkg/gateway/util"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewIngress(gw *securityv1.Gateway) *networkingv1.Ingress {
	tls := gw.Spec.App.Ingress.TLS
	rules := []networkingv1.IngressRule{}
	ingressClassName := gw.Spec.App.Ingress.IngressClassName
	annotations := gw.Spec.App.Ingress.Annotations

	var portName string
	pathTypePrefix := networkingv1.PathTypePrefix
	for _, s := range gw.Spec.App.Service.Ports {
		if s.Port == 8443 {
			portName = s.Name
		}
	}

	for _, r := range gw.Spec.App.Ingress.Rules {
		rule := networkingv1.IngressRule{
			Host: r.Host,
		}
		paths := []networkingv1.HTTPIngressPath{}
		path := networkingv1.HTTPIngressPath{
			Path:     "/",
			PathType: &pathTypePrefix,
			Backend: networkingv1.IngressBackend{
				Service: &networkingv1.IngressServiceBackend{
					Name: gw.Name,
					Port: networkingv1.ServiceBackendPort{
						Name: portName,
					},
				},
			},
		}
		paths = append(paths, path)

		rule.HTTP = &networkingv1.HTTPIngressRuleValue{
			Paths: paths,
		}

		rules = append(rules, rule)
	}

	ls := util.DefaultLabels(gw)
	service := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        gw.Name,
			Namespace:   gw.Namespace,
			Annotations: annotations,
			Labels:      ls,
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Ingress",
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: &ingressClassName,
			TLS:              tls,
			Rules:            rules,
		},
	}
	return service
}
