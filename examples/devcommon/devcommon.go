// Package devcommon provides shared helpers for dev resource examples (storage, security group, ssh key).
package devcommon

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
	"github.com/gpupaas-ai/gpupaas-go/runtime"
	"gopkg.in/yaml.v3"
)

// Config holds CLI flags shared by dev resource examples.
type Config struct {
	UseMemory bool
	Verbose   bool
	Project   string
	Workspace string
	Name      string
}

// ParseFlags reads standard dev example flags from the command line.
func ParseFlags(defaultName string) Config {
	cfg := Config{}
	flag.BoolVar(&cfg.UseMemory, "memory", false, "use in-memory backend (no auth, no HTTP)")
	flag.BoolVar(&cfg.Verbose, "v", false, "log HTTP requests and responses")
	flag.BoolVar(&cfg.Verbose, "verbose", false, "log HTTP requests and responses")
	flag.StringVar(&cfg.Project, "project", envOr("GPUPAAS_PROJECT", "demo"), "project name")
	flag.StringVar(&cfg.Workspace, "workspace", os.Getenv("GPUPAAS_WORKSPACE"), "workspace name (omit for project-scoped resources)")
	flag.StringVar(&cfg.Name, "name", envOr("GPUPAAS_RESOURCE_NAME", defaultName), "resource name")
	flag.Parse()
	return cfg
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// ScopeLabel describes project vs workspace scope for logging.
func ScopeLabel(workspace string) string {
	if workspace == "" {
		return "project-scoped"
	}
	return fmt.Sprintf("workspace-scoped (%s)", workspace)
}

// NewGenericClient returns a generic client for memory mode or nil when using typed clients.
func NewGenericClient(cfg Config) (*client.Client, error) {
	if !cfg.UseMemory {
		return nil, nil
	}
	return client.New(client.Options{UseMemory: true}), nil
}

// NewTypedClientset returns a clientset for live API mode.
func NewTypedClientset(cfg Config) (clientset.Interface, error) {
	gcfg := gpupaas.ConfigFromEnv()
	if gcfg.APIKey == "" {
		return nil, fmt.Errorf("GPUPAAS_API_KEY is required for live API (or use -memory)")
	}
	gcfg.Verbose = cfg.Verbose
	return clientset.NewForConfig(gcfg)
}

// PrintYAML prints a runtime object as YAML.
func PrintYAML(label string, obj runtime.Object) {
	data, err := yaml.Marshal(obj)
	if err != nil {
		log.Fatalf("marshal %s: %v", label, err)
	}
	fmt.Printf("%s (%s):\n%s---\n", label, ScopeLabel(obj.GetWorkspace()), string(data))
}

// ListMemory lists resources via the generic client (memory backend).
func ListMemory(ctx context.Context, c *client.Client, kind, project, workspace string) []runtime.Object {
	gvk := runtime.GroupVersionKind{Group: v1alpha1.Group, Version: v1alpha1.Version, Kind: kind}
	items, err := c.List(ctx, gvk, project, workspace)
	if err != nil {
		log.Fatalf("list: %v", err)
	}
	return items
}

// ApplyMemory applies a resource via the generic client (memory backend).
func ApplyMemory(ctx context.Context, c *client.Client, obj runtime.Object, project, workspace string) runtime.Object {
	applied, err := c.Apply(ctx, obj, project, workspace)
	if err != nil {
		log.Fatalf("apply: %v", err)
	}
	return applied
}

// GetMemory gets a resource via the generic client (memory backend).
func GetMemory(ctx context.Context, c *client.Client, kind, project, workspace, name string) runtime.Object {
	gvk := runtime.GroupVersionKind{Group: v1alpha1.Group, Version: v1alpha1.Version, Kind: kind}
	obj, err := c.Get(ctx, gvk, project, workspace, name)
	if err != nil {
		log.Fatalf("get: %v", err)
	}
	return obj
}

// DeleteMemory deletes a resource via the generic client (memory backend).
func DeleteMemory(ctx context.Context, c *client.Client, kind, project, workspace, name string) {
	gvk := runtime.GroupVersionKind{Group: v1alpha1.Group, Version: v1alpha1.Version, Kind: kind}
	if err := c.Delete(ctx, gvk, project, workspace, name); err != nil {
		log.Fatalf("delete: %v", err)
	}
}

// SampleStorage returns a representative storage spec for create examples.
func SampleStorage(project, workspace, name string) *v1alpha1.Storage {
	return &v1alpha1.Storage{
		TypeMeta: v1alpha1.TypeMeta{APIVersion: v1alpha1.APIVersion, Kind: v1alpha1.KindStorage},
		Metadata: v1alpha1.ObjectMeta{Name: name, Project: project, Workspace: workspace},
		Spec: v1alpha1.StorageSpec{
			Storage: v1alpha1.ResourceRef{Name: name},
			Type:    "standard",
			Size:    "10Gi",
		},
	}
}

// SampleSecurityGroup returns a representative security group spec for create examples.
func SampleSecurityGroup(project, workspace, name string) *v1alpha1.SecurityGroup {
	return &v1alpha1.SecurityGroup{
		TypeMeta: v1alpha1.TypeMeta{APIVersion: v1alpha1.APIVersion, Kind: v1alpha1.KindSecurityGroup},
		Metadata: v1alpha1.ObjectMeta{Name: name, Project: project, Workspace: workspace},
		Spec: v1alpha1.SecurityGroupSpec{
			SecurityGroup: v1alpha1.ResourceRef{Name: name},
			Type:          "default",
			IPRules: []v1alpha1.IpRule{
				{SourceCIDR: "0.0.0.0/0", Application: "ssh", Action: "allow"},
			},
		},
	}
}

// SampleSshKey returns a representative SSH key spec for create examples.
func SampleSshKey(project, workspace, name string) *v1alpha1.SshKey {
	return &v1alpha1.SshKey{
		TypeMeta: v1alpha1.TypeMeta{APIVersion: v1alpha1.APIVersion, Kind: v1alpha1.KindSshKey},
		Metadata: v1alpha1.ObjectMeta{Name: name, Project: project, Workspace: workspace},
		Spec: v1alpha1.SshKeySpec{
			SSHKey:    v1alpha1.ResourceRef{Name: name},
			PublicKey: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQ example-key",
		},
	}
}
