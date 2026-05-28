# Tasks: gpupaas-go SDK

Implement in order. Mark tasks complete as you finish them. Checkpoint after phase 4, phase 8, and phase 11.

Reference: [`paasctl/examples/go-client-basic`](https://github.com/RafaySystems/paasctl/blob/main/examples/go-client-basic/main.go).

- [ ] 1. Repository bootstrap
  - [ ] 1.1 Initialize `go.mod` with module `github.com/gpupaas-ai/gpupaas-go`, Go 1.22+
  - [ ] 1.2 Add `LICENSE`, stub `README.md`, `version.go`
  - [ ] 1.3 Add root packages: `config.go`, `options.go`, `errors.go`
  - _Requirements: 7.1–7.4_

- [ ] 2. API types (`apis/v1alpha1`)
  - [ ] 2.1 Implement `TypeMeta`, `ObjectMeta`, list meta in `types.go`
  - [ ] 2.2 Implement `Project`, `ProjectList`, `ProjectSpec`, `ProjectStatus`
  - [ ] 2.3 Implement `Workspace`, `WorkspaceList`, spec/status
  - [ ] 2.4 Add `register.go` with GVK constants and default registration helpers
  - [ ] 2.5 Implement `GetProject`, `GetWorkspace`, `SetProject`, `SetWorkspace`, `DeepCopyObject` on each kind
  - _Requirements: 1.1–1.7_

- [ ] 3. Runtime scheme and codec
  - [ ] 3.1 Define `runtime.Object`, `ObjectList`, and `GroupVersionKind`
  - [ ] 3.2 Implement `Scheme` with type registration and `New(gvk)`
  - [ ] 3.3 Implement decode: JSON, YAML, multi-document YAML
  - [ ] 3.4 Implement encode: JSON and YAML
  - [ ] 3.5 Add unit tests for decode, encode, unknown kind
  - _Requirements: 2.1–2.6_

- [ ] **Checkpoint A:** `go test ./runtime/...` passes

- [ ] 4. Validation
  - [ ] 4.1 Implement `validation.Options` with `DefaultProject`, `DefaultWorkspace`
  - [ ] 4.2 Implement `ValidateObject` — require apiVersion/kind/name; apply project default
  - [ ] 4.3 Add unit tests for default injection and missing project errors
  - _Requirements: 4.1–4.5_

- [ ] 5. Backend abstraction
  - [ ] 5.1 Define `backend.Backend` interface (Apply, Get, List, Delete)
  - [ ] 5.2 Implement `backend/memory` with in-process map keyed by GVK + scope + name
  - [ ] 5.3 Add memory backend tests: apply create, apply update, list, delete
  - _Requirements: 3.2, 8.1_

- [ ] 6. Engine
  - [ ] 6.1 Implement `engine.Engine` wiring validation + backend
  - [ ] 6.2 Implement Apply/Get/List/Delete dispatch
  - [ ] 6.3 Add unit tests using memory backend
  - _Requirements: 3.4 (via engine)_

- [ ] 7. High-level client
  - [ ] 7.1 Implement `client.Options` with `UseMemory` and `Config`
  - [ ] 7.2 Implement `client.New` — select memory or remote backend
  - [ ] 7.3 Expose `Apply`, `Get`, `List`, `Delete` with `(project, workspace)` args
  - [ ] 7.4 Expose `Scheme` on client for decode helpers
  - [ ] 7.5 Add client tests mirroring go-client-basic flow
  - _Requirements: 3.1–3.6_

- [ ] **Checkpoint B:** `client` + memory backend tests pass (go-client-basic shape)

- [ ] 8. REST client and typed clientset (remote backend)
  - [ ] 8.1 Implement `rest/` with Bearer auth and `APIError` mapping
  - [ ] 8.2 Implement `clientset` — `Projects()`, `Workspaces(project)` CRUD + List
  - [ ] 8.3 Implement `backend/remote` delegating to clientset
  - [ ] 8.4 Add httptest-based remote backend and clientset tests
  - _Requirements: 5.1–5.5, 7.1–7.2_

- [ ] 9. Apply and delete helpers
  - [ ] 9.1 Implement `ApplyReader`, `ApplyFile`, `DeleteReader`, `DeleteFile` using `client.Client`
  - [ ] 9.2 Multi-document YAML; preserve status exclusion; `IgnoreNotFound` on delete
  - [ ] 9.3 Add unit tests for file apply/delete
  - _Requirements: 6.1–6.5_

- [ ] 10. Watch types (minimal)
  - [ ] 10.1 Define `EventType`, `Event`, `Interface`
  - [ ] 10.2 Add fake watcher for tests
  - _Requirements: out of scope note — types only_

- [ ] 11. Examples
  - [ ] 11.1 `examples/typed_client/main.go` — mirror paasctl go-client-basic (Apply + List with `UseMemory: true`)
  - [ ] 11.2 `examples/apply_yaml/main.go` — apply from file via client
  - [ ] 11.3 `examples/delete_yaml/main.go` — delete from file via client
  - [ ] 11.4 Verify `go build ./examples/...`
  - _Requirements: 9.1–9.4_

- [ ] 12. README and docs
  - [ ] 12.1 Document high-level client, env vars, platform scoping, YAML apply/delete
  - [ ] 12.2 Document error handling and v1alpha1 stability note
  - [ ] 12.3 Document relationship to paasctl, terraform-provider, paasctl reference
  - _Requirements: 9.3_

- [ ] 13. Final verification
  - [ ] 13.1 Run `go mod tidy`
  - [ ] 13.2 Run `go fmt ./...`
  - [ ] 13.3 Run `go test ./...`
  - [ ] 13.4 Fix all compile and test failures
  - _Requirements: acceptance criteria_

- [ ] **Checkpoint C:** full test suite green; examples compile
