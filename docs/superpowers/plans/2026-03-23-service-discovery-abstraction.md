# Service Discovery Abstraction Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Centralize Consul naming, registration, and gateway HTTP discovery rules so services and gateway stop hardcoding `-http` / `-grpc` strings independently.

**Architecture:** Introduce a small shared service-discovery model in `internal/shared/consul/` that defines logical service names, protocol enums, and discover-name generation. Service binaries will register endpoints through that shared model, and gateway route targets will reference logical service names plus protocol instead of final Consul names.

**Tech Stack:** Go, Gin, Consul API

---

### Task 1: Add Shared Service Discovery Naming Model

**Files:**
- Modify: `ats-platform/internal/shared/consul/register.go`
- Create: `ats-platform/internal/shared/consul/naming.go`

- [ ] **Step 1: Define the logical service name constants**

Create `ats-platform/internal/shared/consul/naming.go` with constants for:

```go
const (
    ResumeServiceBaseName    = "resume-service"
    InterviewServiceBaseName = "interview-service"
    SearchServiceBaseName    = "search-service"
)
```

Expected outcome:
- A single shared source of truth for logical service names.

- [ ] **Step 2: Define protocol enum and discover-name helper**

Add to `ats-platform/internal/shared/consul/naming.go`:

```go
type Protocol string

const (
    ProtocolHTTP Protocol = "http"
    ProtocolGRPC Protocol = "grpc"
)

func ServiceName(baseName string, protocol Protocol) string {
    return fmt.Sprintf("%s-%s", baseName, protocol)
}
```

Expected outcome:
- Code no longer needs to manually concatenate `-http` / `-grpc`.

- [ ] **Step 3: Define endpoint description struct**

Add a small endpoint model:

```go
type Endpoint struct {
    BaseName string
    Protocol Protocol
    IP       string
    Port     int
}
```

Expected outcome:
- Registration and deregistration can share one endpoint description.

- [ ] **Step 4: Refactor service ID generation to use endpoint semantics**

Update `ats-platform/internal/shared/consul/register.go` to support:

```go
func EndpointServiceID(endpoint Endpoint, instanceID string) string
func RegisterEndpoint(endpoint Endpoint, instanceID string) error
```

Keep existing low-level logic minimal, but route it through `ServiceName(...)`.

Expected outcome:
- Consul naming and service ID generation become protocol-aware and centralized.

- [ ] **Step 5: Review shared helper diff**

Run:

```bash
git diff -- ats-platform/internal/shared/consul/register.go ats-platform/internal/shared/consul/naming.go
```

Expected outcome:
- Shared discovery model is small, clear, and only covers naming + registration.

- [ ] **Step 6: Commit shared service discovery helpers**

```bash
git add ats-platform/internal/shared/consul/register.go ats-platform/internal/shared/consul/naming.go
git commit -m "refactor: centralize consul service naming"
```

### Task 2: Refactor Service Registration To Use Shared Endpoints

**Files:**
- Modify: `ats-platform/cmd/resume-service/main.go`
- Modify: `ats-platform/cmd/interview-service/main.go`
- Modify: `ats-platform/cmd/search-service/main.go`

- [ ] **Step 1: Replace manual resume-service naming**

Update `ats-platform/cmd/resume-service/main.go` so it builds:

```go
httpEndpoint := consul.Endpoint{BaseName: consul.ResumeServiceBaseName, Protocol: consul.ProtocolHTTP, IP: ip, Port: httpPort}
grpcEndpoint := consul.Endpoint{BaseName: consul.ResumeServiceBaseName, Protocol: consul.ProtocolGRPC, IP: ip, Port: grpcPort}
```

Then register and deregister via shared helpers instead of building `httpServiceName` / `grpcServiceName` strings inline.

Expected outcome:
- `resume-service` no longer hardcodes discover-name suffixes.

- [ ] **Step 2: Replace manual interview-service naming**

Apply the same pattern in `ats-platform/cmd/interview-service/main.go`.

Expected outcome:
- `interview-service` uses the same shared endpoint model.

- [ ] **Step 3: Replace manual search-service naming**

Update `ats-platform/cmd/search-service/main.go` to build a single HTTP endpoint:

```go
httpEndpoint := consul.Endpoint{BaseName: consul.SearchServiceBaseName, Protocol: consul.ProtocolHTTP, IP: ip, Port: httpPort}
```

Expected outcome:
- `search-service` participates in the same naming abstraction while still registering only HTTP.

- [ ] **Step 4: Review service registration diff**

Run:

```bash
git diff -- ats-platform/cmd/resume-service/main.go ats-platform/cmd/interview-service/main.go ats-platform/cmd/search-service/main.go
```

Expected outcome:
- Service binaries now declare endpoints, not discover-name strings.

- [ ] **Step 5: Commit service registration refactor**

```bash
git add ats-platform/cmd/resume-service/main.go ats-platform/cmd/interview-service/main.go ats-platform/cmd/search-service/main.go
git commit -m "refactor: use shared consul endpoints in services"
```

### Task 3: Refactor Gateway Discovery To Use Logical Service Names

**Files:**
- Modify: `ats-platform/cmd/gateway/main.go`
- Modify: `ats-platform/cmd/gateway/discovery.go`

- [ ] **Step 1: Change gateway route targets to logical service names**

Update route target shape in `ats-platform/cmd/gateway/main.go` to store:

```go
type routeTarget struct {
    ServiceKey      string
    BaseServiceName string
    Protocol        consul.Protocol
}
```

Expected outcome:
- Route table expresses business intent, not final Consul names.

- [ ] **Step 2: Resolve gateway discover names through shared helper**

Update `ats-platform/cmd/gateway/discovery.go` and call sites to use:

```go
consul.ServiceName(target.BaseServiceName, target.Protocol)
```

Expected outcome:
- Gateway no longer hardcodes `resume-service-http`, `interview-service-http`, or `search-service-http`.

- [ ] **Step 3: Update health page and home page service lookups**

Change gateway home and health handlers to request HTTP discover names through the shared helper rather than inline literals.

Expected outcome:
- Gateway discovery is fully driven by logical service name + protocol.

- [ ] **Step 4: Review gateway diff**

Run:

```bash
git diff -- ats-platform/cmd/gateway/main.go ats-platform/cmd/gateway/discovery.go
```

Expected outcome:
- No final Consul service names remain hardcoded in gateway runtime logic.

- [ ] **Step 5: Commit gateway abstraction update**

```bash
git add ats-platform/cmd/gateway/main.go ats-platform/cmd/gateway/discovery.go
git commit -m "refactor: derive gateway discovery names from shared model"
```

### Task 4: Verify Discovery Naming End-To-End

**Files:**
- Modify: `docs/superpowers/specs/2026-03-20-ats-platform-design.md`
- Modify: `docs/SERVICE_DEVELOPMENT_GUIDE.md`
- Modify: `ats-platform/docs/superpowers/specs/2026-03-23-search-service-design.md`

- [ ] **Step 1: Build all affected binaries**

Run:

```bash
cd ats-platform && go build ./cmd/resume-service ./cmd/interview-service ./cmd/search-service ./cmd/gateway
```

Expected outcome:
- All binaries compile after the abstraction refactor.

- [ ] **Step 2: Run the full local regression curl suite**

Run the same local startup and verification sequence already proven in this branch:

```bash
curl -s http://127.0.0.1:8500/v1/catalog/services
curl -i http://127.0.0.1:18080/health
curl -s -X POST http://127.0.0.1:18080/api/v1/resumes ...
curl -i http://127.0.0.1:18080/api/v1/resumes/<id>
curl -s -X POST http://127.0.0.1:18082/api/v1/interviews ...
curl -i http://127.0.0.1:18080/api/v1/interviews/<id>
curl -i 'http://127.0.0.1:18080/api/v1/search?query=<term>&page=1&page_size=5'
```

Expected outcome:
- Gateway and Consul naming still work end-to-end after abstraction.

- [ ] **Step 3: Update design and development docs**

Adjust:

- `docs/superpowers/specs/2026-03-20-ats-platform-design.md`
- `docs/SERVICE_DEVELOPMENT_GUIDE.md`
- `ats-platform/docs/superpowers/specs/2026-03-23-search-service-design.md`

Describe:
- logical service names
- `ServiceName(base, protocol)` naming convention
- `search-service` registering only HTTP
- gateway depending on logical names plus HTTP protocol

Expected outcome:
- Docs match the new abstraction rather than the old hardcoded strings.

- [ ] **Step 4: Review final abstraction diff**

Run:

```bash
git diff -- ats-platform/internal/shared/consul ats-platform/cmd/resume-service/main.go ats-platform/cmd/interview-service/main.go ats-platform/cmd/search-service/main.go ats-platform/cmd/gateway docs/SERVICE_DEVELOPMENT_GUIDE.md docs/superpowers/specs/2026-03-20-ats-platform-design.md ats-platform/docs/superpowers/specs/2026-03-23-search-service-design.md
```

Expected outcome:
- All registration and discovery naming now flow through one shared model.

- [ ] **Step 5: Commit verification and doc sync**

```bash
git add docs/SERVICE_DEVELOPMENT_GUIDE.md docs/superpowers/specs/2026-03-20-ats-platform-design.md ats-platform/docs/superpowers/specs/2026-03-23-search-service-design.md
git commit -m "docs: align discovery abstraction documentation"
```
