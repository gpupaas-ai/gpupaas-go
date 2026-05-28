// Create a storage.
//
//	go run ./examples/create-storage -memory -project demo -name example-storage
package main

import (
	"context"
	"fmt"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	v1alpha1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/examples/devcommon"
)

func main() {
	cfg := devcommon.ParseFlags("example-storage")
	ctx := context.Background()
	obj := devcommon.SampleStorage(cfg.Project, cfg.Workspace, cfg.Name)

	fmt.Printf("=== create storage %q (%s) ===\n", cfg.Name, devcommon.ScopeLabel(cfg.Workspace))

	if cfg.UseMemory {
		c, _ := devcommon.NewGenericClient(cfg)
		devcommon.PrintYAML("storage", devcommon.ApplyMemory(ctx, c, obj, cfg.Project, cfg.Workspace))
		return
	}

	cs, err := devcommon.NewTypedClientset(cfg)
	if err != nil {
		panic(err)
	}
	var created *v1alpha1.Storage
	if cfg.Workspace != "" {
		created, err = cs.V1alpha1().Workspaces(cfg.Project).Storages(cfg.Workspace).Create(ctx, obj, gpupaas.CreateOptions{})
	} else {
		created, err = cs.V1alpha1().Storages(cfg.Project).Create(ctx, obj, gpupaas.CreateOptions{})
	}
	if err != nil {
		panic(err)
	}
	devcommon.PrintYAML("storage", created)
}
