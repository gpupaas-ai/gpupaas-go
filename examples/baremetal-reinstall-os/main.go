// Reinstall the OS on a baremetal machine (POST .../reinstallOS).
//
// Requires live API — actions are not supported on the in-memory backend.
//
//	go run ./examples/baremetal-reinstall-os \
//	    -project demo \
//	    -name example-bm \
//	    -image-url http://images/example.qcow2
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
		log.Fatal("baremetal-reinstall-os requires live API; omit -memory")
	}
	if cfg.ImageURL == "" {
		log.Fatal("-image-url is required")
	}
	ctx := context.Background()

	bms, _, err := baremetalcommon.NewClients(cfg)
	if err != nil {
		panic(err)
	}

	fmt.Printf("=== reinstall OS on baremetal machine %q (image=%s) ===\n", cfg.Name, cfg.ImageURL)
	image := &v1alpha1.BaremetalImage{
		URL:          cfg.ImageURL,
		Format:       "qcow2",
		ChecksumType: "auto",
		Checksum:     "auto",
	}
	result, err := bms.ReinstallOS(ctx, cfg.Name, image, gpupaas.ActionOptions{})
	if err != nil {
		panic(err)
	}
	baremetalcommon.PrintYAML("baremetalmachine", result)
}
