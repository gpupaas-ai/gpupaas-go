// Full gpupaas-go example: auth, Apply, Get, List, Delete.
//
// In-memory (no auth, no network):
//
//	go run ./examples/full_client -memory
//
// Live API (API key required, same as paasctl):
//
//	export GPUPAAS_API_KEY=your-api-key
//	export GPUPAAS_API_SECRET=your-api-secret   # optional; defaults to API key
//	export GPUPAAS_ENDPOINT=https://console.gpupaas.ai   # optional
//	go run ./examples/full_client -project demo
//
// Or pass token explicitly in code (see newClient below).
package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	v1alpha1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/client"
	"github.com/gpupaas-ai/gpupaas-go/runtime"
	"gopkg.in/yaml.v3"
)

func main() {
	useMemory := flag.Bool("memory", false, "use in-memory backend (no auth, no HTTP)")
	verbose := flag.Bool("v", false, "log HTTP requests and responses")
	verboseLong := flag.Bool("verbose", false, "log HTTP requests and responses")
	project := flag.String("project", "test", "project name")
	workspace := flag.String("workspace", "example", "workspace name to create")
	flag.Parse()

	ctx := context.Background()
	c := newClient(*useMemory, *verbose || *verboseLong)

	// 1. Ensure project exists
	projGVK := runtime.GroupVersionKind{Group: v1alpha1.Group, Version: v1alpha1.Version, Kind: v1alpha1.KindProject}
	projDetails, err := c.Get(ctx, projGVK, "", "", *project)
	if err != nil {
		log.Fatalf("get project: %v", err)
	}
	yamlData, err := yaml.Marshal(projDetails)
	if err != nil {
		log.Fatalf("marshal project details: %v", err)
	}
	fmt.Printf("project %q ready and details: in yaml: \n%s\n---\n", *project, string(yamlData))

	// 2. Apply workspace under that project
	ws := &v1alpha1.Workspace{
		TypeMeta: v1alpha1.TypeMeta{
			APIVersion: v1alpha1.APIVersion,
			Kind:       v1alpha1.KindWorkspace,
		},
		Metadata: v1alpha1.ObjectMeta{
			Name:    *workspace,
			Project: *project,
		},
		Spec: v1alpha1.WorkspaceSpec{
			Description: "Example workspace",
		},
	}
	applied, err := c.Apply(ctx, ws, *project, "")
	if err != nil {
		log.Fatalf("apply workspace: %v", err)
	}
	fmt.Printf("applied %s/%s (project=%s)\n", applied.GetKind(), applied.GetName(), applied.GetProject())

	// 3. List workspaces in project
	wsGVK := runtime.GroupVersionKind{
		Group:   v1alpha1.Group,
		Version: v1alpha1.Version,
		Kind:    v1alpha1.KindWorkspace,
	}
	items, err := c.List(ctx, wsGVK, *project, "")
	if err != nil {
		log.Fatalf("list workspaces: %v", err)
	}
	fmt.Printf("listed %d workspace(s) in project %q\n", len(items), *project)
	for _, item := range items {
		yamlData, err := yaml.Marshal(item)
		if err != nil {
			log.Fatalf("marshal workspace details: %v", err)
		}
		fmt.Printf("workspace %q details: in yaml: \n%s\n---\n", item.GetName(), string(yamlData))
	}

	// 4. Get single workspace by name
	got, err := c.Get(ctx, wsGVK, *project, "", *workspace)
	if err != nil {
		log.Fatalf("get workspace: %v", err)
	}
	fmt.Printf("get OK: %s/%s\n", got.GetKind(), got.GetName())
	yamlData, err = yaml.Marshal(got)
	if err != nil {
		log.Fatalf("marshal workspace details: %v", err)
	}
	fmt.Printf("workspace %q details: in yaml: \n%s\n---\n", got.GetName(), string(yamlData))

	// 5. Delete workspace (optional — comment out to keep resource)
	if err := c.Delete(ctx, wsGVK, *project, "", *workspace); err != nil {
		if gpupaas.IsNotFound(err) {
			fmt.Println("workspace already absent")
		} else {
			log.Fatalf("delete workspace: %v", err)
		}
	} else {
		fmt.Printf("deleted workspace %q\n", *workspace)
	}
}

func newClient(useMemory, verbose bool) *client.Client {
	if useMemory {
		return client.New(client.Options{UseMemory: true})
	}

	cfg := gpupaas.ConfigFromEnv()
	if cfg.APIKey == "" {
		log.Fatal("GPUPAAS_API_KEY is required for live API (or use -memory for local demo)")
	}

	return client.New(client.Options{
		Config:  cfg,
		Verbose: verbose,
	})
}
