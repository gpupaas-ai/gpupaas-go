// Get an SSH key by name.
package main

import (
	"context"
	"fmt"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	v1alpha1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/examples/devcommon"
)

func main() {
	cfg := devcommon.ParseFlags("example-ssh-key")
	ctx := context.Background()
	fmt.Printf("=== get ssh key %q (%s) ===\n", cfg.Name, devcommon.ScopeLabel(cfg.Workspace))

	if cfg.UseMemory {
		c, _ := devcommon.NewGenericClient(cfg)
		devcommon.PrintYAML("sshkey", devcommon.GetMemory(ctx, c, v1alpha1.KindSshKey, cfg.Project, cfg.Workspace, cfg.Name))
		return
	}

	cs, err := devcommon.NewTypedClientset(cfg)
	if err != nil {
		panic(err)
	}
	var obj *v1alpha1.SshKey
	if cfg.Workspace != "" {
		obj, err = cs.V1alpha1().Workspaces(cfg.Project).SshKeys(cfg.Workspace).Get(ctx, cfg.Name, gpupaas.GetOptions{})
	} else {
		obj, err = cs.V1alpha1().SshKeys(cfg.Project).Get(ctx, cfg.Name, gpupaas.GetOptions{})
	}
	if err != nil {
		panic(err)
	}
	devcommon.PrintYAML("sshkey", obj)
}
