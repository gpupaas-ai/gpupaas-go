// Package mkscommon provides shared helpers for MKS (Managed Kubernetes
// Service) examples.
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
// MKSCluster resources are project-scoped only, so there is no -workspace
// flag. Sub-resources (nodes, worker node groups, audit events) are scoped to
// a cluster, identified by -name.
type Config struct {
	UseMemory   bool
	Verbose     bool
	Project     string
	Name        string
	NodeName    string
	NodeGroup   string
	K8sVersion  string
	DesiredSize int
}

// ParseFlags reads the standard MKS example flags.
func ParseFlags() Config {
	cfg := Config{}
	flag.BoolVar(&cfg.UseMemory, "memory", false, "use in-memory backend (no auth, no HTTP)")
	flag.BoolVar(&cfg.Verbose, "v", false, "log HTTP requests and responses")
	flag.BoolVar(&cfg.Verbose, "verbose", false, "log HTTP requests and responses")
	flag.StringVar(&cfg.Project, "project", envOr("GPUPAAS_PROJECT", "demo"), "project name")
	flag.StringVar(&cfg.Name, "name", envOr("GPUPAAS_MKS_CLUSTER", "example-cluster"), "MKS cluster name")
	flag.StringVar(&cfg.NodeName, "node", os.Getenv("GPUPAAS_MKS_NODE"), "MKS node name (for node sub-actions)")
	flag.StringVar(&cfg.NodeGroup, "node-group", os.Getenv("GPUPAAS_MKS_NODE_GROUP"), "node group name (for scale/remove)")
	flag.StringVar(&cfg.K8sVersion, "k8s-version", envOr("GPUPAAS_MKS_K8S_VERSION", "1.32"), "target Kubernetes version (for upgrade)")
	flag.IntVar(&cfg.DesiredSize, "desired-size", 3, "desired node count (for scale)")
	flag.Parse()
	return cfg
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// NewClients returns a typed MKSCluster client and an optional generic client
// for the memory backend.
func NewClients(cfg Config) (typedv1alpha1.MKSClusterInterface, *client.Client, error) {
	if cfg.UseMemory {
		c := client.New(client.Options{UseMemory: true})
		return nil, c, nil
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
	return cs.V1alpha1().MKSClusters(cfg.Project), nil, nil
}

// SampleCluster returns a representative MKSCluster spec for create examples.
// Replace fields with real node/inventory values for live API usage.
func SampleCluster(project, name string) *v1alpha1.MKSCluster {
	ha := false
	dedicated := false
	return &v1alpha1.MKSCluster{
		TypeMeta: v1alpha1.TypeMeta{
			APIVersion: v1alpha1.APIVersion,
			Kind:       v1alpha1.KindMKSCluster,
		},
		Metadata: v1alpha1.ObjectMeta{
			Name:    name,
			Project: project,
		},
		Spec: v1alpha1.MKSClusterSpec{
			KubernetesVersion:     "1.31",
			CNI:                   "calico",
			OS:                    "ubuntu22.04",
			HAEnabled:             &ha,
			DedicatedControlPlane: &dedicated,
			Blueprint:             &v1alpha1.MKSBlueprint{Name: "minimal", Version: "v1"},
			Nodes: []v1alpha1.MKSNodeSpec{
				{
					Hostname:        "master-1",
					PrivateIP:       "10.0.0.10",
					SSHUserName:     "ubuntu",
					SSHKey:          "-----BEGIN OPENSSH PRIVATE KEY-----\n...",
					Roles:           []string{"master", "worker"},
					Arch:            "amd64",
					OperatingSystem: "ubuntu22.04",
				},
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
	fmt.Printf("%s (project-scoped):\n%s---\n", label, string(data))
}

// PrintAny prints any value as YAML (used for node/wng/audit payloads).
func PrintAny(label string, v any) {
	data, err := yaml.Marshal(v)
	if err != nil {
		log.Fatalf("marshal %s: %v", label, err)
	}
	fmt.Printf("%s:\n%s---\n", label, string(data))
}

// ApplyMemory applies an MKSCluster via the generic client (memory backend).
func ApplyMemory(ctx context.Context, c *client.Client, cluster *v1alpha1.MKSCluster) runtime.Object {
	applied, err := c.Apply(ctx, cluster, cluster.Metadata.Project, "")
	if err != nil {
		log.Fatalf("apply: %v", err)
	}
	return applied
}

// GetMemory gets an MKSCluster via the generic client (memory backend).
func GetMemory(ctx context.Context, c *client.Client, project, name string) runtime.Object {
	obj, err := c.Get(ctx, clusterGVK(), project, "", name)
	if err != nil {
		log.Fatalf("get: %v", err)
	}
	return obj
}

// ListMemory lists MKSClusters via the generic client (memory backend).
func ListMemory(ctx context.Context, c *client.Client, project string) []runtime.Object {
	items, err := c.List(ctx, clusterGVK(), project, "")
	if err != nil {
		log.Fatalf("list: %v", err)
	}
	return items
}

// DeleteMemory deletes an MKSCluster via the generic client (memory backend).
func DeleteMemory(ctx context.Context, c *client.Client, project, name string) {
	if err := c.Delete(ctx, clusterGVK(), project, "", name); err != nil {
		log.Fatalf("delete: %v", err)
	}
}

func clusterGVK() runtime.GroupVersionKind {
	return runtime.GroupVersionKind{
		Group: v1alpha1.Group, Version: v1alpha1.Version, Kind: v1alpha1.KindMKSCluster,
	}
}
