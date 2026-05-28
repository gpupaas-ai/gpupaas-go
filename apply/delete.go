package apply

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	v1alpha1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/client"
	"github.com/gpupaas-ai/gpupaas-go/runtime"
)

// DeleteReader decodes and deletes all documents from r.
func DeleteReader(ctx context.Context, c *client.Client, r io.Reader, project, workspace string, ignoreNotFound bool) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	ser := v1alpha1.NewSerializer(c.Scheme)
	objects, err := ser.DecodeAll(data)
	if err != nil {
		return err
	}
	for _, obj := range objects {
		gvk, err := c.Scheme.ObjectGVK(obj)
		if err != nil {
			gv, parseErr := runtime.ParseGroupVersion(obj.GetAPIVersion())
			if parseErr != nil {
				return parseErr
			}
			gvk = runtime.GroupVersionKind{Group: gv.Group, Version: gv.Version, Kind: obj.GetKind()}
		}
		delProject := obj.GetProject()
		if delProject == "" {
			delProject = project
		}
		delWorkspace := obj.GetWorkspace()
		if delWorkspace == "" {
			delWorkspace = workspace
		}
		if err := c.Delete(ctx, gvk, delProject, delWorkspace, obj.GetName()); err != nil {
			if ignoreNotFound && isNotFoundErr(err) {
				continue
			}
			return fmt.Errorf("delete %s/%s: %w", obj.GetKind(), obj.GetName(), err)
		}
	}
	return nil
}

// DeleteFile deletes all documents from path.
func DeleteFile(ctx context.Context, c *client.Client, path, project, workspace string, ignoreNotFound bool) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return DeleteReader(ctx, c, f, project, workspace, ignoreNotFound)
}

func isNotFoundErr(err error) bool {
	if gpupaas.IsNotFound(err) {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "not found")
}
