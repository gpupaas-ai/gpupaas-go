// Package mkscommon provides shared helpers for first-class MKS (managed Kubernetes)
// examples.
package mkscommon

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	v1alpha1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/client"
	"github.com/gpupaas-ai/gpupaas-go/clientset"
	typedv1alpha1 "github.com/gpupaas-ai/gpupaas-go/clientset/typed/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/runtime"
	"gopkg.in/yaml.v3"
)

// Config holds CLI flags shared by MKS examples.
//
// Clusters have project scope (administrators) and workspace scope (developers): pass
// -workspace to use the workspace routes.
type Config struct {
	UseMemory  bool
	Verbose    bool
	Project    string
	Workspace  string
	Name       string
	Profile    string
	K8sVersion string
}

// ParseFlags reads the standard MKS example flags.
func ParseFlags() Config {
	cfg := Config{}
	flag.BoolVar(&cfg.UseMemory, "memory", false, "use in-memory backend (no auth, no HTTP)")
	flag.BoolVar(&cfg.Verbose, "v", false, "log HTTP requests and responses")
	flag.BoolVar(&cfg.Verbose, "verbose", false, "log HTTP requests and responses")
	flag.StringVar(&cfg.Project, "project", envOr("GPUPAAS_PROJECT", "demo"), "project name")
	flag.StringVar(&cfg.Workspace, "workspace", envOr("GPUPAAS_WORKSPACE", ""), "workspace name (empty for project scope)")
	flag.StringVar(&cfg.Name, "name", envOr("GPUPAAS_MKS_CLUSTER", "example-cluster"), "MKS cluster name")
	flag.StringVar(&cfg.Profile, "profile", envOr("GPUPAAS_MKS_PROFILE", "mks-oneclick-1"), "cluster profile (Kubernetes SKU) name")
	flag.StringVar(&cfg.K8sVersion, "k8s-version", envOr("GPUPAAS_MKS_K8S_VERSION", "1.36"), "Kubernetes version (for create and upgrade)")
	flag.Parse()
	return cfg
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// NewTyped returns the typed v1alpha1 client, or a generic client for the memory backend.
func NewTyped(cfg Config) (typedv1alpha1.Interface, *client.Client, error) {
	if cfg.UseMemory {
		return nil, client.New(client.Options{UseMemory: true}), nil
	}
	gcfg := gpupaas.ConfigFromEnv()
	if gcfg.APIKey == "" {
		return nil, nil, fmt.Errorf("GPUPAAS_API_KEY is required for live API (or use -memory)")
	}
	gcfg.Verbose = cfg.Verbose
	cs, err := clientset.NewForConfig(gcfg)
	if err != nil {
		return nil, nil, err
	}
	return cs.V1alpha1(), nil, nil
}

// NewClients returns a typed cluster client at the scope of cfg and an optional generic
// client for the memory backend.
func NewClients(cfg Config) (typedv1alpha1.MKSClusterInterface, *client.Client, error) {
	typed, mem, err := NewTyped(cfg)
	if err != nil || mem != nil {
		return nil, mem, err
	}
	if cfg.Workspace != "" {
		return typed.Workspaces(cfg.Project).MKSClusters(cfg.Workspace), nil, nil
	}
	return typed.MKSClusters(cfg.Project), nil, nil
}

// SampleCluster returns a representative cluster: one control-plane node from the profile's
// default SKU and no worker pools. Replace the SKU, image, datacenter and SSH key with values
// the profile allows (see `get clusterprofile`).
func SampleCluster(cfg Config) *v1alpha1.MKSCluster {
	return &v1alpha1.MKSCluster{
		TypeMeta: v1alpha1.TypeMeta{APIVersion: v1alpha1.APIVersion, Kind: v1alpha1.KindMKSCluster},
		Metadata: v1alpha1.ObjectMeta{Name: cfg.Name, Project: cfg.Project, Workspace: cfg.Workspace},
		Spec: v1alpha1.MKSClusterSpec{
			Profile:           &v1alpha1.MKSProfileRef{Name: cfg.Profile},
			KubernetesVersion: cfg.K8sVersion,
			ControlPlane: &v1alpha1.MKSNodeGroup{
				Name:    "control-plane",
				OSImage: "ubuntu-26",
				SKUName: "bmaas-test-1",
				SKUType: "BMAAS",
				SSHKey:  "mine",
				Scaling: &v1alpha1.MKSScaling{Mode: "manual", Count: 1},
			},
		},
	}
}

// PrintYAML prints a runtime object as YAML.
func PrintYAML(label string, obj runtime.Object) {
	data, err := yaml.Marshal(obj)
	if err != nil {
		log.Fatalf("marshal %s: %v", label, err)
	}
	fmt.Printf("%s:\n%s---\n", label, string(data))
}

// PrintAny prints any value as YAML.
func PrintAny(label string, v any) {
	data, err := yaml.Marshal(v)
	if err != nil {
		log.Fatalf("marshal %s: %v", label, err)
	}
	fmt.Printf("%s:\n%s---\n", label, string(data))
}

// ApplyMemory applies a cluster via the generic client (memory backend).
func ApplyMemory(ctx context.Context, c *client.Client, cluster *v1alpha1.MKSCluster) runtime.Object {
	applied, err := c.Apply(ctx, cluster, cluster.Metadata.Project, cluster.Metadata.Workspace)
	if err != nil {
		log.Fatalf("apply: %v", err)
	}
	return applied
}

// GetMemory gets a cluster via the generic client (memory backend).
func GetMemory(ctx context.Context, c *client.Client, cfg Config) runtime.Object {
	obj, err := c.Get(ctx, clusterGVK(), cfg.Project, cfg.Workspace, cfg.Name)
	if err != nil {
		log.Fatalf("get: %v", err)
	}
	return obj
}

// ListMemory lists clusters via the generic client (memory backend).
func ListMemory(ctx context.Context, c *client.Client, cfg Config) []runtime.Object {
	items, err := c.List(ctx, clusterGVK(), cfg.Project, cfg.Workspace)
	if err != nil {
		log.Fatalf("list: %v", err)
	}
	return items
}

// DeleteMemory deletes a cluster via the generic client (memory backend).
func DeleteMemory(ctx context.Context, c *client.Client, cfg Config) {
	if err := c.Delete(ctx, clusterGVK(), cfg.Project, cfg.Workspace, cfg.Name); err != nil {
		log.Fatalf("delete: %v", err)
	}
}

func clusterGVK() runtime.GroupVersionKind {
	return runtime.GroupVersionKind{Group: v1alpha1.Group, Version: v1alpha1.Version, Kind: v1alpha1.KindMKSCluster}
}
