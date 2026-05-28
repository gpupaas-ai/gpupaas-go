# Steering: gpupaas-go

AI steering documents for implementing `github.com/gpupaas-ai/gpupaas-go`.

## Always read first

1. [project-overview.md](./project-overview.md) — purpose, consumers, API group
2. [architecture-patterns.md](./architecture-patterns.md) — client/engine/backend layering, apply pattern
3. [coding-standards.md](./coding-standards.md) — Go conventions, testing, security

## Spec workflow

Before coding, read the parent spec files in order:

1. [../requirements.md](../requirements.md)
2. [../design.md](../design.md)
3. [../tasks.md](../tasks.md) — follow task list; mark complete as you go

Legacy full prompt (same scope): [../../requirements.md](../../requirements.md)

## Source prompt

Original implementation prompt: `gpupaas-go/specs/requirements.md` in the gpupaas-go repository.

## Reference

- `paasctl/examples/go-client-basic/main.go` — canonical typed client example
- `paasctl/pkg/client`, `paasctl/pkg/engine`, `paasctl/pkg/validation`, `paasctl/pkg/backend/memory`
- `paasctl/specs/kubectl_style_cli_architecture_prompt.md` — kubectl-style CLI/SDK architecture
- `terraform-provider-gpupaas/specs/terraform-provider/` — provider specs (depends on this SDK)

## Inclusion hints

| File | When to include |
|---|---|
| project-overview.md | Always (`inclusion: auto`) |
| architecture-patterns.md | Always |
| coding-standards.md | Always |

Copy these files to `gpupaas-go/.kiro/steering/` or `.cursor/rules/` when wiring IDE rules for the gpupaas-go repo directly.
