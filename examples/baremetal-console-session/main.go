// Create a short-lived SOL console session for a baremetal machine
// (POST .../consoleSessions). Returns a WebSocket URL.
//
// Requires live API — actions are not supported on the in-memory backend.
//
//	go run ./examples/baremetal-console-session \
//	    -project demo \
//	    -name example-bm \
//	    -compute-id compute-42
package main

import (
	"context"
	"fmt"
	"log"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	v1alpha1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/examples/baremetalcommon"
)

func main() {
	cfg := baremetalcommon.ParseFlags()
	if cfg.UseMemory {
		log.Fatal("baremetal-console-session requires live API; omit -memory")
	}
	if cfg.ComputeID == "" {
		log.Fatal("-compute-id is required")
	}
	ctx := context.Background()

	bms, _, err := baremetalcommon.NewClients(cfg)
	if err != nil {
		panic(err)
	}

	fmt.Printf("=== create SOL console session for baremetal machine %q (compute_id=%s) ===\n", cfg.Name, cfg.ComputeID)
	session, err := bms.CreateConsoleSession(ctx, cfg.Name, &v1alpha1.BaremetalConsoleSessionRequest{
		ComputeID: cfg.ComputeID,
	}, gpupaas.ActionOptions{})
	if err != nil {
		panic(err)
	}
	baremetalcommon.PrintAny("console-session", session)
}
