package kubego

import (
	"context"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"io"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	v13 "k8s.io/client-go/kubernetes/typed/batch/v1"
	"k8s.io/client-go/kubernetes/typed/batch/v1beta1"
	v12 "k8s.io/client-go/kubernetes/typed/core/v1"
	v14 "k8s.io/client-go/kubernetes/typed/networking/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"path/filepath"
)

// Func is a function that executes against a Client
type Func func(c *Client) error

// Client is a kubernetes client
type Client struct {
	clientset *kubernetes.Clientset
}

// NewInClusterClient returns a client for use when inside the kubernetes cluster
func NewInClusterClient() (*Client, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get in cluster config")
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get in cluster clientset")
	}
	return &Client{
		clientset: clientset,
	}, nil
}

// NewOutOfClusterClient returns a client for use when not inside the kubernetes cluster
func NewOutOfClusterClient() (*Client, error) {
	dir, _ := homedir.Dir()
	kubeconfig := filepath.Join(dir, ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get in cluster config")
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get in cluster clientset")
	}
	return &Client{
		clientset: clientset,
	}, nil
}

// Pods returns an interface for managing k8s pods
func (p *Client) Pods(namespace string) v12.PodInterface {
	return p.clientset.CoreV1().Pods(namespace)
}

// Services returns an interface for managing k8s services
func (p *Client) Services(namespace string) v12.ServiceInterface {
	return p.clientset.CoreV1().Services(namespace)
}

// Namespaces returns an interface for managing k8s namespaces
func (p *Client) Namespaces() v12.NamespaceInterface {
	return p.clientset.CoreV1().Namespaces()
}

// ConfigMaps returns an interface for managing k8s config maps
func (p *Client) ConfigMaps(namespace string) v12.ConfigMapInterface {
	return p.clientset.CoreV1().ConfigMaps(namespace)
}

// Nodes returns an interface for managing k8s nodes
func (p *Client) Nodes() v12.NodeInterface {
	return p.clientset.CoreV1().Nodes()
}

// PersistentVolumeClaims returns an interface for managing k8s persistant volume claims
func (p *Client) PersistentVolumeClaims(namespace string) v12.PersistentVolumeClaimInterface {
	return p.clientset.CoreV1().PersistentVolumeClaims(namespace)
}

// PersistentVolumes returns an interface for managing k8s persistant volumes
func (p *Client) PersistentVolumes() v12.PersistentVolumeInterface {
	return p.clientset.CoreV1().PersistentVolumes()
}

// Secrets returns an interface for managing k8s secrets
func (p *Client) Secrets(namespace string) v12.SecretInterface {
	return p.clientset.CoreV1().Secrets(namespace)
}

// ServiceAccounts returns an interface for managing k8s service accounts
func (p *Client) ServiceAccounts(namespace string) v12.ServiceAccountInterface {
	return p.clientset.CoreV1().ServiceAccounts(namespace)
}

// Endpoints returns an interface for managing k8s endpoints
func (p *Client) Endpoints(namespace string) v12.EndpointsInterface {
	return p.clientset.CoreV1().Endpoints(namespace)
}

// Events returns an interface for managing k8s events
func (p *Client) Events(namespace string) v12.EventInterface {
	return p.clientset.CoreV1().Events(namespace)
}

// ResourceQuotas returns an interface for managing k8s resource quotas
func (p *Client) ResourceQuotas(namespace string) v12.ResourceQuotaInterface {
	return p.clientset.CoreV1().ResourceQuotas(namespace)
}

// StatefulSets returns an interface for managing k8s statefulsets
func (p *Client) StatefulSets(namespace string) v1.StatefulSetInterface {
	return p.clientset.AppsV1().StatefulSets(namespace)
}

// Deployments returns an interface for managing k8s deployments
func (p *Client) Deployments(namespace string) v1.DeploymentInterface {
	return p.clientset.AppsV1().Deployments(namespace)
}

// DaemonSets returns an interface for managing k8s daemonsets
func (p *Client) DaemonSets(namespace string) v1.DaemonSetInterface {
	return p.clientset.AppsV1().DaemonSets(namespace)
}

// ReplicaSets returns an interface for managing k8s replicasets
func (p *Client) ReplicaSets(namespace string) v1.ReplicaSetInterface {
	return p.clientset.AppsV1().ReplicaSets(namespace)
}

// Jobs returns an interface for managing k8s jobs
func (p *Client) Jobs(namespace string) v13.JobInterface {
	return p.clientset.BatchV1().Jobs(namespace)
}

// CronJobs returns an interface for managing k8s cronjobs
func (p *Client) CronJobs(namespace string) v1beta1.CronJobInterface {
	return p.clientset.BatchV1beta1().CronJobs(namespace)
}

// Ingresses returns an interface for managing k8s ingresses
func (p *Client) Ingresses(namespace string) v14.IngressInterface {
	return p.clientset.NetworkingV1().Ingresses(namespace)
}

// GetLogs returns a readerCloser that streams the pod's logs
func (p *Client) GetLogs(ctx context.Context, podName, namespace string, opts *corev1.PodLogOptions) (io.ReadCloser, error) {
	return p.Pods(namespace).GetLogs(podName, opts).Stream(ctx)
}

// Do executes the given function against the client
func (c *Client) Do(fn Func) error {
	return fn(c)
}
