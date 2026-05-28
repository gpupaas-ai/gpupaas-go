You are implementing a production-quality Go SDK repository for GPU PaaS.

Repository:
github.com/gpupaas/gpupaas-go

Goal:
Create a reusable Go client library for the gpupaas.ai API. This SDK will be used by:
1. The official CLI: paasctl
2. The Terraform/OpenTofu provider: terraform-provider-gpupaas
3. External automation programs written by users

The SDK should feel similar in structure and usability to Kubernetes client-go, but much smaller and simpler. The user should be able to write Go programs that create, read, update, delete, list, watch, and apply declarative resource specs.

The SDK uses a Kubernetes-inspired resource shape (apiVersion, kind, metadata, spec) for familiarity, but all objects are scoped to the gpupaas platform — projects, workspaces, and (later) project+workspace workload resources.

Core design principles:
- The CLI and Terraform provider must depend on this SDK, not duplicate API logic.
- The SDK must expose typed clients.
- The SDK must support YAML and JSON specs.
- The SDK must support Kubernetes-like resource objects with apiVersion, kind, metadata, and spec.
- The SDK must support Apply and Delete operations based on declarative specs.
- The SDK must be testable without a real server.
- The SDK must be extensible for future resource types.
- The public API should be stable and clean.
- Avoid unnecessary dependencies.
- Use idiomatic Go.

Use Go module:
module github.com/gpupaas-ai/gpupaas-go

Minimum Go version:
go 1.22

Create this repository structure:

.
├── go.mod
├── README.md
├── LICENSE
├── config.go
├── errors.go
├── options.go
├── version.go
├── apis/
│   └── v1alpha1/
│       ├── doc.go
│       ├── types.go
│       ├── register.go
│       ├── project.go
│       └── workspace.go
├── client/
│   └── client.go
├── engine/
│   └── engine.go
├── validation/
│   └── validation.go
├── backend/
│   ├── backend.go
│   ├── memory/
│   │   └── memory.go
│   └── remote/
│       └── remote.go
├── clientset/
│   ├── clientset.go
│   └── typed/
│       └── v1alpha1/
│           ├── client.go
│           ├── project.go
│           └── workspace.go
├── runtime/
│   ├── object.go
│   ├── scheme.go
│   ├── codec.go
│   └── gvk.go
├── apply/
│   ├── apply.go
│   └── delete.go
├── rest/
│   ├── client.go
│   └── fake.go
├── watch/
│   ├── watch.go
│   └── event.go
├── examples/
│   ├── typed_client/main.go
│   ├── apply_yaml/main.go
│   └── delete_yaml/main.go
└── internal/
    └── testutil/
        └── server.go

Implement the SDK with the following requirements.

## 1. Resource model

Create Kubernetes-like base types in apis/v1alpha1/types.go.

Every resource must have:

type TypeMeta struct {
    APIVersion string `json:"apiVersion" yaml:"apiVersion"`
    Kind       string `json:"kind" yaml:"kind"`
}

type ObjectMeta struct {
    Name        string            `json:"name" yaml:"name"`
    Project     string            `json:"project,omitempty" yaml:"project,omitempty"`
    Workspace   string            `json:"workspace,omitempty" yaml:"workspace,omitempty"`
    Labels      map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
    Annotations map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
}

Every API object must follow this pattern:

type Workspace struct {
    TypeMeta   `json:",inline" yaml:",inline"`
    Metadata   ObjectMeta `json:"metadata" yaml:"metadata"`
    Spec       WorkspaceSpec    `json:"spec" yaml:"spec"`
    Status     WorkspaceStatus  `json:"status,omitempty" yaml:"status,omitempty"`
}

Implement these initial resources:

Project:
- apiVersion: gpupaas.ai/v1alpha1
- kind: Project
- metadata.name
- spec.displayName
- spec.description

Workspace:
- apiVersion: gpupaas.ai/v1alpha1
- kind: Workspace
- metadata.name
- metadata.project is the owning project name
- spec.description

Future workload resources (not in v1):
- metadata.project and metadata.workspace required
- scoped under /projects/{project}/workspaces/{workspace}/...

## 2. Runtime object system

Implement package runtime.

Define interface:

type Object interface {
    GetAPIVersion() string
    GetKind() string
    GetName() string
    GetProject() string
    GetWorkspace() string
    SetProject(string)
    SetWorkspace(string)
    DeepCopyObject() Object
}

type GroupVersionKind struct {
    Group   string
    Version string
    Kind    string
}

Also define:

type ObjectList interface {
    Object
    GetItems() []Object
}

Implement a Scheme similar to Kubernetes, but simpler.

Scheme should:
- Register object constructors by apiVersion and kind
- Decode YAML or JSON into typed Go structs
- Return runtime.Object
- Encode runtime.Object to JSON or YAML
- Support multi-document YAML
- Validate required fields: apiVersion, kind, metadata.name
- Return clear errors for unknown apiVersion/kind

Implement support for these formats:
- JSON
- YAML
- Multi-document YAML separated by ---

Use dependency:
gopkg.in/yaml.v3

## 2b. High-level client, engine, validation, backend

Primary consumer API modeled on paasctl/examples/go-client-basic.

Implement package client:

type Options struct {
    Config    gpupaas.Config
    UseMemory bool
}

func New(opts Options) *Client

func (c *Client) Apply(ctx context.Context, obj runtime.Object, project, workspace string) (runtime.Object, error)
func (c *Client) Get(ctx context.Context, gvk runtime.GroupVersionKind, project, workspace, name string) (runtime.Object, error)
func (c *Client) List(ctx context.Context, gvk runtime.GroupVersionKind, project, workspace string) ([]runtime.Object, error)
func (c *Client) Delete(ctx context.Context, gvk runtime.GroupVersionKind, project, workspace, name string) error

When UseMemory is true, use backend/memory (no live API). Otherwise use backend/remote with clientset.

Implement package validation:

type Options struct {
    DefaultProject   string
    DefaultWorkspace string
}

func ValidateObject(obj runtime.Object, opts Options) error

Rules:
- Require apiVersion, kind, metadata.name
- If metadata.project empty, set from DefaultProject; error if still empty
- For Workspace, do not require metadata.workspace
- For future project+workspace kinds, set metadata.workspace from DefaultWorkspace when empty

Implement package engine to call validation then backend.

Implement package backend with memory and remote implementations.

Reference program (examples/typed_client must match):

c := client.New(client.Options{UseMemory: true})
ws := &v1alpha1.Workspace{... Metadata: ObjectMeta{Name: "example", Project: "demo"}}
applied, err := c.Apply(ctx, ws, "demo", "")
items, err := c.List(ctx, gvk, "demo", "")

## 3. REST client

Implement package rest.

The rest client should wrap net/http.

Config:

type Config struct {
    BaseURL    string
    Token      string
    UserAgent  string
    HTTPClient *http.Client
}

Defaults:
- UserAgent: gpupaas-go/<version>
- HTTP timeout: 30 seconds
- Authentication: Bearer token if token is provided

Implement methods:

func NewClient(config Config) (*Client, error)

func (c *Client) Get(ctx context.Context, path string, out any) error
func (c *Client) Post(ctx context.Context, path string, in any, out any) error
func (c *Client) Put(ctx context.Context, path string, in any, out any) error
func (c *Client) Patch(ctx context.Context, path string, in any, out any) error
func (c *Client) Delete(ctx context.Context, path string, out any) error

Requirements:
- Use context.Context everywhere
- Encode requests as JSON
- Decode responses as JSON
- Treat 2xx as success
- For non-2xx, return typed APIError
- Include status code, request id if available, and response body in error
- Do not log secrets
- Do not print anything from the SDK

## 4. Typed clientset

Implement package clientset.

Clientset should expose:

type Interface interface {
    V1alpha1() v1alpha1.Interface
}

func NewForConfig(config gpupaas.Config) (*Clientset, error)

The typed v1alpha1 client should expose:

type Interface interface {
    Projects() ProjectInterface
	Workspaces(project string) WorkspaceInterface
}

Each resource interface should expose:

Create(ctx context.Context, obj *Type, opts CreateOptions) (*Type, error)
Get(ctx context.Context, name string, opts GetOptions) (*Type, error)
Update(ctx context.Context, obj *Type, opts UpdateOptions) (*Type, error)
Delete(ctx context.Context, name string, opts DeleteOptions) error
List(ctx context.Context, opts ListOptions) (*TypeList, error)

For project-scoped resources, include the project name in the URL path. For project+workspace-scoped resources (future), include both in the path.

Suggested REST paths:

Projects:
GET    /apis/gpupaas.ai/v1alpha1/projects
POST   /apis/gpupaas.ai/v1alpha1/projects
GET    /apis/gpupaas.ai/v1alpha1/projects/{name}
PUT    /apis/gpupaas.ai/v1alpha1/projects/{name}
DELETE /apis/gpupaas.ai/v1alpha1/projects/{name}

Workspace:
GET    /apis/gpupaas.ai/v1alpha1/projects/{project}/workspaces
POST   /apis/gpupaas.ai/v1alpha1/projects/{project}/workspaces
GET    /apis/gpupaas.ai/v1alpha1/projects/{project}/workspaces/{name}
PUT    /apis/gpupaas.ai/v1alpha1/projects/{project}/workspaces/{name}
DELETE /apis/gpupaas.ai/v1alpha1/projects/{project}/workspaces/{name}

Use url.PathEscape for names .

## 5. Apply package

Implement package apply as helpers over client.Client (not a separate Applier+clientset entry point).

Expose:

func ApplyReader(ctx context.Context, c *client.Client, r io.Reader, project, workspace string) error
func ApplyFile(ctx context.Context, c *client.Client, path string, project, workspace string) error
func DeleteReader(ctx context.Context, c *client.Client, r io.Reader, project, workspace string, ignoreNotFound bool) error
func DeleteFile(ctx context.Context, c *client.Client, path string, project, workspace string, ignoreNotFound bool) error

Apply semantics:
- If object does not exist, create it.
- If object exists, update it.
- Do not modify status.
- Work with JSON or YAML specs.
- Work with multi-document YAML.
- Validate object before apply.
- Return useful errors with apiVersion/kind/project/workspace/name context as applicable.

Delete semantics:
- Delete object by apiVersion/kind and metadata scope (name; project and workspace when required by kind).
- If NotFound and IgnoreNotFound is true, do not return error.
- Work with JSON or YAML specs.
- Work with multi-document YAML.

ApplyOptions:
- DryRun bool
- FieldManager string
- Validate bool

DeleteOptions:
- IgnoreNotFound bool
- DryRun bool

## 6. Watch package

Create a simple watch package, but keep implementation basic.

Define:

type EventType string

const (
    Added    EventType = "ADDED"
    Modified EventType = "MODIFIED"
    Deleted  EventType = "DELETED"
    Error    EventType = "ERROR"
)

type Event struct {
    Type   EventType      `json:"type"`
    Object runtime.Object `json:"object"`
}

type Interface interface {
    Stop()
    ResultChan() <-chan Event
}

It is okay to implement only the types and a simple fake watcher for tests in the first version.

## 7. Options

Create root package options:

type CreateOptions struct {
    DryRun bool
}

type UpdateOptions struct {
    DryRun bool
}

type DeleteOptions struct {
    DryRun bool
    IgnoreNotFound bool
}

type GetOptions struct {}

type ListOptions struct {
    Limit string
    Continue string
    LabelSelector string
}

## 8. Error handling

Implement APIError in errors.go:

type APIError struct {
    StatusCode int
    Reason     string
    Message    string
    RequestID  string
    Body       string
}

Methods:
- Error() string
- IsNotFound() bool
- IsConflict() bool
- IsUnauthorized() bool
- IsForbidden() bool
- IsServerError() bool

Also expose helper funcs:
func IsNotFound(err error) bool
func IsConflict(err error) bool
func IsUnauthorized(err error) bool
func IsForbidden(err error) bool

Use errors.As.

## 9. Config

Root package should expose:

type Config struct {
    Endpoint   string
    Token      string
    UserAgent  string
    HTTPClient *http.Client
}

func NewConfig(endpoint, token string) Config

func ConfigFromEnv() Config

ConfigFromEnv should read:
- GPUPAAS_ENDPOINT
- GPUPAAS_TOKEN

Default endpoint:
https://api.gpupaas.ai

## 10. Examples

Create examples that compile.

Example 1: examples/typed_client/main.go

Mirror paasctl/examples/go-client-basic:

c := client.New(client.Options{UseMemory: true})
Apply a Workspace under project "demo"
List workspaces in project "demo"

Example 2: examples/apply_yaml/main.go

Read a YAML file from path argument.
Apply using apply.ApplyReader with client.New(client.Options{UseMemory: true}) or ConfigFromEnv().

Example 3: examples/delete_yaml/main.go

Read a YAML file from path argument.
Delete using apply.DeleteReader with ignoreNotFound support.

## 11. README

Create a high-quality README with:

- Project title: gpupaas-go
- Description
- Installation
- Environment variables
- Typed client usage (client.New, Apply, List)
- Apply YAML usage (apply.ApplyReader with client)
- Delete YAML usage
- Resource YAML examples
- Error handling example
- Relationship to:
  - paasctl
  - terraform-provider-gpupaas
  - gpupaas.ai API
- Compatibility note:
  This SDK is intended to be used by Terraform, OpenTofu, CLI, and automation programs.
- Stability note:
  APIs are v1alpha1 and may change until v1beta1/v1.

Include example YAML:


## 12. Testing

Create unit tests for:
- YAML decode
- JSON decode
- multi-document YAML decode
- unknown kind error
- required metadata.name validation
- memory backend apply create/update/list/delete
- client Apply + List (go-client-basic flow)
- validation default project injection
- REST client success
- REST client non-2xx error
- typed Project client create/get/list/delete
- Apply-from-file create path
- Apply-from-file update path
- Delete ignore not found path

Use httptest for HTTP tests.

Tests should run with:

go test ./...

## 13. Code quality

Run and satisfy:

go fmt ./...
go test ./...

Do not leave TODO placeholders in core functionality.
Do not create fake implementations where real code is simple.
Use idiomatic Go.
Keep public APIs documented with comments where appropriate.
Avoid global mutable state except for a default scheme if necessary.

## 14. Public API expectations

The primary user program should compile (same shape as paasctl go-client-basic):

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
    c := client.New(client.Options{UseMemory: true})
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

Also this apply-from-file program should compile:

package main

import (
    "context"
    "os"

    "github.com/gpupaas-ai/gpupaas-go/apply"
    "github.com/gpupaas-ai/gpupaas-go/client"
)

func main() {
    ctx := context.Background()
    c := client.New(client.Options{UseMemory: true})

    f, err := os.Open("project.yaml")
    if err != nil {
        panic(err)
    }
    defer f.Close()

    if err := apply.ApplyReader(ctx, c, f, "demo", ""); err != nil {
        panic(err)
    }
}

## 15. Implementation sequence

Implement in this order:

1. go.mod
2. root config/options/errors/version
3. API types
4. runtime scheme and codec
5. validation
6. backend (memory first)
7. engine
8. client (UseMemory path — enables go-client-basic example)
9. rest client
10. typed clientset
11. backend/remote
12. apply/delete file helpers
13. watch types
14. examples
15. tests
16. README

After implementation, run:

go mod tidy
go fmt ./...
go test ./...

Fix all compile and test failures.

Deliver a complete working initial commit for github.com/gpupaas/gpupaas-go.