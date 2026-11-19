# Contributing (SDK)

One file. Terse on purpose. For people and AI agents. Internal: no credentials here, ever.

`gpupaas-go` = Go SDK for the GPU PaaS platform API. Source of truth for what the platform speaks. Consumer:
`../terraform-provider-gpupaas` (check out beside this repo; its `go.mod` uses `replace ... => ../gpupaas-go`).
Other docs: `README.md` (resource catalog for users). Provider side: that repo's `CONTRIBUTING.md`. Releases: provider repo `RELEASING.md`.

---

## 1. Rules

1. **Envelope.** Every kind: `apiVersion`, `kind`, `metadata`, `spec`, `status`.
2. **Status is observed-only.** Strip it before every write. Never derive spec defaults from status.
3. **Backward compatible.** Never remove or rename a public Go identifier, field, method, kind, or JSON tag. Never change a
   kind's scope (project <-> workspace). Platform drops a field: ask first; default keep it, mark `// Deprecated:`.
4. **Casing.** `apis/` JSON tags are camelCase. Wire casing (often snake_case) lives only in `convert/`.
5. **Layers.** No HTTP in `apis/`. Wire = `convert/` + `rest/`. No `k8s.io/*` imports. No CLI or Terraform logic here.
6. **Context first.** Every method takes `ctx context.Context` first.
7. **Mocks mirror live.** What a live run teaches goes into the mock, with a test.
8. **No secrets in logs.** `rest/verbose.go` redacts; keep it that way.
9. **Every fix gets a test that fails without it.**
10. **Done = section 6 gate passes.**

---

## 2. Map

| Path | What |
|---|---|
| `config.go`, `options.go`, `errors.go`, `version.go` | Config (endpoint, key, org), call options, error helpers (`IsNotFound`, ...), `Version` + `ControllerAPI` |
| `apis/v1alpha1/types_*.go` | Kinds: spec, status, deep copy. Kind constants (`KindVMInstance`, ...) |
| `convert/dev_*.go`, `paas_*.go`, `paths.go` | Wire types + `ToX` / `FromX` converters; URL path constants. Fixtures: `convert/testdata/` |
| `clientset/typed/v1alpha1/interfaces.go` | One `<Kind>Interface` per kind; the contract the provider uses |
| `.../<name>_client.go`, `client.go` | Client per kind (dev kinds embed `devScopedClient`: project scope when workspace is empty) |
| `.../<name>_wait.go`, `wait_poll.go` | Wait helpers; one shared generic `pollUntil` |
| `validation/` | Required fields + scope defaults per kind |
| `rest/` | HTTP client, auth, retries, verbose log (secrets redacted) |
| `mock/{vmapi,bmapi,devapi,clusterapi}` | In-process mock platform APIs for tests (provider acceptance tests use them too) |
| `backend/`, `apply/`, `client/`, `engine/`, `watch/`, `runtime/` | Generic apply/delete engine. `apis/v1alpha1/register.go` registers **only MKSCluster**; other kinds use the typed clientset |
| `examples/` | One small program per feature |

Find API facts: Go types first, then the mocks, then a live controller (`/apis/.well-known/openapi.json`). Go code beats prose. Live beats mock.

---

## 3. Add or change a kind

1. **Plan.** Kind, scope (project, workspace, or both), paths, spec fields, status fields, sub-actions (start/stop/...),
   upsert or not. Unclear: ask. Dev kinds `POST` as upsert, so `Create` = `Get` first, 409 if it exists (details in step 6).
   **From a swagger / OpenAPI doc** (a controller serves one at `/apis/.well-known/openapi.json`): take kind, plural name, the
   paths per scope (`/projects/{p}/<plural>`, `/projects/{p}/workspaces/{w}/<plural>`, `.../{name}/<action>`), request body
   fields = spec, response-only fields = status, a sharing block if present, and whether create is documented as upsert.
   **Kind already exists?** Diff old vs doc: new field = safe, add. Changed type or rename = ask; keep the old field, add the new
   one. Field gone = **stop and ask**; default keep it with `// Deprecated:`. Treat the doc as input, never as the output format:
   do not generate code wholesale; fit it into the steps below.
2. **Types.** `apis/v1alpha1/types_<name>.go`: spec, status, deep copy, Kind constant. Register in `register.go` **only if** the
   apply engine must handle it.
3. **Paths.** `convert/paths.go`: collection + item + action paths, project and workspace forms.
4. **Wire.** `convert/dev_<name>.go`: wire structs + `ToX` / `FromX`. `ToX` strips status. Add fixtures in `convert/testdata/`.
   New field only: spec struct (camelCase tag, `omitempty`), wire struct (the API's casing: check a fixture or the controller doc),
   both converters, deep copy for maps and slices, a fixture, and the mock must store and echo it.
5. **Interface.** `<Kind>Interface` in `interfaces.go`; hook into `Interface` and `WorkspaceInterface`.
6. **Client.** `<name>_client.go`. Rules baked into every client:
   - `Create`: `Get` -> 409 if found -> `lockCreate` (per-name, per process) -> `POST`. MKS asks the server instead (`?fail-on-exists=true`).
   - Never write scope into the caller's object: copy it.
   - `Delete`: `DryRun` returns an error (API has no dry run). `Force` only for MKS.
   - Map missing objects through `IsNotFound` (it also knows the 502 `ent: ... not found` body).
7. **Wait helper** (slow kinds). `<name>_wait.go`: an options struct, then call `pollUntil`. Needs `Done` or `GoneIsDone`
   (else error at once), `Stable` polls in a row, tolerates transient errors (`MaxConsecutiveErrors`).
8. **Validation.** Add the kind to the scope switch in `validation/validation.go`.
9. **Backend** (rarely). `backend/remote/remote.go` dispatches MKS only. Add a kind only if generic Apply/Get/List/Delete needs it.
10. **Mock.** Add the endpoints to the right mock (`devapi`: workspaces, SSH keys, security groups, storage; `vmapi`; `bmapi`;
    `clusterapi`). Copy live quirks: upsert, stale reads, 502-as-not-found, failure causes, capacity. Add `Options` for failure injection.
11. **Tests.** Converter round trip (fixtures). Client against the mock: create / get / list / update / delete, 404, errors, waits,
    concurrency (`-race`).
12. **Example + README.** A program in `examples/`; a row and appendix entry in `README.md`.
13. **Version.** When a controller line freezes: bump `Version` and `ControllerAPI` together (`go test .` fails if they disagree).
    Release steps: provider repo `RELEASING.md`.

---

## 4. Rules that prevent real bugs

- **Waits:** use `pollUntil` (`clientset/typed/v1alpha1/wait_poll.go`); never hand-roll a loop. A wait with no `Done` and no `GoneIsDone` is an error, not an endless poll. A transient API error resets the stable streak.
- **Create race:** `lockCreate` only serializes inside one process. Two processes can still overwrite each other on dev kinds; only the server can stop that.
- **Not found:** call `IsNotFound`; never compare status codes. It treats 404, and 502 whose body matches `ent: ... not found`, as not found. A gateway 502 that merely says "not found" is **not**.
- **Options:** an option the API cannot honour must fail loudly (`DryRun` on delete), never be silently ignored.
- **Log redaction:** `rest/verbose.go` hides secret keys of any value type, `name`/`value` secret pairs, and `Authorization` / `Cookie` / `Set-Cookie`. New secret-looking field: add a test.
- **Config:** `OrganizationID` is trimmed in `Normalize`; the platform matches it exactly, case-sensitive.
- **Deep copy:** copy maps and slices, not just the struct (`MKSNode.DeepCopyObject`).

---

## 5. Mocks

Mocks serve the SDK's own tests **and** the provider's acceptance tests. A mock that is wrong breaks both. Change a mock:
add an option, keep the default as live behaves, test the option. Handy options: `vmapi.Options{Step, SSHKeys, NotFoundStatus}`,
`bmapi.Options{Capacity, DropMetadataText, UnsupportedActions}`. `devapi.New()` needs none.

---

## 6. Gate

```bash
gofmt -l .            # prints nothing
go vet ./...
go build ./...
go test ./...         # add -race for concurrency code
```
Then build the provider against your change: in `../terraform-provider-gpupaas`, `go build ./... && TF_ACC=1 go test ./... -count=1`.

**A fix needs a test that fails without it** (a new feature needs tests for the new behaviour). Stash only the source change (`git stash push -- <source files>`), run the test, see it fail, `git stash pop`.

Before handover:
- [ ] Tests for success, 404, error and (slow kinds) wait paths
- [ ] Mock updated for new behaviour; provider still builds and passes
- [ ] README row + example; no public rename or removal
- [ ] No secret in code, logs, fixtures, commits. No AI attribution lines in commits/PRs (team rule)

---

## 7. AI agents

**Do:** read this file, then the types of the kind you touch. Copy a neighbour (`ssh_key`, small; `vm_instance`, full). Small change, one concern.
Run the gate and report real output. Say what you did not verify.

**Ask a human before:** removing or renaming anything public; changing a scope; changing behaviour proven live (timings, waits,
teardown) without a live run; touching a shared test environment; commit, push, force-push, PR (unless asked).

**Never:** credentials in code, fixtures or logs; skip a failing check or weaken a test; work around a permission denial.

**Traps:** parallel commands share a working directory (use `git -C <repo>`, one repo-changing command at a time). Provider
tests use these mocks: always rebuild and test the provider after a mock or interface change.

---

## 8. Quirks

- Kinds on the `dev.envmgmt.io` API (`VirtualMachine`, `BareMetal`, `Storage`, `SshKey`, `SecurityGroup`) use upsert `POST`; no `PUT`.
  Their mocks: `vmapi` (VM), `bmapi` (bare metal), `devapi` (workspaces, SSH keys, security groups, storage).
- A backing compute instance exists for VM / bare metal / cluster (`ComputeInstance`); it outlives the record while destroying.
- Failed objects: first delete may fail once; parked `delete_pending` needs an admin (412 on retry).
- The platform may not store `description` / `display_name`; clients return what the server holds.
- A host may hold one VM; a second fails `No Capacity for VM allocation`.
