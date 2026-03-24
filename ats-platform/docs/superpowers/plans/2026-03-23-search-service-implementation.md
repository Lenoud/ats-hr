# Search-Service Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans (batch execution with checkpoints)

**Goal:** Implement a complete search-service module following SERVICE_DEVELOPMENT_GUIDE.md standards, with Elasticsearch integration, Redis event consumption, and Consul service registration.

**Architecture:**
- Standard service template (following resume-service pattern)
- Layered design: Handler → Service → Repository
- Elasticsearch for search indexing
- Redis Stream for event consumption
- Consul for service discovery

**Tech Stack:**
- Go 1.21+
- Gin (HTTP)
- go-elasticsearch/v8 (Elasticsearch client)
- go-redis/v9 (Redis client)
- hashicorp/consul/api (Consul client)
- Zap (Logging)

---

## Tasks

### Phase 1: ES Repository Implementation
- [ ] 1.1: Add esRepositoryImpl struct and constructor to es_repository.go
- [ ] 1.2: Implement Index method - index document to Elasticsearch
- [ ] 1.3: Implement GetByID method - retrieve document by ID
- [ ] 1.4: Implement Delete method - delete document from index
- [ ] 1.5: Implement Search method - build and execute ES query
- [ ] 1.6: Implement UpdateStatus method - update document status
- [ ] 1.7: Add EnsureIndex helper - create index with mappings if not exists

### Phase 2: Service Entry Point (main.go)
- [ ] 2.1: Define Config struct with all required configuration
- [ ] 2.2: Implement loadConfig function with environment variables
- [ ] 2.3: Implement getEnv helper function
- [ ] 2.4: Initialize logger in main function
- [ ] 2.5: Initialize Elasticsearch client and ensure index
- [ ] 2.6: Initialize Redis client with connection check
- [ ] 2.7: Initialize Consul client and register service
- [ ] 2.8: Initialize Repository, Service, and Handler
- [ ] 2.9: Start Event Consumer goroutine
- [ ] 2.10: Configure HTTP routes (health, ready, search endpoints)
- [ ] 2.11: Start HTTP Server goroutine
- [ ] 2.12: Implement graceful shutdown (SIGINT, SIGTERM handling)
- [ ] 2.13: Cleanup: Close HTTP, Consul deregister, Redis close

### Phase 3: Static Assets
- [ ] 3.1: Create static/index.html with service info page

### Phase 4: Event Handler Integration
- [ ] 4.1: Create handleSearchEvent function in main.go
- [ ] 4.2: Handle "created" action - parse payload and index document
- [ ] 4.3: Handle "deleted" action - delete document from index
- [ ] 4.4: Handle "status_changed" action - update document status
- [ ] 4.5: Add error handling and logging for event processing

---

## Commit Strategy
- [ ] Commit after Phase 1: ES Repository implementation
- [ ] Commit after Phase 2: main.go implementation
- [ ] Commit after Phase 3: Static assets
- [ ] Commit after Phase 4: Event handler integration

---

## Dependencies
- github.com/gin-gonic/gin
- github.com/redis/go-redis/v9
- github.com/elastic/go-elasticsearch/v8
- github.com/hashicorp/consul/api
- go.uber.org/zap
- github.com/google/uuid
