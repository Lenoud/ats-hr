# Consul Topology And Doc Normalization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make local Consul registration/discovery behavior consistent for host-run services plus Docker Consul, and normalize valuable untracked ATS docs into a maintainable docs structure.

**Architecture:** Keep the existing shared Consul naming abstraction and add only the minimum runtime/config/documentation changes needed to make local topology behavior explicit and verifiable. Normalize docs by separating formal docs, test/experiment docs, and superpowers planning/design docs, then add a small index so future additions follow the same structure.

**Tech Stack:** Go, Gin, Consul API, Markdown docs, curl

---

### Task 1: Audit Current Consul Runtime Boundaries

**Files:**
- Review: `ats-platform/internal/shared/consul/register.go`
- Review: `ats-platform/internal/shared/consul/naming.go`
- Review: `ats-platform/cmd/resume-service/main.go`
- Review: `ats-platform/cmd/interview-service/main.go`
- Review: `ats-platform/cmd/search-service/main.go`
- Review: `ats-platform/cmd/gateway/discovery.go`
- Review: `ats-platform/cmd/gateway/main.go`

- [ ] **Step 1: Re-read the current shared Consul helpers**

Run:

```bash
sed -n '1,220p' ats-platform/internal/shared/consul/register.go
sed -n '1,220p' ats-platform/internal/shared/consul/naming.go
```

Expected outcome:
- Confirm current address resolution, endpoint naming, and service ID generation behavior before changing anything.

- [ ] **Step 2: Re-read service startup code paths**

Run:

```bash
sed -n '120,260p' ats-platform/cmd/resume-service/main.go
sed -n '110,240p' ats-platform/cmd/interview-service/main.go
sed -n '140,260p' ats-platform/cmd/search-service/main.go
```

Expected outcome:
- Confirm exactly where `SERVICE_ADDRESS`, registration, and deregistration are wired today.

- [ ] **Step 3: Re-read gateway discovery behavior**

Run:

```bash
sed -n '1,220p' ats-platform/cmd/gateway/discovery.go
sed -n '1,260p' ats-platform/cmd/gateway/main.go
```

Expected outcome:
- Confirm current `host.docker.internal` normalization behavior and route discovery assumptions.

- [ ] **Step 4: Capture current docs inventory before edits**

Run:

```bash
find ats-platform/docs -maxdepth 3 -type f | sort
```

Expected outcome:
- Confirm which docs are already tracked and where new or untracked docs should be moved.

- [ ] **Step 5: Commit only if audit reveals required code changes are broader than expected**

Expected outcome:
- No commit in this task unless an audit-only note file is intentionally introduced, which is unlikely.

### Task 2: Make Consul Local Topology Behavior Explicit In Code And Scripts

**Files:**
- Modify: `ats-platform/internal/shared/consul/register.go`
- Modify: `ats-platform/cmd/gateway/discovery.go`
- Modify: `ats-platform/scripts/run-services.sh`
- Modify: `ats-platform/Makefile`
- Optional modify: `ats-platform/cmd/resume-service/main.go`
- Optional modify: `ats-platform/cmd/interview-service/main.go`
- Optional modify: `ats-platform/cmd/search-service/main.go`

- [ ] **Step 1: Keep address resolution rules centralized**

Review whether `ResolveServiceAddress(...)` in `ats-platform/internal/shared/consul/register.go` already matches the approved spec.

Expected outcome:
- No new address-resolution helper is introduced elsewhere.

- [ ] **Step 2: Add explicit local-dev environment support to service orchestration**

Update `ats-platform/scripts/run-services.sh` so local host-run services can be started with a shared `SERVICE_ADDRESS` override in the Docker-Consul development case.

Implementation notes:
- Use a single env var source such as `SERVICE_ADDRESS="${SERVICE_ADDRESS:-host.docker.internal}"` only for local orchestrated runs where that is the intended default.
- Do not hardcode this behavior inside service binaries as the only path.

Expected outcome:
- Local orchestration uses one explicit service address convention instead of relying on implicit outbound IP detection.

- [ ] **Step 3: Surface the orchestration behavior through Makefile targets**

Update `ats-platform/Makefile` so the recommended local run path makes the topology assumption visible.

Expected outcome:
- Developers can discover the supported local run mode from the existing command entrypoints.

- [ ] **Step 4: Keep gateway normalization narrow**

Review `ats-platform/cmd/gateway/discovery.go` and ensure normalization is limited to the documented local-dev special case rather than becoming a general address rewrite mechanism.

Expected outcome:
- No extra discovery abstraction is introduced; only the approved local fallback remains.

- [ ] **Step 5: Re-check service binaries for duplicated topology logic**

Run:

```bash
rg -n "SERVICE_ADDRESS|host.docker.internal|GetOutboundIP|ResolveServiceAddress" ats-platform/cmd ats-platform/internal/shared/consul
```

Expected outcome:
- Topology handling is explicit, minimal, and not duplicated across multiple code paths unnecessarily.

- [ ] **Step 6: Review topology code diff**

Run:

```bash
git diff -- ats-platform/internal/shared/consul/register.go ats-platform/cmd/gateway/discovery.go ats-platform/scripts/run-services.sh ats-platform/Makefile ats-platform/cmd/resume-service/main.go ats-platform/cmd/interview-service/main.go ats-platform/cmd/search-service/main.go
```

Expected outcome:
- The diff only documents and reinforces the approved local runtime behavior.

- [ ] **Step 7: Commit topology/runtime adjustments**

```bash
git add ats-platform/internal/shared/consul/register.go ats-platform/cmd/gateway/discovery.go ats-platform/scripts/run-services.sh ats-platform/Makefile ats-platform/cmd/resume-service/main.go ats-platform/cmd/interview-service/main.go ats-platform/cmd/search-service/main.go
git commit -m "chore: align local consul topology behavior"
```

### Task 3: Normalize ATS Docs Into Formal And Test Categories

**Files:**
- Create: `ats-platform/docs/README.md`
- Modify: `ats-platform/docs/SERVICE_DEVELOPMENT_GUIDE.md`
- Modify: `ats-platform/docs/consul-integration.md`
- Modify: `ats-platform/docs/resume-service-api.md`
- Create: `ats-platform/docs/tests/`
- Move: `ats-platform/docs/interview-service-api-test.md` -> `ats-platform/docs/tests/interview-service-api-test.md`
- Add tracked file: `ats-platform/docs/superpowers/plans/2026-03-23-search-service-implementation.md`

- [ ] **Step 1: Create docs index**

Create `ats-platform/docs/README.md` that explains:
- formal docs live at `ats-platform/docs/`
- test and experiment docs live at `ats-platform/docs/tests/`
- superpowers design/plan docs live at `ats-platform/docs/superpowers/`
- documents should be updated with implementation changes

Expected outcome:
- There is a single entry document explaining where future docs belong.

- [ ] **Step 2: Normalize the service development guide**

Review and update `ats-platform/docs/SERVICE_DEVELOPMENT_GUIDE.md` so it matches the current ATS implementation and the new docs structure.

Expected outcome:
- The guide reads as a formal maintained doc, not a loose scratch note.

- [ ] **Step 3: Update Consul integration doc to current reality**

Update `ats-platform/docs/consul-integration.md` to reflect:
- gateway discovery is implemented
- search service is integrated
- shared endpoint naming is in place
- local topology guidance includes `SERVICE_ADDRESS`
- any remaining deregistration limitation is described accurately if still true after implementation

Expected outcome:
- The doc stops describing already-completed work as pending.

- [ ] **Step 4: Update resume API doc to current implementation**

Review `ats-platform/docs/resume-service-api.md` against current handlers and service behavior, then correct stale examples or env defaults that are clearly outdated.

Expected outcome:
- The API doc is useful as a current formal reference.

- [ ] **Step 5: Move interview API test doc into tests category**

Create `ats-platform/docs/tests/` if missing and move `ats-platform/docs/interview-service-api-test.md` into it.

Expected outcome:
- Interface verification docs are separated from formal product docs.

- [ ] **Step 6: Track the existing search-service implementation plan**

Add `ats-platform/docs/superpowers/plans/2026-03-23-search-service-implementation.md` to git without rewriting it unless a tiny metadata correction is necessary.

Expected outcome:
- Valuable implementation-history docs are not left untracked.

- [ ] **Step 7: Review docs diff**

Run:

```bash
git diff -- ats-platform/docs
```

Expected outcome:
- Formal docs, test docs, and superpowers docs are clearly separated and internally consistent.

- [ ] **Step 8: Commit normalized docs**

```bash
git add ats-platform/docs
git commit -m "docs: normalize ats platform documentation"
```

### Task 4: Verify Local Consul Flow And Documentation State

**Files:**
- Verify: `ats-platform/cmd/resume-service/main.go`
- Verify: `ats-platform/cmd/interview-service/main.go`
- Verify: `ats-platform/cmd/search-service/main.go`
- Verify: `ats-platform/cmd/gateway/main.go`
- Verify: `ats-platform/docs/README.md`
- Verify: `ats-platform/docs/consul-integration.md`
- Verify: `ats-platform/docs/tests/interview-service-api-test.md`

- [ ] **Step 1: Build affected binaries**

Run:

```bash
cd ats-platform && go build ./cmd/resume-service ./cmd/interview-service ./cmd/search-service ./cmd/gateway
```

Expected outcome:
- The affected services and gateway still compile after the topology and docs changes.

- [ ] **Step 2: Start local stack using the documented topology convention**

Run the same local Docker infra plus host-run services using the documented `SERVICE_ADDRESS` path.

Expected outcome:
- Runtime behavior matches the docs instead of relying on hidden assumptions.

- [ ] **Step 3: Re-run Consul and gateway verification**

Run a curl suite covering at least:

```bash
curl -s http://127.0.0.1:8500/v1/catalog/services
curl -s http://127.0.0.1:18080/health
curl -i http://127.0.0.1:18080/api/v1/resumes/<id>
curl -i http://127.0.0.1:18080/api/v1/interviews/<id>
curl -s 'http://127.0.0.1:18080/api/v1/search?query=<term>&page=1&page_size=5'
```

Expected outcome:
- Registration, healthy discovery, and gateway proxying work under the documented topology.

- [ ] **Step 4: Verify deregistration behavior explicitly**

Stop one or more services and inspect:

```bash
curl -s 'http://127.0.0.1:8500/v1/health/service/resume-service-http?passing=false'
```

Expected outcome:
- Either deregistration works, or the exact remaining limitation is confirmed and documented.

- [ ] **Step 5: Verify docs inventory is no longer left floating**

Run:

```bash
git status --short
```

Expected outcome:
- The targeted ATS docs are tracked and no longer sit as unexplained untracked files.

- [ ] **Step 6: Review final diff**

Run:

```bash
git diff -- ats-platform/internal/shared/consul ats-platform/cmd/gateway ats-platform/scripts/run-services.sh ats-platform/Makefile ats-platform/docs
```

Expected outcome:
- Runtime behavior and docs structure line up with the approved spec.

- [ ] **Step 7: Commit final verification-aligned changes if needed**

```bash
git add ats-platform/internal/shared/consul ats-platform/cmd/gateway ats-platform/scripts/run-services.sh ats-platform/Makefile ats-platform/docs
git commit -m "docs: align consul topology guidance and docs structure"
```
