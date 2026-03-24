# Gateway Orchestrator UI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rebuild the gateway homepage into an orchestration console that can run and observe the main ATS flow through real gateway routes.

**Architecture:** Move the gateway HTML out of `main.go` into an embedded static file, then implement a lightweight client-side state machine that drives a five-step demo flow and per-step replay. Keep all runtime calls on top of the existing gateway `/api/v1/*` proxy surface plus `/health`, and avoid adding any orchestration backend API.

**Tech Stack:** Go, `go:embed`, static HTML, CSS, vanilla JavaScript, Gin

---

### Task 1: Move Gateway HTML Into A Dedicated Static File

**Files:**
- Create: `ats-platform/cmd/gateway/static/index.html`
- Modify: `ats-platform/cmd/gateway/main.go`

- [ ] **Step 1: Read the current gateway homepage implementation**

Read:
- `ats-platform/cmd/gateway/main.go`

Expected outcome:
- Identify the current inline `indexHTMLTemplate`, `r.GET("/")`, and any dynamic values injected through `fmt.Sprintf`.

- [ ] **Step 2: Create the static asset directory**

Create:
- `ats-platform/cmd/gateway/static/`

Expected outcome:
- Gateway has the same static asset structure as the other service consoles.

- [ ] **Step 3: Copy the current homepage into `static/index.html` as the starting point**

Create:
- `ats-platform/cmd/gateway/static/index.html`

Expected outcome:
- The old inline HTML becomes an editable standalone file before any UI redesign begins.

- [ ] **Step 4: Replace the inline HTML constant with `go:embed`**

Modify:
- `ats-platform/cmd/gateway/main.go`

Implementation notes:
- Add `//go:embed static/index.html`
- Remove the large inline HTML string
- Serve the embedded file directly in `r.GET("/")`

Expected outcome:
- `main.go` only embeds and serves the static asset, instead of owning the page markup inline.

- [ ] **Step 5: Build gateway to verify embed wiring**

Run:
```bash
cd ats-platform && go build ./cmd/gateway
```

Expected outcome:
- Gateway still compiles after the static file extraction.

- [ ] **Step 6: Commit the static asset extraction**

```bash
git add ats-platform/cmd/gateway/main.go ats-platform/cmd/gateway/static/index.html
git commit -m "refactor: extract gateway homepage into static asset"
```

### Task 2: Build The Orchestrator Layout And Shared UI Primitives

**Files:**
- Modify: `ats-platform/cmd/gateway/static/index.html`

- [ ] **Step 1: Re-read the three existing service consoles for shared design cues**

Read:
- `ats-platform/cmd/resume-service/static/index.html`
- `ats-platform/cmd/interview-service/static/index.html`
- `ats-platform/cmd/search-service/static/index.html`

Expected outcome:
- Confirm the common design language to reuse: serif typography, gradient background, hero/panel/card structure, pill badges, inspector blocks, toast feedback.

- [ ] **Step 2: Define the final page sections in the HTML shell**

Modify:
- `ats-platform/cmd/gateway/static/index.html`

Include sections for:
- Hero
- Flow Overview
- Step Cards
- Runtime Context
- Request / Response Inspector
- Service Health

Expected outcome:
- The page structure reflects the orchestrator role before adding behavior.

- [ ] **Step 3: Add unified visual styling**

Modify:
- `ats-platform/cmd/gateway/static/index.html`

Implementation notes:
- Reuse the existing service console visual language
- Keep gateway visually distinct via orchestration-focused layout and status rail
- Ensure mobile-safe layout with responsive breakpoints

Expected outcome:
- Gateway looks like part of the same product family, but clearly reads as a flow console.

- [ ] **Step 4: Add reusable UI placeholders and result containers**

Modify:
- `ats-platform/cmd/gateway/static/index.html`

Include:
- Step status chips
- Summary placeholders
- Context placeholders
- Inspector text areas / code blocks
- Toast container

Expected outcome:
- All dynamic content has stable DOM targets before JS behavior is added.

- [ ] **Step 5: Review the visual diff**

Run:
```bash
git diff -- ats-platform/cmd/gateway/static/index.html
```

Expected outcome:
- The markup and CSS changes remain focused on the orchestrator layout rather than introducing unrelated UI abstractions.

- [ ] **Step 6: Commit the layout pass**

```bash
git add ats-platform/cmd/gateway/static/index.html
git commit -m "feat: add gateway orchestrator console layout"
```

### Task 3: Implement Client-Side State And Per-Step Execution

**Files:**
- Modify: `ats-platform/cmd/gateway/static/index.html`

- [ ] **Step 1: Add a lightweight page state object**

Modify:
- `ats-platform/cmd/gateway/static/index.html`

The state should track:
- step statuses
- `resumeId`
- `interviewId`
- `searchQuery`
- latest request
- latest response
- last error
- current service health snapshot

Expected outcome:
- Gateway UI has one canonical runtime state object for rendering and orchestration.

- [ ] **Step 2: Add common helpers for rendering and feedback**

Modify:
- `ats-platform/cmd/gateway/static/index.html`

Implement small helpers for:
- showing toast messages
- updating step status
- updating runtime context
- writing request / response inspector output
- writing summary text into cards

Expected outcome:
- Later step handlers can stay short and avoid duplicated DOM manipulation.

- [ ] **Step 3: Add an API wrapper that always targets gateway routes**

Modify:
- `ats-platform/cmd/gateway/static/index.html`

Implementation notes:
- Centralize `fetch` usage
- Support JSON request bodies
- Capture method, URL, payload, status, and response body for the inspector
- Use gateway `/api/v1/*` routes for business calls
- Use `/health` only for gateway health snapshots

Expected outcome:
- All user-visible debugging reflects real gateway traffic instead of direct service calls.

- [ ] **Step 4: Implement Step 1 - Create Resume**

Modify:
- `ats-platform/cmd/gateway/static/index.html`

Implementation notes:
- Build a minimal resume payload from page inputs or sensible demo defaults
- Send `POST /api/v1/resumes`
- Persist `resumeId` into page state

Expected outcome:
- The first orchestration step can create a real resume through gateway.

- [ ] **Step 5: Implement Step 2 - Update Status / Attempt Parse**

Modify:
- `ats-platform/cmd/gateway/static/index.html`

Implementation notes:
- Guard on existing `resumeId`
- First update resume status through gateway
- Optionally attempt parse when conditions allow
- Record `success`, `degraded`, or `error` explicitly

Expected outcome:
- Step 2 no longer blocks the whole flow on parsing volatility.

- [ ] **Step 6: Implement Step 3 - Create Interview**

Modify:
- `ats-platform/cmd/gateway/static/index.html`

Implementation notes:
- Guard on existing `resumeId`
- Send `POST /api/v1/interviews`
- Persist `interviewId`

Expected outcome:
- The page can attach an interview to the created resume through gateway.

- [ ] **Step 7: Implement Step 4 - Search Verify**

Modify:
- `ats-platform/cmd/gateway/static/index.html`

Implementation notes:
- Use a predictable query derived from the created resume context
- Send gateway search request
- Show whether the newly created resume appears in search results

Expected outcome:
- The page can verify index visibility as part of the demo flow.

- [ ] **Step 8: Implement Step 5 - Gateway Summary**

Modify:
- `ats-platform/cmd/gateway/static/index.html`

Implementation notes:
- Refresh `/health`
- Render discovered upstream addresses
- Summarize the created resume, interview, and search visibility outcome

Expected outcome:
- The final step shows gateway’s own view of the ecosystem plus the flow summary.

- [ ] **Step 9: Add per-step button wiring**

Modify:
- `ats-platform/cmd/gateway/static/index.html`

Expected outcome:
- Every step card can run independently for replay and debugging.

- [ ] **Step 10: Commit the per-step behavior**

```bash
git add ats-platform/cmd/gateway/static/index.html
git commit -m "feat: add gateway step-by-step orchestration flow"
```

### Task 4: Implement Full Demo Orchestration And Guardrails

**Files:**
- Modify: `ats-platform/cmd/gateway/static/index.html`

- [ ] **Step 1: Implement `Run Full Demo` orchestration**

Modify:
- `ats-platform/cmd/gateway/static/index.html`

Implementation notes:
- Execute the five steps in order
- Stop on hard failure
- Allow Step 2 to continue on `degraded`

Expected outcome:
- One button can run the main ATS demo chain end-to-end.

- [ ] **Step 2: Add dependency guardrails**

Modify:
- `ats-platform/cmd/gateway/static/index.html`

Required guards:
- No Step 3 without `resumeId`
- No Step 5 without at least one previous successful step
- Search step must have a usable query

Expected outcome:
- Users get deterministic, actionable feedback instead of silent failures.

- [ ] **Step 3: Add reset behavior**

Modify:
- `ats-platform/cmd/gateway/static/index.html`

Implementation notes:
- Reset step statuses
- Clear IDs and inspector contents
- Preserve configurable defaults for demo inputs

Expected outcome:
- A user can restart the flow without reloading the page.

- [ ] **Step 4: Add health refresh behavior**

Modify:
- `ats-platform/cmd/gateway/static/index.html`

Expected outcome:
- The service health panel can be refreshed independently of the main demo flow.

- [ ] **Step 5: Review orchestration diff**

Run:
```bash
git diff -- ats-platform/cmd/gateway/static/index.html
```

Expected outcome:
- The control flow remains readable and does not drift into a generic workflow engine.

- [ ] **Step 6: Commit the orchestration controls**

```bash
git add ats-platform/cmd/gateway/static/index.html
git commit -m "feat: add gateway full-demo orchestration controls"
```

### Task 5: Verify The Gateway Orchestrator And Update Docs

**Files:**
- Modify: `ats-platform/docs/SERVICE_DEVELOPMENT_GUIDE.md`
- Modify: `docs/superpowers/specs/2026-03-20-ats-platform-design.md`

- [ ] **Step 1: Build the affected gateway binary**

Run:
```bash
cd ats-platform && go build ./cmd/gateway
```

Expected outcome:
- Gateway still compiles after the UI refactor.

- [ ] **Step 2: Start the local stack for manual verification**

Run:
```bash
cd ats-platform && ./scripts/run-services.sh --no-infra --gateway
```

Expected outcome:
- Gateway and supporting services are available for browser and curl validation.

- [ ] **Step 3: Verify gateway health by curl**

Run:
```bash
curl -s http://127.0.0.1:8080/health
```

Expected outcome:
- Gateway health aggregation still reports the discovered services.

- [ ] **Step 4: Verify the UI page loads**

Run:
```bash
curl -s http://127.0.0.1:8080/ | head -n 40
```

Expected outcome:
- The new orchestrator page is served instead of the previous minimal homepage.

- [ ] **Step 5: Manually run the main flow in the browser**

Verify:
- `Run Full Demo` executes all five steps
- Step 2 can surface `degraded` without killing the flow
- IDs, summaries, and inspector output all update
- health panel shows current upstream addresses

Expected outcome:
- The gateway page can debug the main ATS flow from a single screen.

- [ ] **Step 6: Update formal docs**

Modify:
- `ats-platform/docs/SERVICE_DEVELOPMENT_GUIDE.md`
- `docs/superpowers/specs/2026-03-20-ats-platform-design.md`

Document:
- gateway now provides an orchestration-style frontend
- the page uses real gateway routes for debugging
- the role remains “gateway UI for flow verification”, not a new backend subsystem

Expected outcome:
- Docs describe the current gateway front-end accurately.

- [ ] **Step 7: Review the final diff**

Run:
```bash
git diff -- ats-platform/cmd/gateway/main.go ats-platform/cmd/gateway/static/index.html ats-platform/docs/SERVICE_DEVELOPMENT_GUIDE.md docs/superpowers/specs/2026-03-20-ats-platform-design.md
```

Expected outcome:
- The final changeset is limited to the gateway UI refactor and related docs.

- [ ] **Step 8: Commit the verified gateway orchestrator UI**

```bash
git add ats-platform/cmd/gateway/main.go ats-platform/cmd/gateway/static/index.html ats-platform/docs/SERVICE_DEVELOPMENT_GUIDE.md docs/superpowers/specs/2026-03-20-ats-platform-design.md
git commit -m "feat: add gateway orchestrator ui"
```
