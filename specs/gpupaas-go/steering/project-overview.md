---
inclusion: auto
---

# gpupaas-go — Project Overview

## Purpose

`github.com/gpupaas-ai/gpupaas-go` is the public Go SDK for [gpupaas.ai](https://gpupaas.ai). It provides typed clients, declarative apply/delete, and YAML/JSON resource decoding for GPU PaaS automation.

## Consumers

| Consumer | Repository | Role |
|---|---|---|
| CLI | `paasctl` | kubectl-style commands (`apply`, `get`, `delete`) |
| IaC | `terraform-provider-gpupaas` | Terraform/OpenTofu provider |
| User code | external | Custom automation programs |

**Rule:** CLI and Terraform provider MUST depend on this SDK. They must NOT duplicate HTTP or API logic.

## Design inspiration

Kubernetes client-go, simplified — **shape only**, not Kubernetes scope:

- High-level `client` package with Apply/Get/List/Delete (see [`paasctl/examples/go-client-basic`](https://github.com/RafaySystems/paasctl/blob/main/examples/go-client-basic/main.go))
- `engine` + `validation` + pluggable `backend` (`memory` for tests, `remote` for API)
- Runtime scheme for encode/decode
- Declarative resource objects (`apiVersion`, `kind`, `metadata`, `spec`, `status`)
- Apply semantics (create or update from desired state)

Platform scope is gpupaas-native: **projects** contain **workspaces**; future resources are **project + workspace** scoped.

References:

- `paasctl/examples/go-client-basic/main.go` — canonical typed client example
- `paasctl/pkg/client`, `paasctl/pkg/engine`, `paasctl/pkg/validation`
- `paasctl/specs/kubectl_style_cli_architecture_prompt.md`

## API group

```text
apiVersion: gpupaas.ai/v1alpha1
```

Initial kinds: `Project`, `Workspace`

## Module

```text
module github.com/gpupaas-ai/gpupaas-go
go 1.22
```

## Configuration env vars

| Variable | Default | Description |
|---|---|---|
| `GPUPAAS_ENDPOINT` | `https://api.gpupaas.ai` | API base URL |
| `GPUPAAS_TOKEN` | — | Bearer token |

## Common commands

```bash
go mod tidy
go fmt ./...
go test ./...
go build ./examples/...
```

## Spec location

Implementation specs and steering for this SDK:

```text
gpupaas-go/specs/gpupaas-go/
├── requirements.md
├── design.md
├── tasks.md
└── steering/
```

When implementing in `gpupaas-go`, read `requirements.md` → `design.md` → `tasks.md` in that order.

## Implementation order relative to provider

```text
1. gpupaas-go (this SDK)     ← implement first
2. terraform-provider-gpupaas ← consumes SDK
3. paasctl                 ← consumes SDK
```
