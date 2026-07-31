# Copilot Agent Instructions — exchange-client

Trust these instructions first. Only search the repo if a detail here is incomplete or incorrect.

## Repository Summary

This is a **multi-language client library monorepo** for the Xifer Exchange. It provides Go and TypeScript clients generated from a shared OpenAPI spec.

- **Go module:** `github.com/ciphermountain/exchange-client/go` (in `go/`)
- **npm package:** `@ciphermountain/exchange-client` (in `ts/`)
- **Code generation:** OpenAPI 3.0 spec → Go client via `oapi-codegen`, TypeScript types via `openapi-typescript`
- **OpenAPI bundling:** Redocly CLI (Node.js, `@redocly/cli`)
- **No CI/CD workflows** (no `.github/workflows/` directory exists)

## Build & Validation Commands

### Prerequisites

- **Go ≥ 1.25.5** (`go version`)
- **Node.js ≥ 22** with `@redocly/cli` installed globally (`npm install -g @redocly/cli`)
- **golangci-lint v2** for linting (`golangci-lint run`)

### Command Reference

| Task | Command | Notes |
|------|---------|-------|
| **Build (Go)** | `cd go && go build ./...` | Always succeeds if `go mod tidy` has been run |
| **Test (Go)** | `cd go && go test ./...` | Tests exist only in `go/pkg/rest/` |
| **Lint (Go)** | `make lint` | Uses default golangci-lint v2 config (no `.golangci.yml`) |
| **Tidy modules** | `cd go && go mod tidy` | Always run after changing dependencies |
| **Generate Go** | `make generate-go` | Runs `scripts/bundle.sh` then `cd go && go generate ./...` |
| **Generate TS** | `make generate-ts` | Runs `scripts/bundle.sh` then `cd ts && npm run generate` |
| **Generate all** | `make generate-all` | Bundles spec, generates Go + TS |
| **Build (TS)** | `cd ts && npm run build` | Compiles TypeScript to `ts/dist/` |
| **Bundle only** | `./scripts/bundle.sh` | Bundles OpenAPI specs via `redocly bundle` |

### Critical: Code Generation Order

Always use `make generate-go` or `make generate-ts` to regenerate clients. These run two steps in order:

1. `./scripts/bundle.sh` — bundles OpenAPI YAML files into `openapi/v1_bundle.yaml` using Redocly
2. Language-specific generation from the bundled spec

**Do NOT run `go generate ./...` alone** unless `openapi/v1_bundle.yaml` already exists.

### Known Dependency Issues (Go)

The `go/go.mod` file contains three critical `replace` directives. **Do not remove them:**

```
replace (
    github.com/dprotaso/go-yit => github.com/dprotaso/go-yit v0.0.0-20220510233725-9ba8df137936
    github.com/speakeasy-api/jsonpath => github.com/speakeasy-api/jsonpath v0.6.0
    github.com/speakeasy-api/openapi-overlay => github.com/speakeasy-api/openapi-overlay v0.10.2
)
```

- **`go-yit`**: Pinned to avoid `go.yaml.in/yaml/v4` which causes type conflicts with `gopkg.in/yaml.v3`
- **`jsonpath`**: Pinned to v0.6.0 because newer versions reference a non-existent `pkg/overlay` package
- **`openapi-overlay`**: Pinned to v0.10.2 for compatibility with the jsonpath pin

### Lint Note

`make lint` currently reports 1 existing `errcheck` issue in `go/pkg/ws.go` (unchecked `WriteMessage` return). This is pre-existing; do not introduce new lint issues.

## Project Layout

```
Makefile                    # Build targets: lint, generate-go, generate-ts, generate-all
redocly.yaml                # Redocly CLI config for OpenAPI bundling
scripts/
  bundle.sh                 # Bundles OpenAPI specs (installs redocly if missing)
  check-generate.sh         # Verifies generated files are up to date
openapi/
  root.yaml                 # OpenAPI 3.0 root spec (entry point)
  v1_bundle.yaml            # Generated bundled spec (output of redocly bundle)
  paths/                    # Per-endpoint OpenAPI path definitions
  components/
    schemas/                # OpenAPI schema definitions
    parameters/             # OpenAPI parameter definitions
    responses/              # OpenAPI response definitions
go/
  go.mod                    # Go module definition
  go.sum                    # Dependency checksums
  pkg/
    generate.go             # go:generate directive for oapi-codegen
    config.yaml             # oapi-codegen configuration (output: rest/client.gen.go)
    rest.go                 # RestClient — high-level REST API wrapper
    ws.go                   # WSClient — WebSocket client with subscriptions
    mailbox.go              # Generic mailbox (channel-based message queue)
    symbol.go               # Symbol/Market type parsing helpers
    messages/
      ws.go                 # WebSocket message types (Heartbeat, Order, Trade, etc.)
    rest/
      client.gen.go         # GENERATED — do not edit manually (~3300 lines)
      client_test.go        # Tests for generated client types
ts/
  package.json              # @ciphermountain/exchange-client npm package
  tsconfig.json             # TypeScript configuration
  src/
    index.ts                # Public exports (types + client factory)
    client.ts               # openapi-fetch wrapper
    generated/
      schema.ts             # GENERATED — TypeScript types from OpenAPI spec
```

### Key Architecture

- **`go/pkg/rest.go`** (`RestClient`): Wraps the generated `go/pkg/rest/client.gen.go` client with authentication and response parsing
- **`go/pkg/ws.go`** (`WSClient`): WebSocket client with heartbeat, order book, ticker, and trade subscriptions
- **`go/pkg/rest/client.gen.go`**: Auto-generated by `oapi-codegen` — never edit this file directly
- **`ts/src/client.ts`**: Type-safe fetch client using `openapi-fetch`
- **`ts/src/generated/schema.ts`**: Auto-generated by `openapi-typescript` — never edit this file directly

### Adding/Modifying API Endpoints

1. Edit or add YAML files in `openapi/paths/` and `openapi/components/`
2. Update `openapi/root.yaml` if adding new paths
3. Run `make generate-all` to regenerate both clients
4. Run `cd go && go build ./... && go test ./...` to validate Go
5. Run `cd ts && npm run build` to validate TypeScript
6. Run `make lint` to check for lint issues

### Validation Checklist

After any change, always run these in order:

1. `cd go && go mod tidy` (if Go dependencies changed)
2. `make generate-all` (if OpenAPI specs changed)
3. `cd go && go build ./...`
4. `cd go && go test ./...`
5. `cd ts && npm run build`
6. `make lint`
