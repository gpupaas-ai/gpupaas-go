// List bare-metal machines at project or workspace scope.
//
//	export GPUPAAS_API_KEY=your-api-key
//	go run ./examples/list-baremetals -project demo
//	go run ./examples/list-baremetals -project demo -workspace dev -v
package main

import (
	"context"
	"fmt"
	"log"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	"github.com/gpupaas-ai/gpupaas-go/examples/baremetalcommon"
)

func main() {
	cfg := baremetalcommon.ParseFlags()
	ctx := context.Background()

	bms, err := baremetalcommon.NewClient(cfg)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("=== list bare-metal machines (%s) ===\n", baremetalcommon.ScopeLabel(cfg.Workspace))
	list, err := bms.List(ctx, gpupaas.ListOptions{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("count: %d\n", len(list.Items))
	for i := range list.Items {
		baremetalcommon.Print("baremetal", &list.Items[i])
	}
}
