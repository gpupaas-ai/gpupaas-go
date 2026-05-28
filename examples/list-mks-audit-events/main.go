// List audit events for an MKS cluster.
//
// Requires live API — MKS sub-resources are not supported on the in-memory backend.
//
//	go run ./examples/list-mks-audit-events -project demo -name my-cluster
package main

import (
	"context"
	"fmt"
	"log"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	"github.com/gpupaas-ai/gpupaas-go/examples/mkscommon"
)

func main() {
	cfg := mkscommon.ParseFlags()
	if cfg.UseMemory {
		log.Fatal("list-mks-audit-events requires live API; omit -memory")
	}
	ctx := context.Background()

	clusters, _, err := mkscommon.NewClients(cfg)
	if err != nil {
		panic(err)
	}

	fmt.Printf("=== list audit events for MKS cluster %q ===\n", cfg.Name)
	list, err := clusters.AuditEvents(cfg.Name).List(ctx, gpupaas.ListOptions{})
	if err != nil {
		panic(err)
	}
	for i := range list.Items {
		mkscommon.PrintAny("mksauditevent", &list.Items[i])
	}
}
