package apply

import (
	"context"
	"fmt"
	"io"
	"os"

	v1alpha1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/client"
)

// ApplyReader decodes and applies all documents from r.
func ApplyReader(ctx context.Context, c *client.Client, r io.Reader, project, workspace string) error {
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
		if _, err := c.Apply(ctx, obj, project, workspace); err != nil {
			return fmt.Errorf("apply %s/%s: %w", obj.GetKind(), obj.GetName(), err)
		}
	}
	return nil
}

// ApplyFile applies all documents from path.
func ApplyFile(ctx context.Context, c *client.Client, path, project, workspace string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return ApplyReader(ctx, c, f, project, workspace)
}
