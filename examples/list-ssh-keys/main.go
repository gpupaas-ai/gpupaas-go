// List SSH keys at project or workspace scope.
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
	fmt.Printf("=== list ssh keys (%s) ===\n", devcommon.ScopeLabel(cfg.Workspace))

	if cfg.UseMemory {
		c, _ := devcommon.NewGenericClient(cfg)
		items := devcommon.ListMemory(ctx, c, v1alpha1.KindSshKey, cfg.Project, cfg.Workspace)
		fmt.Printf("count: %d\n", len(items))
		for _, item := range items {
			devcommon.PrintYAML("sshkey", item)
		}
		return
	}

	cs, err := devcommon.NewTypedClientset(cfg)
	if err != nil {
		panic(err)
	}
	var list *v1alpha1.SshKeyList
	if cfg.Workspace != "" {
		list, err = cs.V1alpha1().Workspaces(cfg.Project).SshKeys(cfg.Workspace).List(ctx, gpupaas.ListOptions{})
	} else {
		list, err = cs.V1alpha1().SshKeys(cfg.Project).List(ctx, gpupaas.ListOptions{})
	}
	if err != nil {
		panic(err)
	}
	fmt.Printf("count: %d\n", len(list.Items))
	for i := range list.Items {
		devcommon.PrintYAML("sshkey", &list.Items[i])
	}
}
