// Package baremetalcommon provides shared helpers for first-class bare-metal examples.
package baremetalcommon

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

// Config holds CLI flags shared by bare-metal examples.
//
// Bare-metal machines have project scope and workspace scope: pass -workspace to use the
// workspace routes. SKU, Image and SSHKey only matter to create-baremetal and
// baremetal-reinstall-os; they must name values the BmProfile (SKU) of your project allows.
type Config struct {
	Verbose   bool
	Project   string
	Workspace string
	Name      string
	SKU       string
	Image     string
	SSHKey    string
	Wait      bool
}

// ParseFlags reads the standard bare-metal example flags.
func ParseFlags() Config {
	cfg := Config{}
	flag.BoolVar(&cfg.Verbose, "v", false, "log HTTP requests and responses")
	flag.BoolVar(&cfg.Verbose, "verbose", false, "log HTTP requests and responses")
	flag.StringVar(&cfg.Project, "project", envOr("GPUPAAS_PROJECT", "demo"), "project name")
	flag.StringVar(&cfg.Workspace, "workspace", os.Getenv("GPUPAAS_WORKSPACE"), "workspace name (omit for project scope)")
	flag.StringVar(&cfg.Name, "name", envOr("GPUPAAS_BAREMETAL_NAME", "example-bm"), "bare-metal machine name")
	flag.StringVar(&cfg.SKU, "sku", envOr("GPUPAAS_BAREMETAL_SKU", "bm-sku"), "BmProfile (SKU) name (create)")
	flag.StringVar(&cfg.Image, "image", envOr("GPUPAAS_BAREMETAL_IMAGE", "ubuntu-24"), "operating system image of the SKU (create, reinstall)")
	flag.StringVar(&cfg.SSHKey, "ssh-key", envOr("GPUPAAS_BAREMETAL_SSH_KEY", "my-ssh-key"), "SSH key of the workspace (create)")
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

// NewClient returns the bare-metal client at the scope of cfg. It needs GPUPAAS_API_KEY.
func NewClient(cfg Config) (typedv1alpha1.BareMetalInterface, error) {
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
		return cs.V1alpha1().Workspaces(cfg.Project).BareMetals(cfg.Workspace), nil
	}
	return cs.V1alpha1().BareMetals(cfg.Project), nil
}

// SampleBareMetal returns a representative machine for create examples. Replace the SKU, image
// and SSH key with values your BmProfile allows (see the BmProfiles client).
func SampleBareMetal(cfg Config) *v1alpha1.BareMetal {
	return &v1alpha1.BareMetal{
		TypeMeta: v1alpha1.TypeMeta{APIVersion: v1alpha1.APIVersion, Kind: v1alpha1.KindBareMetal},
		Metadata: v1alpha1.ObjectMeta{Name: cfg.Name, Project: cfg.Project, Workspace: cfg.Workspace},
		Spec: v1alpha1.BareMetalSpec{
			Type:   cfg.SKU,
			Image:  cfg.Image,
			SSHKey: cfg.SSHKey,
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
