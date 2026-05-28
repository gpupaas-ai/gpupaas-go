// Package vmcommon provides shared helpers for virtual machine examples.
package vmcommon

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

// Config holds CLI flags shared by VM examples.
type Config struct {
	UseMemory bool
	Verbose   bool
	Project   string
	Workspace string
	Name      string
}

// ParseFlags reads standard VM example flags from the command line.
func ParseFlags() Config {
	cfg := Config{}
	flag.BoolVar(&cfg.UseMemory, "memory", false, "use in-memory backend (no auth, no HTTP)")
	flag.BoolVar(&cfg.Verbose, "v", false, "log HTTP requests and responses")
	flag.BoolVar(&cfg.Verbose, "verbose", false, "log HTTP requests and responses")
	flag.StringVar(&cfg.Project, "project", envOr("GPUPAAS_PROJECT", "demo"), "project name")
	flag.StringVar(&cfg.Workspace, "workspace", os.Getenv("GPUPAAS_WORKSPACE"), "workspace name (omit for project-scoped VMs)")
	flag.StringVar(&cfg.Name, "name", envOr("GPUPAAS_VM_NAME", "example-vm"), "virtual machine name")
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

// NewClients returns a typed VM client and optional generic client for memory mode.
func NewClients(cfg Config) (typedv1alpha1.VirtualMachineInterface, *client.Client, error) {
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
	return VMsClient(cs, cfg.Project, cfg.Workspace), nil, nil
}

// VMsClient selects project- or workspace-scoped virtual machine client.
func VMsClient(cs clientset.Interface, project, workspace string) typedv1alpha1.VirtualMachineInterface {
	if workspace != "" {
		return cs.V1alpha1().Workspaces(project).VirtualMachines(workspace)
	}
	return cs.V1alpha1().VirtualMachines(project)
}

// SampleVM returns a representative virtual machine spec for create examples.
func SampleVM(project, workspace, name string) *v1alpha1.VirtualMachine {
	return &v1alpha1.VirtualMachine{
		TypeMeta: v1alpha1.TypeMeta{
			APIVersion: v1alpha1.APIVersion,
			Kind:       v1alpha1.KindVirtualMachine,
		},
		Metadata: v1alpha1.ObjectMeta{
			Name:      name,
			Project:   project,
			Workspace: workspace,
		},
		Spec: v1alpha1.VirtualMachineSpec{
			VirtualMachine: v1alpha1.ResourceRef{Name: "ubuntu-22-profile", SystemCatalog: true},
			Type:           "kvm",
			CPUCount:       "2",
			Memory:         "4Gi",
			SecurityGroup:  "default-sg",
			SSHKey:         "my-ssh-key",
			VPC:            "tenant-vpc",
			Subnet:         "private-subnet",
			Image:          "ubuntu-22.04",
			BootDiskSize:   50,
		},
	}
}

// PrintYAML prints a runtime object as YAML.
func PrintYAML(label string, obj runtime.Object) {
	data, err := yaml.Marshal(obj)
	if err != nil {
		log.Fatalf("marshal %s: %v", label, err)
	}
	fmt.Printf("%s (%s):\n%s---\n", label, ScopeLabel(obj.GetWorkspace()), string(data))
}

// ListMemory lists VMs via the generic client (memory backend).
func ListMemory(ctx context.Context, c *client.Client, project, workspace string) []runtime.Object {
	gvk := runtime.GroupVersionKind{
		Group: v1alpha1.Group, Version: v1alpha1.Version, Kind: v1alpha1.KindVirtualMachine,
	}
	items, err := c.List(ctx, gvk, project, workspace)
	if err != nil {
		log.Fatalf("list: %v", err)
	}
	return items
}

// ApplyMemory applies a VM via the generic client (memory backend).
func ApplyMemory(ctx context.Context, c *client.Client, vm *v1alpha1.VirtualMachine) runtime.Object {
	applied, err := c.Apply(ctx, vm, vm.Metadata.Project, vm.Metadata.Workspace)
	if err != nil {
		log.Fatalf("apply: %v", err)
	}
	return applied
}

// GetMemory gets a VM via the generic client (memory backend).
func GetMemory(ctx context.Context, c *client.Client, project, workspace, name string) runtime.Object {
	gvk := runtime.GroupVersionKind{
		Group: v1alpha1.Group, Version: v1alpha1.Version, Kind: v1alpha1.KindVirtualMachine,
	}
	obj, err := c.Get(ctx, gvk, project, workspace, name)
	if err != nil {
		log.Fatalf("get: %v", err)
	}
	return obj
}

// DeleteMemory deletes a VM via the generic client (memory backend).
func DeleteMemory(ctx context.Context, c *client.Client, project, workspace, name string) {
	gvk := runtime.GroupVersionKind{
		Group: v1alpha1.Group, Version: v1alpha1.Version, Kind: v1alpha1.KindVirtualMachine,
	}
	if err := c.Delete(ctx, gvk, project, workspace, name); err != nil {
		log.Fatalf("delete: %v", err)
	}
}
