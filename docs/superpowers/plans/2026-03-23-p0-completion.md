# P0 Completion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Complete P0 by aligning `interview-service` gRPC behavior with real capabilities, formalizing the resume-to-search event contract, and upgrading `gateway` to resolve services dynamically through Consul.

**Architecture:** The work stays split into three bounded change sets. `interview-service` is corrected by narrowing gRPC exposure to true service-layer support, shared event contracts move into `internal/shared/events` and are consumed by both publisher and search consumer, and `gateway` keeps path-based routing while replacing static service URLs with Consul-backed resolution.

**Tech Stack:** Go, Gin, gRPC, Protobuf, Redis Streams, Elasticsearch, Consul

---

### Task 1: Align Interview gRPC To Real Capabilities

**Files:**
- Modify: `ats-platform/proto/interview.proto`
- Modify: `ats-platform/internal/shared/pb/interview/interview.pb.go`
- Modify: `ats-platform/internal/shared/pb/interview/interview_grpc.pb.go`
- Modify: `ats-platform/internal/interview/grpc/server.go`
- Modify: `ats-platform/docs/interview-service-grpc.md`
- Modify: `docs/superpowers/specs/2026-03-20-ats-platform-design.md`

- [ ] **Step 1: Map the exact mismatch set**

Read:
- `ats-platform/proto/interview.proto`
- `ats-platform/internal/interview/grpc/server.go`
- `ats-platform/internal/interview/service/`
- `ats-platform/docs/interview-service-grpc.md`

Write down which methods are:
- fully supported
- partially supported
- explicit stubs / `Unimplemented`

Expected outcome:
- A short mismatch list with concrete method names such as `UpdateInterview`, `GetFeedback`, and any portfolio method with incomplete backing service behavior.

- [ ] **Step 2: Decide the final exposed gRPC surface**

Apply this rule:
- keep methods that have real, stable service-layer backing
- remove methods that are only placeholders
- avoid inventing new service-layer behavior just to preserve an existing proto method

Expected outcome:
- Final keep/remove decision for `InterviewService`, `FeedbackService`, and `PortfolioService`.

- [ ] **Step 3: Update `proto/interview.proto` to the chosen surface**

Edit:
- `ats-platform/proto/interview.proto`

Remove or retain RPCs based on Step 2 so the proto only advertises supported behavior.

Expected outcome:
- The proto contract matches intended real capability.

- [ ] **Step 4: Regenerate protobuf code**

Run:
```bash
cd ats-platform && make proto
```

Expected outcome:
- `internal/shared/pb/interview/interview.pb.go`
- `internal/shared/pb/interview/interview_grpc.pb.go`

are regenerated to match the updated proto.

- [ ] **Step 5: Update gRPC server implementation**

Edit:
- `ats-platform/internal/interview/grpc/server.go`

Remove handlers for deleted RPCs or adjust the server implementation so every remaining RPC has clear semantics and no placeholder behavior.

Expected outcome:
- No remaining exported gRPC method is “fake-implemented”.

- [ ] **Step 6: Update gRPC documentation**

Edit:
- `ats-platform/docs/interview-service-grpc.md`

Reflect the final method list, remove stale examples, and explicitly describe current supported behavior only.

Expected outcome:
- The gRPC doc, proto, and server implementation all match.

- [ ] **Step 7: Update platform design notes if service capability changed**

Edit:
- `docs/superpowers/specs/2026-03-20-ats-platform-design.md`

Only adjust wording if the published gRPC capability statement needs to reflect the narrowed scope.

Expected outcome:
- No top-level architecture doc overstates interview gRPC capability.

- [ ] **Step 8: Verify generated and source files are in sync**

Run:
```bash
git diff -- ats-platform/proto/interview.proto ats-platform/internal/interview/grpc/server.go ats-platform/internal/shared/pb/interview/interview.pb.go ats-platform/internal/shared/pb/interview/interview_grpc.pb.go ats-platform/docs/interview-service-grpc.md docs/superpowers/specs/2026-03-20-ats-platform-design.md
```

Expected outcome:
- Only intended alignment changes appear.

- [ ] **Step 9: Commit the interview gRPC alignment**

```bash
git add ats-platform/proto/interview.proto ats-platform/internal/shared/pb/interview/interview.pb.go ats-platform/internal/shared/pb/interview/interview_grpc.pb.go ats-platform/internal/interview/grpc/server.go ats-platform/docs/interview-service-grpc.md docs/superpowers/specs/2026-03-20-ats-platform-design.md
git commit -m "refactor: align interview grpc surface with implementation"
```

### Task 2: Formalize The Resume Search Event Contract

**Files:**
- Create: `ats-platform/internal/shared/events/contracts.go`
- Modify: `ats-platform/internal/shared/events/publisher.go`
- Modify: `ats-platform/internal/shared/events/consumer.go`
- Modify: `ats-platform/cmd/search-service/main.go`
- Modify: `ats-platform/internal/search/service/search_service.go`
- Modify: `docs/superpowers/specs/2026-03-20-ats-platform-design.md`
- Modify: `docs/SERVICE_DEVELOPMENT_GUIDE.md`

- [ ] **Step 1: Inventory the current published and consumed event shapes**

Read:
- `ats-platform/internal/shared/events/publisher.go`
- `ats-platform/internal/shared/events/consumer.go`
- `ats-platform/cmd/search-service/main.go`
- `ats-platform/internal/resume/service/`

Document:
- action names in use
- payload shapes currently produced
- payload shapes currently assumed by search consumer

Expected outcome:
- A precise list of event contract mismatches and implicit assumptions.

- [ ] **Step 2: Define a shared event contract file**

Create:
- `ats-platform/internal/shared/events/contracts.go`

Include:
- action constants
- payload structs for supported actions
- comments describing which actions require payloads

Expected outcome:
- One canonical place for resume event names and payload schemas.

- [ ] **Step 3: Refactor publisher helpers to use shared contract types**

Edit:
- `ats-platform/internal/shared/events/publisher.go`

Replace free-form action strings and ad hoc payload maps where possible with the shared constants and payload structs from `contracts.go`.

Expected outcome:
- Publisher-side event generation is contract-driven instead of stringly-typed.

- [ ] **Step 4: Refactor consumer parsing to use shared contract types**

Edit:
- `ats-platform/internal/shared/events/consumer.go`
- `ats-platform/cmd/search-service/main.go`

Update consumer-side decoding so it relies on the same shared constants and payload types.

Expected outcome:
- Search event handling no longer redefines its own payload understanding.

- [ ] **Step 5: Keep search indexing semantics explicit**

Edit:
- `ats-platform/cmd/search-service/main.go`
- `ats-platform/internal/search/service/search_service.go`

Make the mapping from each event action to search behavior explicit:
- `created` / `updated` / `parsed` => index or reindex
- `deleted` => delete
- `status_changed` => status update

Expected outcome:
- The indexing behavior is readable and tied directly to the contract.

- [ ] **Step 6: Update architecture and development docs**

Edit:
- `docs/superpowers/specs/2026-03-20-ats-platform-design.md`
- `docs/SERVICE_DEVELOPMENT_GUIDE.md`

Describe the shared event contract and clarify that publisher and consumer use the same event definitions.

Expected outcome:
- The docs describe a real shared contract, not an informal event flow.

- [ ] **Step 7: Verify the contract is consistently referenced**

Run:
```bash
rg -n '"created"|"updated"|"deleted"|"status_changed"|"parsed"' ats-platform/internal ats-platform/cmd
```

Expected outcome:
- Remaining raw strings are either gone or clearly limited to the shared contract definition.

- [ ] **Step 8: Review the diff for contract drift**

Run:
```bash
git diff -- ats-platform/internal/shared/events ats-platform/cmd/search-service/main.go ats-platform/internal/search/service/search_service.go docs/superpowers/specs/2026-03-20-ats-platform-design.md docs/SERVICE_DEVELOPMENT_GUIDE.md
```

Expected outcome:
- Event changes are localized and the contract is centralized.

- [ ] **Step 9: Commit the event contract work**

```bash
git add ats-platform/internal/shared/events/contracts.go ats-platform/internal/shared/events/publisher.go ats-platform/internal/shared/events/consumer.go ats-platform/cmd/search-service/main.go ats-platform/internal/search/service/search_service.go docs/superpowers/specs/2026-03-20-ats-platform-design.md docs/SERVICE_DEVELOPMENT_GUIDE.md
git commit -m "refactor: formalize resume search event contracts"
```

### Task 3: Upgrade Gateway To Dynamic Discovery

**Files:**
- Create: `ats-platform/cmd/gateway/discovery.go`
- Modify: `ats-platform/cmd/gateway/main.go`
- Modify: `docs/superpowers/specs/2026-03-20-ats-platform-design.md`
- Modify: `docs/SERVICE_DEVELOPMENT_GUIDE.md`

- [ ] **Step 1: Inspect existing Consul helpers and gateway routing flow**

Read:
- `ats-platform/cmd/gateway/main.go`
- `ats-platform/internal/shared/consul/`

Document:
- what helper functions already exist
- what gateway currently needs to translate a route prefix into a concrete upstream address

Expected outcome:
- A simple integration point for Consul-based lookup is identified.

- [ ] **Step 2: Add a focused service discovery helper for gateway**

Create:
- `ats-platform/cmd/gateway/discovery.go`

Implement a small helper that:
- resolves a logical service name through Consul
- returns one usable upstream base URL
- produces a clear error when no healthy instance exists

Expected outcome:
- Consul lookup logic is isolated from HTTP proxy flow.

- [ ] **Step 3: Replace the static service URL map with logical service routing**

Edit:
- `ats-platform/cmd/gateway/main.go`

Keep:
- path-prefix to service-name mapping

Remove:
- path-prefix to hardcoded URL mapping

Expected outcome:
- Gateway decides *which service* to route to based on path, but resolves *where* through discovery.

- [ ] **Step 4: Integrate discovery into proxy request handling**

Edit:
- `ats-platform/cmd/gateway/main.go`

Update proxy flow so each request:
- identifies target service name
- resolves an upstream instance via Consul
- forwards the request to that instance

Expected outcome:
- Gateway behavior remains the same from the client perspective, but upstream resolution becomes dynamic.

- [ ] **Step 5: Improve gateway error responses**

Edit:
- `ats-platform/cmd/gateway/main.go`

Return clear gateway errors for:
- unknown service prefixes
- no discovered instances
- upstream request failure

Expected outcome:
- Gateway failures are diagnosable rather than generic.

- [ ] **Step 6: Update architecture documentation**

Edit:
- `docs/superpowers/specs/2026-03-20-ats-platform-design.md`
- `docs/SERVICE_DEVELOPMENT_GUIDE.md`

Clarify that gateway is now a light dynamic-discovery gateway backed by Consul, while still remaining path-based and intentionally minimal.

Expected outcome:
- Docs no longer describe gateway as static-only.

- [ ] **Step 7: Verify no static upstream map remains**

Run:
```bash
rg -n "localhost:8081|localhost:8082|localhost:8083|var services = map" ats-platform/cmd/gateway
```

Expected outcome:
- Static upstream URLs are removed from gateway runtime logic.

- [ ] **Step 8: Review the gateway diff**

Run:
```bash
git diff -- ats-platform/cmd/gateway/main.go ats-platform/cmd/gateway/discovery.go docs/superpowers/specs/2026-03-20-ats-platform-design.md docs/SERVICE_DEVELOPMENT_GUIDE.md
```

Expected outcome:
- Gateway logic is still focused and limited to discovery plus proxying.

- [ ] **Step 9: Commit the gateway dynamic discovery change**

```bash
git add ats-platform/cmd/gateway/main.go ats-platform/cmd/gateway/discovery.go docs/superpowers/specs/2026-03-20-ats-platform-design.md docs/SERVICE_DEVELOPMENT_GUIDE.md
git commit -m "feat: add consul-backed gateway discovery"
```

### Task 4: Final P0 Documentation And Completion Review

**Files:**
- Modify: `ats-platform/docs/interview-service-grpc.md`
- Modify: `ats-platform/docs/superpowers/specs/2026-03-23-search-service-design.md`
- Modify: `docs/superpowers/specs/2026-03-20-ats-platform-design.md`
- Modify: `docs/SERVICE_DEVELOPMENT_GUIDE.md`

- [ ] **Step 1: Re-read all updated docs after code changes**

Read:
- `ats-platform/docs/interview-service-grpc.md`
- `ats-platform/docs/superpowers/specs/2026-03-23-search-service-design.md`
- `docs/superpowers/specs/2026-03-20-ats-platform-design.md`
- `docs/SERVICE_DEVELOPMENT_GUIDE.md`

Expected outcome:
- Any stale wording introduced by the P0 implementation is visible before finalizing.

- [ ] **Step 2: Align search-service design doc with final event contract**

Edit:
- `ats-platform/docs/superpowers/specs/2026-03-23-search-service-design.md`

Ensure it reflects the new shared event contract and final event semantics.

Expected outcome:
- Search design doc matches the actual contract used in code.

- [ ] **Step 3: Do a final targeted repository diff review**

Run:
```bash
git diff --stat
git diff -- ats-platform/proto/interview.proto ats-platform/internal/interview/grpc/server.go ats-platform/internal/shared/events ats-platform/cmd/search-service/main.go ats-platform/internal/search/service/search_service.go ats-platform/cmd/gateway docs/SERVICE_DEVELOPMENT_GUIDE.md docs/superpowers/specs/2026-03-20-ats-platform-design.md ats-platform/docs/interview-service-grpc.md ats-platform/docs/superpowers/specs/2026-03-23-search-service-design.md
```

Expected outcome:
- The whole P0 scope remains limited to the planned files and behaviors.

- [ ] **Step 4: Commit the final documentation sync**

```bash
git add ats-platform/docs/interview-service-grpc.md ats-platform/docs/superpowers/specs/2026-03-23-search-service-design.md docs/superpowers/specs/2026-03-20-ats-platform-design.md docs/SERVICE_DEVELOPMENT_GUIDE.md
git commit -m "docs: sync p0 platform capabilities"
```
