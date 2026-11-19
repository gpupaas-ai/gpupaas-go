package v1alpha1

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/convert"
	"github.com/gpupaas-ai/gpupaas-go/rest"
)

// mksMatchmakerClient implements MKSMatchmakerInterface.
type mksMatchmakerClient struct {
	rest    *rest.Client
	project string
}

// query builds the query string of a matchmaker call. Datacenters and images repeat; SKU kinds and compute types are
// joined with "+" ("bmaas+vmaas") as the API reference writes them. The plus is percent-encoded: a raw "+" in a
// query string means a space to a server that decodes it, and the compound form the platform accepts has a plus.
func (q MKSMatchQuery) query() string {
	var parts []string
	for _, d := range q.Datacenters {
		parts = append(parts, "datacenter="+url.QueryEscape(d))
	}
	joined := func(key string, values []string) {
		if len(values) == 0 {
			return
		}
		lower := make([]string, len(values))
		for i, v := range values {
			lower[i] = strings.ToLower(v)
		}
		parts = append(parts, key+"="+url.QueryEscape(strings.Join(lower, "+")))
	}
	joined("sku_kind", q.SKUKinds)
	// The reference accepts "custom" (the bare-metal counterpart of cpu), but a v3.1.41 controller refuses it with
	// `compute_type must be "cpu" or "gpu"`: send it as cpu, once, keeping the order given.
	var types []string
	seen := map[string]bool{}
	for _, t := range q.ComputeTypes {
		t = strings.ToLower(t)
		if t == "custom" {
			t = "cpu"
		}
		if !seen[t] {
			seen[t] = true
			types = append(types, t)
		}
	}
	joined("compute_type", types)
	for _, i := range q.Images {
		parts = append(parts, "image="+url.QueryEscape(i))
	}
	for _, n := range []struct {
		key string
		val int
	}{{"cpu", q.CPU}, {"ram_gb", q.RAMGb}, {"storage_gb", q.StorageGb}, {"gpu_count", q.GPUCount}} {
		if n.val > 0 {
			parts = append(parts, n.key+"="+strconv.Itoa(n.val))
		}
	}
	if q.GPUType != "" {
		parts = append(parts, "gpu_type="+url.QueryEscape(q.GPUType))
	}
	return strings.Join(parts, "&")
}

func (c *mksMatchmakerClient) get(ctx context.Context, call string, q *MKSMatchQuery, out any) error {
	path := fmt.Sprintf(convert.MKSMatchmakerPath, url.PathEscape(c.project), call)
	if q != nil {
		if qs := q.query(); qs != "" {
			path += "?" + qs
		}
	}
	raw, err := c.rest.GetRaw(ctx, path)
	if err != nil {
		return err
	}
	if err := decodeLoose(raw, out); err != nil {
		return fmt.Errorf("decode %s answer: %w", call, err)
	}
	return nil
}

func needMatch(call string, q MKSMatchQuery, datacenters, kinds, computeTypes, images bool) error {
	missing := func(what string) error {
		return &gpupaas.APIError{StatusCode: 400, Message: fmt.Sprintf("%s needs %s", call, what)}
	}
	switch {
	case datacenters && len(q.Datacenters) == 0:
		return missing("at least one datacenter")
	case kinds && len(q.SKUKinds) == 0:
		return missing("sku_kind (bmaas, vmaas or both)")
	case computeTypes && len(q.ComputeTypes) == 0:
		return missing("compute_type (cpu, gpu or custom)")
	case images && len(q.Images) == 0:
		return missing("at least one image")
	}
	return nil
}

func (c *mksMatchmakerClient) KubernetesVersions(ctx context.Context) (*apiv1.MKSKubernetesVersions, error) {
	var out apiv1.MKSKubernetesVersions
	if err := c.get(ctx, "kubernetesversions", nil, &out); err != nil {
		return nil, err
	}
	if len(out.Minors) == 0 {
		for _, v := range out.Versions {
			out.Minors = append(out.Minors, v.Minor)
		}
	}
	return &out, nil
}

func (c *mksMatchmakerClient) AvailableOS(ctx context.Context, q MKSMatchQuery) (*apiv1.MKSAvailableOS, error) {
	if err := needMatch("availableos", q, true, true, true, false); err != nil {
		return nil, err
	}
	var out apiv1.MKSAvailableOS
	if err := c.get(ctx, "availableos", &q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *mksMatchmakerClient) AvailableImages(ctx context.Context, q MKSMatchQuery) ([]apiv1.MKSImageOption, error) {
	if err := needMatch("availableimages", q, true, true, true, false); err != nil {
		return nil, err
	}
	var out struct {
		Images []apiv1.MKSImageOption `json:"images"`
	}
	if err := c.get(ctx, "availableimages", &q, &out); err != nil {
		return nil, err
	}
	return out.Images, nil
}

func (c *mksMatchmakerClient) AvailableSpecs(ctx context.Context, q MKSMatchQuery) ([]apiv1.MKSSpecOption, error) {
	if err := needMatch("availablespecs", q, true, true, true, true); err != nil {
		return nil, err
	}
	var out struct {
		Specs []apiv1.MKSSpecOption `json:"specs"`
	}
	if err := c.get(ctx, "availablespecs", &q, &out); err != nil {
		return nil, err
	}
	return out.Specs, nil
}

func (c *mksMatchmakerClient) MatchSKUs(ctx context.Context, q MKSMatchQuery) ([]apiv1.MKSSKURef, error) {
	if err := needMatch("matchskus", q, true, true, true, false); err != nil {
		return nil, err
	}
	for _, t := range q.ComputeTypes {
		if strings.EqualFold(t, "gpu") && q.GPUType == "" {
			return nil, &gpupaas.APIError{StatusCode: 400, Message: "matchskus needs gpu_type when compute_type includes gpu"}
		}
	}
	var out struct {
		SKUs []apiv1.MKSSKURef `json:"skus"`
	}
	if err := c.get(ctx, "matchskus", &q, &out); err != nil {
		return nil, err
	}
	return out.SKUs, nil
}

func (c *mksMatchmakerClient) AvailableComputeSKUs(ctx context.Context, q MKSMatchQuery) ([]apiv1.MKSInstanceType, error) {
	if err := needMatch("availablecomputeskus", q, true, true, true, true); err != nil {
		return nil, err
	}
	var out struct {
		SKUs []apiv1.MKSInstanceType `json:"skus"`
	}
	if err := c.get(ctx, "availablecomputeskus", &q, &out); err != nil {
		return nil, err
	}
	return out.SKUs, nil
}

// decodeLoose decodes a discovery answer whose keys may be written camelCase (protobuf JSON) or snake_case
// (the proto field names): every key is read as camelCase. It is only for answers without free-form maps,
// where a key such as "cluster_name" is data and must not be renamed.
func decodeLoose(raw []byte, out any) error {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return err
	}
	b, err := json.Marshal(camelKeys(v))
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}

func camelKeys(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			out[snakeToCamel(k)] = camelKeys(val)
		}
		return out
	case []any:
		for i := range t {
			t[i] = camelKeys(t[i])
		}
	}
	return v
}

func snakeToCamel(s string) string {
	if !strings.Contains(s, "_") {
		return s
	}
	parts := strings.Split(s, "_")
	for i := 1; i < len(parts); i++ {
		if parts[i] != "" {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}
