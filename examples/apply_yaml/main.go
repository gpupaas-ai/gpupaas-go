package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/gpupaas-ai/gpupaas-go/apply"
	"github.com/gpupaas-ai/gpupaas-go/client"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: apply_yaml <file.yaml> [project]")
	}
	project := "demo"
	if len(os.Args) > 2 {
		project = os.Args[2]
	}

	c := client.New(client.Options{UseMemory: true})
	if err := apply.ApplyFile(context.Background(), c, os.Args[1], project, ""); err != nil {
		log.Fatal(err)
	}
	fmt.Println("applied")
}
