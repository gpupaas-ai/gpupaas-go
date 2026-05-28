// Get a storage by name.
//
//	go run ./examples/get-storage -memory -project demo -name example-storage
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

	fmt.Printf("=== get storage %q (%s) ===\n", cfg.Name, devcommon.ScopeLabel(cfg.Workspace))

	if cfg.UseMemory {
		c, _ := devcommon.NewGenericClient(cfg)
		devcommon.PrintYAML("storage", devcommon.GetMemory(ctx, c, v1alpha1.KindStorage, cfg.Project, cfg.Workspace, cfg.Name))
		return
	}

	cs, err := devcommon.NewTypedClientset(cfg)
	if err != nil {
		panic(err)
	}
	var obj *v1alpha1.Storage
	if cfg.Workspace != "" {
		obj, err = cs.V1alpha1().Workspaces(cfg.Project).Storages(cfg.Workspace).Get(ctx, cfg.Name, gpupaas.GetOptions{})
	} else {
		obj, err = cs.V1alpha1().Storages(cfg.Project).Get(ctx, cfg.Name, gpupaas.GetOptions{})
	}
	if err != nil {
		panic(err)
	}
	devcommon.PrintYAML("storage", obj)
}
