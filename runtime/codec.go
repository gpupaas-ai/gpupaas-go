package runtime

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Serializer decodes and encodes API objects.
type Serializer struct {
	scheme *Scheme
}

// NewSerializer creates a serializer for the given scheme.
func NewSerializer(scheme *Scheme) *Serializer {
	return &Serializer{scheme: scheme}
}

// Scheme returns the underlying scheme.
func (s *Serializer) Scheme() *Scheme {
	return s.scheme
}

// Decode parses a single JSON or YAML document into a typed Object.
func (s *Serializer) Decode(data []byte) (Object, error) {
	var meta struct {
		APIVersion string `json:"apiVersion" yaml:"apiVersion"`
		Kind       string `json:"kind" yaml:"kind"`
	}
	if err := decodeMeta(data, &meta); err != nil {
		return nil, err
	}
	if meta.APIVersion == "" || meta.Kind == "" {
		return nil, fmt.Errorf("document missing apiVersion or kind")
	}

	gv, err := ParseGroupVersion(meta.APIVersion)
	if err != nil {
		return nil, err
	}
	gvk := GroupVersionKind{Group: gv.Group, Version: gv.Version, Kind: meta.Kind}
	if !s.scheme.Recognizes(gvk) {
		return nil, fmt.Errorf("unknown apiVersion/kind: %s", gvk)
	}

	obj, err := s.scheme.New(gvk)
	if err != nil {
		return nil, err
	}
	if err := unmarshalAny(data, obj); err != nil {
		return nil, err
	}
	return obj, nil
}

// DecodeAll parses multi-document YAML (--- separated) or a single document.
func (s *Serializer) DecodeAll(data []byte) ([]Object, error) {
	docs := splitDocuments(data)
	if len(docs) == 0 {
		return nil, fmt.Errorf("no documents found")
	}
	out := make([]Object, 0, len(docs))
	for i, doc := range docs {
		obj, err := s.Decode(doc)
		if err != nil {
			return nil, fmt.Errorf("document %d: %w", i+1, err)
		}
		out = append(out, obj)
	}
	return out, nil
}

// Encode writes obj as indented JSON.
func (s *Serializer) Encode(obj Object) ([]byte, error) {
	return json.MarshalIndent(obj, "", "  ")
}

// EncodeYAML writes obj as YAML.
func (s *Serializer) EncodeYAML(obj Object) ([]byte, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	var doc interface{}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	return yaml.Marshal(doc)
}

func splitDocuments(data []byte) [][]byte {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil
	}
	parts := strings.Split(string(trimmed), "\n---")
	var docs [][]byte
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		docs = append(docs, []byte(part))
	}
	return docs
}

func decodeMeta(data []byte, meta interface{}) error {
	if err := yaml.Unmarshal(data, meta); err == nil {
		return nil
	}
	return json.Unmarshal(data, meta)
}

func unmarshalAny(data []byte, out interface{}) error {
	if err := yaml.Unmarshal(data, out); err == nil {
		return nil
	}
	var generic interface{}
	if err := yaml.Unmarshal(data, &generic); err == nil {
		j, err := json.Marshal(generic)
		if err != nil {
			return err
		}
		return json.Unmarshal(j, out)
	}
	return json.Unmarshal(data, out)
}
