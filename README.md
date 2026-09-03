MyGoCloud Control Plane

A control plane for deploying containerized applications via CLI. The CLI communicates exclusively through a Go HTTP API, which manages/stores state in PostgreSQL and orchestrates Docker containers — no direct CLI-to-Docker coupling. The idea is later after more phases to look like a mini cloud ( for now it's a tiny project).



---

## 🎯 Why This Project?

After graduating my Bachelor's Degree in CS i decided i wanted a creative break from the programming industry.
Spending few months without coding made me realise, that the break was indeed from the university lifestile (not the coding/and programming and creating things)
I decided i wanted to make a comeback with another language - Golang. 

Indeed that is one of my first bigger projects, where i decided i don't want to make some small/module app, but doing a larger scale so i can make a proejct which is getting bulkier and watching it evolve as a system ( btw, i love system design thinking - it's a larger scale)


This demonstrates how to build a **real-world platform CLI** similar to Heroku, Fly.io, or Vercel:

- **Clean Architecture**: Domain-driven design with repository/service/handler layers
- **Dual Storage**: In-memory repos for fast testing, PostgreSQL for production
- **Async Deployments**: Goroutine-based deployment polling with panic recovery
- **Rollback Safety**: Application-level locking prevents concurrent deploys/rollbacks

---

## 🏗️ Architecture Overview

┌─────────────┐      HTTP/JSON      ┌─────────────────────┐      SQL     ┌─────────────┐
│             │ ◄──────────────────►│                     │ ◄──────────► │             │
│  mygocloud  │   /api/v1/...       │  Control Plane API  │              │  PostgreSQL │
│    (CLI)    │                     │   (Go + Chi Router) │              │             │
│             │                     │                     │              │             │
└─────────────┘                     └──────────┬──────────┘              └─────────────┘
│
│ exec
▼
┌─────────────────┐
│  Docker Engine  │
│  pull/run/logs  │
└─────────────────┘

The project is build on phases, for the first pushing to git we will include the first 3 phases and we will showcase what has been achieved as a result at each phase:

## 🚧 Build Phases

### Phase 1 — Control Plane API
> **Goal**: Core HTTP API with PostgreSQL persistence

- Go HTTP server (`net/http` + `chi/v5`)
- RESTful resources: `Users`, `Applications`, `Deployments`
- PostgreSQL schema with migrations (users → apps → deployments)
- `POST/GET` endpoints with structured JSON error responses
- Domain layer with repository interfaces (swappable in-memory/Postgres)

┌──────────┐     POST /users        ┌──────────┐     INSERT     ┌──────────┐
│  Client  │ ─────────────────────► │   API    │ ─────────────► │ Postgres │
│ (curl)   │     GET  /applications │  Server  │     SELECT     │          │
└──────────┘                        └──────────┘                └──────────┘


### Phase 2 — MyGoCloud CLI
> **Goal**: CLI talks **only** to the API, never to Docker/K8s directly

- Cobra-based CLI: `mygocloud login | user | app | viz`
- Config persisted to `~/.mygocloud/config.yaml`
- Client package wraps all API calls with timeout & error handling
- Printer package supports `table/json/yaml` output
- Full integration tests using `httptest` + in-memory services


┌─────────────┐     POST /api/v1/users     ┌─────────────┐
│ mygocloud   │ ─────────────────────────► │  API        │
│ user create │                            │  Server     │
└─────────────┘                            └─────────────┘
│
▼
~/.mygocloud/config.yaml  (endpoint + future auth token)



### Phase 3 — Docker Integration
> **Goal**: Async container lifecycle management

- `docker.Executor` implements the `Executor` interface
- Deployments run in background goroutines with app-level mutex locks
- Container status polling (`pending` → `running` → `successful/failed`)
- Rollback stops active containers and redeploys last successful version
- Log streaming via `docker logs`

Deployment Flow
─────────────────
mygocloud app deploy <id><app><dep>


## 🚀 Quick Start

### 1. Start PostgreSQL
```bash
make docker-up

make run
# or
go run ./cmd/api
# Server starts on :8080

make build-cli
./bin/mygocloud login --endpoint http://localhost:8080

# Create a user
mygocloud user create --email dev@example.com --name "Developer"

# Create an application
mygocloud app create --name api-gateway --description "Edge proxy" \
  --image nginx --owner-id <user-uuid>

# Deploy a version
mygocloud app deploy <app-uuid> --version 1.0.0

# Watch deployment status
mygocloud viz status <app-uuid> --watch

# View deployment history
mygocloud app deployments <app-uuid>

# Rollback to last successful
mygocloud app rollback <app-uuid>

# Stream logs
mygocloud viz logs <app-uuid> --follow


| Target                      | What It Runs                                                                 | Purpose                                                |
| --------------------------- | ---------------------------------------------------------------------------- | ------------------------------------------------------ |
| `make test-unit`            | `go test ./internal/... ./cli/config/... ./cli/printer/... ./cli/client/...` | Fast, isolated tests with mocked repos/services        |
| `make test-integration`     | `go test -run Integration ./...`                                             | End-to-end API tests with in-memory DB + noop executor |
| `make test-cli`             | `go test ./cli/commands/...`                                                 | Cobra command tests with mock HTTP servers             |
| `make test-cli-integration` | `go test -run TestCLI ./cli/...`                                             | CLI → Real API integration (spins up httptest server)  |
| `make coverage`             | Full suite + HTML report                                                     | Portfolio-ready coverage visualization                 |



make test        # Runs all four test suites

Developer] wants to deploy api-gateway:v1.0.0

1. $ mygocloud app deploy <app-id> -v 1.0.0
   └──► API creates deployment record (pending)
   └──► Background worker pulls nginx:1.0.0
   └──► Container starts → status updated to "successful"

2. $ mygocloud app deploy <app-id> -v 1.1.0
   └──► New deployment queued
   └──► Old container stopped (status: "stopped")
   └──► New container running (status: "successful")

3. $ mygocloud app rollback <app-id>
   └──► Finds last successful version (1.1.0)
   └──► Stops current container
   └──► Redeploys 1.1.0 → "successful"

   📂 Project Structure
plain
.
├── cmd/api              # API server entrypoint
├── cmd/mygocloud        # CLI entrypoint
├── cli/                 # CLI packages (client, commands, config, printer)
├── internal/
│   ├── config/          # Env-based configuration
│   ├── domain/          # Models, repository interfaces, service interfaces
│   ├── handler/         # HTTP handlers (chi)
│   ├── middleware/      # Request logging
│   ├── repository/      # memory + postgres implementations
│   ├── server/          # HTTP server bootstrap
│   ├── service/         # Business logic
│   └── executor/        # Docker abstraction
├── migrations/          # PostgreSQL schema
└── Makefile             # Build & test automation


