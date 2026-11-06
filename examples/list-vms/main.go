// List virtual machines at project or workspace scope.
//
//	export GPUPAAS_API_KEY=your-api-key
//	go run ./examples/list-vms -project demo
//	go run ./examples/list-vms -project demo -workspace dev -v
package main

import (
	"context"
	"fmt"
	"log"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	"github.com/gpupaas-ai/gpupaas-go/examples/vmcommon"
)

func main() {
	cfg := vmcommon.ParseFlags()
	ctx := context.Background()

	vms, err := vmcommon.NewClient(cfg)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("=== list virtual machines (%s) ===\n", vmcommon.ScopeLabel(cfg.Workspace))
	list, err := vms.List(ctx, gpupaas.ListOptions{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("count: %d\n", len(list.Items))
	for i := range list.Items {
		vmcommon.Print("virtualmachine", &list.Items[i])
	}
}
