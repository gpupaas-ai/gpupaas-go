package main

import (
	"context"
	"fmt"
	"log"

	v1alpha1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/client"
	"github.com/gpupaas-ai/gpupaas-go/runtime"
)

func main() {
	//c := client.New(client.Options{Config: gpupaas.ConfigFromEnv()})
	// OR
	c := client.New(client.Options{UseMemory: true})
	// OR
	// c := client.New(client.Options{
	// 	Config: gpupaas.NewConfig("https://api.gpupaas.ai", "your-api-token"),
	// })

	ctx := context.Background()

	ws := &v1alpha1.Workspace{
		TypeMeta: v1alpha1.TypeMeta{APIVersion: v1alpha1.APIVersion, Kind: v1alpha1.KindWorkspace},
		Metadata: v1alpha1.ObjectMeta{Name: "example", Project: "demo"},
	}
	applied, err := c.Apply(ctx, ws, "demo", "")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("applied %s/%s\n", applied.GetKind(), applied.GetName())

	gvk := runtime.GroupVersionKind{Group: v1alpha1.Group, Version: v1alpha1.Version, Kind: v1alpha1.KindWorkspace}
	items, err := c.List(ctx, gvk, "demo", "")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("listed %d workspace(s)\n", len(items))
}
