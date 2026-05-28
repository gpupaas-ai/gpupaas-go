// List virtual machines at project or workspace scope.
//
// Project-scoped:
//
//	go run ./examples/list-vms -memory -project demo
//
// Workspace-scoped:
//
//	go run ./examples/list-vms -memory -project demo -workspace dev
//
// Live API:
//
//	export GPUPAAS_API_KEY=your-api-key
//	go run ./examples/list-vms -project demo -workspace dev -v
package main

import (
	"context"
	"fmt"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	"github.com/gpupaas-ai/gpupaas-go/examples/vmcommon"
)

func main() {
	cfg := vmcommon.ParseFlags()
	ctx := context.Background()

	vms, memClient, err := vmcommon.NewClients(cfg)
	if err != nil {
		panic(err)
	}

	fmt.Printf("=== list virtual machines (%s) ===\n", vmcommon.ScopeLabel(cfg.Workspace))

	if memClient != nil {
		items := vmcommon.ListMemory(ctx, memClient, cfg.Project, cfg.Workspace)
		fmt.Printf("count: %d\n", len(items))
		for _, item := range items {
			vmcommon.PrintYAML("virtualmachine", item)
		}
		return
	}

	list, err := vms.List(ctx, gpupaas.ListOptions{})
	if err != nil {
		panic(err)
	}
	fmt.Printf("count: %d\n", len(list.Items))
	for i := range list.Items {
		vmcommon.PrintYAML("virtualmachine", &list.Items[i])
	}
}
