// Package vmcommon provides shared helpers for first-class virtual machine examples.
package vmcommon

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	v1alpha1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/clientset"
	typedv1alpha1 "github.com/gpupaas-ai/gpupaas-go/clientset/typed/v1alpha1"
)

// Config holds CLI flags shared by VM examples.
//
// VMs have project scope and workspace scope: pass -workspace to use the workspace routes.
// SKU, Image, SSHKey, CPU and Memory only matter to create-vm; they must name values the
// VM profile (SKU) of your project allows.
type Config struct {
	Verbose   bool
	Project   string
	Workspace string
	Name      string
	SKU       string
	Image     string
	SSHKey    string
	CPU       string
	Memory    string
	Wait      bool
}

// ParseFlags reads the standard VM example flags.
func ParseFlags() Config {
	cfg := Config{}
	flag.BoolVar(&cfg.Verbose, "v", false, "log HTTP requests and responses")
	flag.BoolVar(&cfg.Verbose, "verbose", false, "log HTTP requests and responses")
	flag.StringVar(&cfg.Project, "project", envOr("GPUPAAS_PROJECT", "demo"), "project name")
	flag.StringVar(&cfg.Workspace, "workspace", os.Getenv("GPUPAAS_WORKSPACE"), "workspace name (omit for project scope)")
	flag.StringVar(&cfg.Name, "name", envOr("GPUPAAS_VM_NAME", "example-vm"), "virtual machine name")
	flag.StringVar(&cfg.SKU, "sku", envOr("GPUPAAS_VM_SKU", "vm-sku"), "VM profile (SKU) name (create)")
	flag.StringVar(&cfg.Image, "image", envOr("GPUPAAS_VM_IMAGE", "ubuntu-24"), "operating system image of the SKU (create)")
	flag.StringVar(&cfg.SSHKey, "ssh-key", envOr("GPUPAAS_VM_SSH_KEY", "my-ssh-key"), "SSH key of the workspace (create)")
	flag.StringVar(&cfg.CPU, "cpu", envOr("GPUPAAS_VM_CPU", "2"), "CPU count, a preset of the SKU (create)")
	flag.StringVar(&cfg.Memory, "memory", envOr("GPUPAAS_VM_MEMORY", "4"), "memory, a preset of the SKU (create)")
	flag.BoolVar(&cfg.Wait, "wait", false, "wait until the operation has settled")
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

// NewClient returns the VM client at the scope of cfg. It needs GPUPAAS_API_KEY.
func NewClient(cfg Config) (typedv1alpha1.VMInstanceInterface, error) {
	gcfg := gpupaas.ConfigFromEnv()
	if gcfg.APIKey == "" {
		return nil, fmt.Errorf("GPUPAAS_API_KEY is required")
	}
	gcfg.Verbose = cfg.Verbose
	cs, err := clientset.NewForConfig(gcfg)
	if err != nil {
		return nil, err
	}
	if cfg.Workspace != "" {
		return cs.V1alpha1().Workspaces(cfg.Project).VMInstances(cfg.Workspace), nil
	}
	return cs.V1alpha1().VMInstances(cfg.Project), nil
}

// SampleVM returns a representative VM for create examples. Replace the SKU, image, SSH key and
// size with values your VM profile allows (see the VMProfiles client).
func SampleVM(cfg Config) *v1alpha1.VMInstance {
	return &v1alpha1.VMInstance{
		TypeMeta: v1alpha1.TypeMeta{APIVersion: v1alpha1.APIVersion, Kind: v1alpha1.KindVMInstance},
		Metadata: v1alpha1.ObjectMeta{Name: cfg.Name, Project: cfg.Project, Workspace: cfg.Workspace},
		Spec: v1alpha1.VMInstanceSpec{
			Type:     cfg.SKU,
			CPUCount: cfg.CPU,
			Memory:   cfg.Memory,
			Image:    cfg.Image,
			SSHKey:   cfg.SSHKey,
		},
	}
}

// Print prints a value as indented JSON.
func Print(label string, v any) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		log.Fatalf("marshal %s: %v", label, err)
	}
	fmt.Printf("%s:\n%s\n", label, data)
}
