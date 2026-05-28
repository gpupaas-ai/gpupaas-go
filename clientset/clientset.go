package clientset

import (
	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	v1alpha1 "github.com/gpupaas-ai/gpupaas-go/clientset/typed/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/rest"
)

// Clientset provides typed API clients.
type Clientset struct {
	v1alpha1 *v1alpha1.Client
}

// Interface is the clientset API.
type Interface interface {
	V1alpha1() v1alpha1.Interface
}

// NewForConfig creates a Clientset from gpupaas Config.
func NewForConfig(cfg gpupaas.Config) (*Clientset, error) {
	restClient, err := rest.NewClient(rest.ConfigFromGPUPAAS(cfg))
	if err != nil {
		return nil, err
	}
	return &Clientset{v1alpha1: v1alpha1.New(restClient)}, nil
}

// V1alpha1 returns the v1alpha1 client.
func (c *Clientset) V1alpha1() v1alpha1.Interface {
	return c.v1alpha1
}
