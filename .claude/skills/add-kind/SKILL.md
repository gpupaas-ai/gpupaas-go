---
name: add-kind
description: Add or change a kind (resource) in the gpupaas-go SDK: types, wire mapping, client, waits, validation, mock, tests, README. Use when the platform API gained or changed a resource, or when fixing an SDK defect. The Terraform provider builds on this.
---

# add-kind

Procedure only. **The rules live in `CONTRIBUTING.md`: read it first** (sections 1, 3, 4, 6). Do not restate or override them here.
The user's request follows this text.

## 1. Preflight

Run `.claude/skills/add-kind/preflight.sh`. It checks Go, the build, the branch, and whether the provider is checked out
beside this repo (needed to prove the provider still builds). `MISSING` = fix or tell the user, then stop. Uncommitted changes you did not make: keep them out of your change, tell the user, and do not stash, commit or discard them.

## 2. Gather (ask only what is missing)

- New kind, new field, new action, or a fix? Scope (project, workspace, or both)? Slow (needs a wait helper)?
- From a swagger / OpenAPI doc? Take kind, paths per scope, spec fields (request bodies), status fields (response-only),
  sharing block, upsert or not (`CONTRIBUTING.md` section 3, step 1).
- Kind already exists? Diff old against new. New field: add. Changed or renamed: ask, keep the old and add the new.
  **Field gone: stop and ask.** Default: keep it, mark `// Deprecated:`.

## 3. Pick a neighbour

Small: `ssh_key` (`clientset/typed/v1alpha1/ssh_key_client.go`, `convert/dev_ssh_key.go`). Full, with waits and actions:
`vm_instance`. Say which one you copy.

## 4. Implement

Follow `CONTRIBUTING.md` section 3 in order: plan, types (+ Kind constant), paths, wire converters with fixtures, interface,
client (create rules, delete options, `IsNotFound`), wait helper via `pollUntil`, validation scope, mock, tests, example, README.
Apply section 4 as you go. Keep every change backward compatible.

## 5. Tests

Converter round trip with fixtures. Client against the mock: create, get, list, update, delete, 404, errors, waits and
concurrency (`-race`). For a fix: write the test, stash only the source change, see it fail, restore.
The mock must reproduce live behaviour you rely on; add an option instead of changing its default.

## 6. Checks

Run the gate in `CONTRIBUTING.md` section 6. Then build and test the provider against your change:
in `../terraform-provider-gpupaas`, `go build ./... && TF_ACC=1 go test ./... -count=1` (needs `terraform` or `tofu` on PATH, no credentials; mirror new interface methods in
`../terraform-provider-gpupaas/internal/client/fake.go` and `fake_compute.go`). Never weaken a test to pass.

## 7. Report

State: what changed (files), each check with its real result (including the provider build), and a **Not verified** list
(no live run, anything skipped). Open questions last.

## Stop and ask first

Removing or renaming anything public; changing a scope; changing behaviour proven live; touching a shared test
environment; committing, pushing or opening a PR. Never: secrets in code, fixtures or logs; AI attribution lines in commits
or PRs; working around a permission denial. No commit unless asked.
