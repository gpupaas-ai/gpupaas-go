// Package baremetalcommon provides shared helpers for BaremetalMachine examples.
package baremetalcommon

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

// Config holds CLI flags shared by BaremetalMachine examples.
//
// BaremetalMachine resources are project-scoped only, so there is no
// -workspace flag (any workspace value would be ignored by the backend).
type Config struct {
	UseMemory bool
	Verbose   bool
	Project   string
	Name      string
	ComputeID string
	ImageURL  string
}

// ParseFlags reads the standard BaremetalMachine example flags.
func ParseFlags() Config {
	cfg := Config{}
	flag.BoolVar(&cfg.UseMemory, "memory", false, "use in-memory backend (no auth, no HTTP)")
	flag.BoolVar(&cfg.Verbose, "v", false, "log HTTP requests and responses")
	flag.BoolVar(&cfg.Verbose, "verbose", false, "log HTTP requests and responses")
	flag.StringVar(&cfg.Project, "project", envOr("GPUPAAS_PROJECT", "demo"), "project name")
	flag.StringVar(&cfg.Name, "name", envOr("GPUPAAS_BAREMETAL_NAME", "example-bm"), "baremetal machine name")
	flag.StringVar(&cfg.ComputeID, "compute-id", os.Getenv("GPUPAAS_BAREMETAL_COMPUTE_ID"), "compute_id for SOL console session")
	flag.StringVar(&cfg.ImageURL, "image-url", envOr("GPUPAAS_BAREMETAL_IMAGE_URL", "http://images/example.qcow2"), "image URL for ReinstallOS")
	flag.Parse()
	return cfg
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// NewClients returns a typed BaremetalMachine client and an optional generic
// client for the memory backend.
func NewClients(cfg Config) (typedv1alpha1.BaremetalMachineInterface, *client.Client, error) {
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
	return cs.V1alpha1().BaremetalMachines(cfg.Project), nil, nil
}

// SampleBaremetalMachine returns a representative BaremetalMachine spec for
// create examples. Replace fields with real datacenter/inventory values for
// live API usage.
func SampleBaremetalMachine(project, name, imageURL string) *v1alpha1.BaremetalMachine {
	online := true
	return &v1alpha1.BaremetalMachine{
		TypeMeta: v1alpha1.TypeMeta{
			APIVersion: v1alpha1.APIVersion,
			Kind:       v1alpha1.KindBaremetalMachine,
		},
		Metadata: v1alpha1.ObjectMeta{
			Name:    name,
			Project: project,
		},
		Spec: v1alpha1.BaremetalMachineSpec{
			Architecture:             "x86_64",
			AutomatedCleaningMode:    "metadata",
			BaremetalProvisionerName: "example-provisioner",
			BootMode:                 "UEFI",
			Datacenter:               "dc1",
			DeviceID:                 "device-001",
			Hostname:                 name,
			MACAddress:               "aa:bb:cc:dd:ee:ff",
			SSHKey:                   "my-ssh-key",
			Online:                   &online,
			Image: &v1alpha1.BaremetalImage{
				URL:          imageURL,
				Format:       "qcow2",
				ChecksumType: "auto",
				Checksum:     "auto",
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

// PrintAny prints any value as YAML for convenience (used for non-runtime
// payloads such as ConsoleSession responses and Info envelopes).
func PrintAny(label string, v any) {
	data, err := yaml.Marshal(v)
	if err != nil {
		log.Fatalf("marshal %s: %v", label, err)
	}
	fmt.Printf("%s:\n%s---\n", label, string(data))
}

// ApplyMemory applies a BaremetalMachine via the generic client (memory backend).
func ApplyMemory(ctx context.Context, c *client.Client, bm *v1alpha1.BaremetalMachine) runtime.Object {
	applied, err := c.Apply(ctx, bm, bm.Metadata.Project, "")
	if err != nil {
		log.Fatalf("apply: %v", err)
	}
	return applied
}

// GetMemory gets a BaremetalMachine via the generic client (memory backend).
func GetMemory(ctx context.Context, c *client.Client, project, name string) runtime.Object {
	gvk := runtime.GroupVersionKind{
		Group: v1alpha1.Group, Version: v1alpha1.Version, Kind: v1alpha1.KindBaremetalMachine,
	}
	obj, err := c.Get(ctx, gvk, project, "", name)
	if err != nil {
		log.Fatalf("get: %v", err)
	}
	return obj
}

// ListMemory lists BaremetalMachines via the generic client (memory backend).
func ListMemory(ctx context.Context, c *client.Client, project string) []runtime.Object {
	gvk := runtime.GroupVersionKind{
		Group: v1alpha1.Group, Version: v1alpha1.Version, Kind: v1alpha1.KindBaremetalMachine,
	}
	items, err := c.List(ctx, gvk, project, "")
	if err != nil {
		log.Fatalf("list: %v", err)
	}
	return items
}

// DeleteMemory deletes a BaremetalMachine via the generic client (memory backend).
func DeleteMemory(ctx context.Context, c *client.Client, project, name string) {
	gvk := runtime.GroupVersionKind{
		Group: v1alpha1.Group, Version: v1alpha1.Version, Kind: v1alpha1.KindBaremetalMachine,
	}
	if err := c.Delete(ctx, gvk, project, "", name); err != nil {
		log.Fatalf("delete: %v", err)
	}
}
