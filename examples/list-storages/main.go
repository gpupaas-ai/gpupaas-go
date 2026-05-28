// List storages at project or workspace scope.
//
//	go run ./examples/list-storages -memory -project demo
//	go run ./examples/list-storages -memory -project demo -workspace dev
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

	fmt.Printf("=== list storages (%s) ===\n", devcommon.ScopeLabel(cfg.Workspace))

	if cfg.UseMemory {
		c, _ := devcommon.NewGenericClient(cfg)
		items := devcommon.ListMemory(ctx, c, v1alpha1.KindStorage, cfg.Project, cfg.Workspace)
		fmt.Printf("count: %d\n", len(items))
		for _, item := range items {
			devcommon.PrintYAML("storage", item)
		}
		return
	}

	cs, err := devcommon.NewTypedClientset(cfg)
	if err != nil {
		panic(err)
	}
	var list *v1alpha1.StorageList
	if cfg.Workspace != "" {
		list, err = cs.V1alpha1().Workspaces(cfg.Project).Storages(cfg.Workspace).List(ctx, gpupaas.ListOptions{})
	} else {
		list, err = cs.V1alpha1().Storages(cfg.Project).List(ctx, gpupaas.ListOptions{})
	}
	if err != nil {
		panic(err)
	}
	fmt.Printf("count: %d\n", len(list.Items))
	for i := range list.Items {
		devcommon.PrintYAML("storage", &list.Items[i])
	}
}
