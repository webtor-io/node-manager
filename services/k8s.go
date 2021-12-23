package services

import (
	"context"
	"os"
	"path/filepath"
	"sync"

	"github.com/pkg/errors"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "k8s.io/api/core/v1"

	t "github.com/webtor-io/node-manager/services/types"
)

var (
	NodeNotFoundErr = errors.New("node not found")
)

type K8s struct {
	once sync.Once
	cl   *kubernetes.Clientset
	err  error
}

func NewK8s() *K8s {
	return &K8s{}
}

func (s *K8s) GetNodeByName(ctx context.Context, name string) (*t.Node, error) {
	cl, err := s.client()
	if err != nil {
		return nil, err
	}
	opts := metav1.GetOptions{}
	n, err := cl.CoreV1().Nodes().Get(ctx, name, opts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get node")
	}
	if n == nil {
		return nil, NodeNotFoundErr
	}
	ips := []string{}
	for _, a := range n.Status.Addresses {
		ips = append(ips, a.Address)
	}
	return &t.Node{
		Name:     n.Name,
		Provider: n.Labels["provider"],
		IPs:      ips,
	}, nil
}

func (s *K8s) GetNotReadyNodes(ctx context.Context) ([]t.Node, error) {
	cl, err := s.client()
	if err != nil {
		return nil, err
	}
	timeout := int64(5)
	opts := metav1.ListOptions{
		TimeoutSeconds: &timeout,
	}
	nodes, err := cl.CoreV1().Nodes().List(ctx, opts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get nodes")
	}
	res := []t.Node{}

	for _, n := range nodes.Items {
		for _, c := range n.Status.Conditions {
			if c.Type == corev1.NodeReady && c.Status != corev1.ConditionTrue {
				ips := []string{}
				for _, a := range n.Status.Addresses {
					ips = append(ips, a.Address)
				}
				res = append(res, t.Node{
					Name:     n.Name,
					Provider: n.Labels["provider"],
					IPs:      ips,
				})
			}
		}
	}
	return res, nil
}

func (s *K8s) client() (*kubernetes.Clientset, error) {
	s.once.Do(func() {
		s.cl, s.err = s.initClient()
	})
	return s.cl, s.err
}

func (s *K8s) initClient() (*kubernetes.Clientset, error) {
	kubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	// log.Infof("checking local kubeconfig path=%s", kubeconfig)
	var config *rest.Config
	if _, err := os.Stat(kubeconfig); err == nil {
		// log.WithField("kubeconfig", kubeconfig).Info("loading config from file (local mode)")
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, errors.Wrap(err, "failed to make config")
		}
	} else {
		// log.Info("loading config from cluster (cluster mode)")
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, errors.Wrap(err, "failed to make config")
		}
	}
	return kubernetes.NewForConfig(config)
}
