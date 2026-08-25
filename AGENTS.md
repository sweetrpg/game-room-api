# AGENTS.md

This file provides guidance to Claude Code, Codex, GitHub Copilot, and other AI coding agents
working in this repository.

## About This Project

game-room-api is the HTTP microservice for the SweetRPG Game Room domain (library, wishlist, tables,
visibility). Migrated from Python to Go (see sweetrpg/platform's shelf-service OpenSpec change)
to match the platform's Go microservice baseline. A thin Gin-based layer: `server/*.go` wires
JSON:API routes to `game-room-data.go`'s data-access functions. Dependencies: api-core.go,
game-room-data.go, game-room-objects.go, common.go, mongodb.go.

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

## Releases

See `RELEASE.md`. Summary: trigger `prepare-release.yaml` (`workflow_dispatch` against
`develop`), which computes the next version from conventional commits via git-cliff and opens
a `release/<version>` PR into `master`. Merging that PR tags the release
(`tag-release.yaml`), which triggers `release.yaml` - re-runs tests, creates a GitHub
Release, and merges `master` back into `develop`.
