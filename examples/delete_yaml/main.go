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
		log.Fatal("usage: delete_yaml <file.yaml> [project]")
	}
	project := "demo"
	if len(os.Args) > 2 {
		project = os.Args[2]
	}

	c := client.New(client.Options{UseMemory: true})
	if err := apply.DeleteFile(context.Background(), c, os.Args[1], project, "", true); err != nil {
		log.Fatal(err)
	}
	fmt.Println("deleted")
}
