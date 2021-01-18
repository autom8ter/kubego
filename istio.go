package kubego

import (
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	istio "istio.io/client-go/pkg/clientset/versioned"
	"istio.io/client-go/pkg/clientset/versioned/typed/networking/v1alpha3"
	"istio.io/client-go/pkg/clientset/versioned/typed/security/v1beta1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"path/filepath"
)

// Istio is an istio client
type Istio struct {
	clientset *istio.Clientset
}

// NewInClusterClient returns a client for use when inside the kubernetes cluster
func NewInClusterIstioClient() (*Istio, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get in cluster config")
	}
	ic, err := istio.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get in cluster istio clientset")
	}
	return &Istio{
		clientset: ic,
	}, nil
}

// NewOutOfClusterIstioClient returns an istio client for use when not inside the kubernetes cluster
func NewOutOfClusterIstioClient() (*Istio, error) {
	dir, _ := homedir.Dir()
	kubeconfig := filepath.Join(dir, ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get out of cluster config")
	}
	ic, err := istio.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get out of cluster istio clientset")
	}
	return &Istio{
		clientset: ic,
	}, nil
}

// VirtualServices is a VirtualServices client. ref: https://istio.io/latest/docs/reference/config/networking/virtual-service/
func (c *Istio) VirtualServices(namespace string) v1alpha3.VirtualServiceInterface {
	return c.clientset.NetworkingV1alpha3().VirtualServices(namespace)
}

// Gateways is a Istio Gateways client. ref: https://istio.io/latest/docs/reference/config/networking/gateway/
func (c *Istio) Gateways(namespace string) v1alpha3.GatewayInterface {
	return c.clientset.NetworkingV1alpha3().Gateways(namespace)
}

// Istio WorkloadEntries client. ref: https://istio.io/latest/docs/reference/config/networking/workload-entry/
func (c *Istio) WorkloadEntries(namespace string) v1alpha3.WorkloadEntryInterface {
	return c.clientset.NetworkingV1alpha3().WorkloadEntries(namespace)
}

// Istio WorkloadGroups client. ref: https://istio.io/latest/docs/reference/config/networking/workload-group/
func (c *Istio) WorkloadGroups(namespace string) v1alpha3.WorkloadGroupInterface {
	return c.clientset.NetworkingV1alpha3().WorkloadGroups(namespace)
}

// Istio DestinationRules client. ref: https://istio.io/latest/docs/reference/config/networking/destination-rule/
func (c *Istio) DestinationRules(namespace string) v1alpha3.DestinationRuleInterface {
	return c.clientset.NetworkingV1alpha3().DestinationRules(namespace)
}

// Istio Sidecars client. ref: https://istio.io/latest/docs/reference/config/networking/sidecar/
func (c *Istio) Sidecars(namespace string) v1alpha3.SidecarInterface {
	return c.clientset.NetworkingV1alpha3().Sidecars(namespace)
}

// Istio EnvoyFilters client. ref: https://istio.io/latest/docs/reference/config/networking/envoy-filter/
func (c *Istio) EnvoyFilters(namespace string) v1alpha3.EnvoyFilterInterface {
	return c.clientset.NetworkingV1alpha3().EnvoyFilters(namespace)
}

// Istio Request Authentication client. ref: https://istio.io/latest/docs/reference/config/networking/service-entry/
func (c *Istio) ServiceEntries(namespace string) v1alpha3.ServiceEntryInterface {
	return c.clientset.NetworkingV1alpha3().ServiceEntries(namespace)
}

// Istio AuthorizationPolicies client. ref: https://istio.io/latest/docs/reference/config/security/authorization-policy/
func (c *Istio) AuthorizationPolicies(namespace string) v1beta1.AuthorizationPolicyInterface {
	return c.clientset.SecurityV1beta1().AuthorizationPolicies(namespace)
}

// Istio PeerAuthentications client. ref: https://istio.io/latest/docs/reference/config/security/peer_authentication/
func (c *Istio) PeerAuthentications(namespace string) v1beta1.PeerAuthenticationInterface {
	return c.clientset.SecurityV1beta1().PeerAuthentications(namespace)
}

// Istio Request Authentication client. ref: https://istio.io/latest/docs/reference/config/security/request_authentication/
func (c *Istio) RequestAuthentications(namespace string) v1beta1.RequestAuthenticationInterface {
	return c.clientset.SecurityV1beta1().RequestAuthentications(namespace)
}
