# AGENTS.md

This file provides guidance to Claude Code, Codex, GitHub Copilot, and other AI coding agents
working in this repository.

## About This Project

game-room-api is the HTTP microservice for the SweetRPG Game Room domain (library, wishlist, tables,
visibility). Migrated from Python to Go (see sweetrpg/platform's shelf-service OpenSpec change)
to match the platform's Go microservice baseline. A thin Gin-based layer: `server/*.go` wires
JSON:API routes to `game-room-data.go`'s data-access functions. Dependencies: api-core.go,
game-room-data.go, game-room-objects.go, common.go, mongodb.go.

## Rate limiting

Per-client/IP rate limiting is on by default via the shared `api-core.go/ratelimit` middleware:
Redis-backed counters keyed by `X-API-Key` else client IP, `cheap` tier for `/status/*` and
`standard` for everything else, fail-closed 503 when Redis is unreachable, 429 on exceed. Tune
with `RATE_LIMIT_CHEAP`/`RATE_LIMIT_CHEAP_WINDOW_SECONDS`/`RATE_LIMIT_STANDARD`/
`RATE_LIMIT_STANDARD_WINDOW_SECONDS`. See `platform`'s
`openspec/changes/fix-rate-limiting-per-client-ip`.

## Committing Code

Use [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>
```

## Branches and Workflow

* `develop` - integration branch, default branch, target for all PRs.
* `master` - latest released state, nothing committed directly.
* `feature/*`, `fix/*` branched from `develop`; `hotfix/*` branched from `master`.

See `CONTRIBUTING.md` for the full workflow.

## Running Checks Locally

```bash
go build -v ./...
go vet ./...
go test -v -coverprofile coverage.out ./...
```

The database-backed tests in `server` and `internal/consumer` need a MongoDB and fail without one:

```bash
TEST_DB_URI=mongodb://localhost:27017/unit-tests go test ./...
```

## Releases

See `RELEASE.md`. Summary: trigger `prepare-release.yaml` (`workflow_dispatch` against
`develop`), which computes the next version from conventional commits via git-cliff and opens
a `release/<version>` PR into `master`. Merging that PR tags the release
(`tag-release.yaml`), which triggers `release.yaml` - re-runs tests, creates a GitHub
Release, and merges `master` back into `develop`.

## Platform Conventions and Decisions

This repo is a submodule of `sweetrpg/platform`. Platform-wide conventions and Architecture
Decision Records live there:

- checked out inside the platform tree: `../../docs/README.md` (convention index) and
  `../../docs/adr/README.md` (platform ADRs, `PADR-*`)
- standalone or module-only checkout:
  <https://github.com/sweetrpg/platform/tree/master/docs> and
  <https://github.com/sweetrpg/platform/tree/master/docs/adr>

Accepted `PADR-*` records are binding constraints. Read the ADR index before proposing a
structural change; if a task needs to contradict an accepted record, stop and say so -
propose a superseding ADR rather than working around it. This repo's own service-local
decisions are `ADR-*` and belong in `docs/adr/` here.

Before implementing any feature, complete the
[feature work checkpoint](https://github.com/sweetrpg/platform/blob/master/docs/openspec.md#feature-work-checkpoint)
and record the answers in the issue or PR.
