# SweetRPG Game Room API

[![Unit tests](https://github.com/sweetrpg/game-room-api/actions/workflows/ci.yaml/badge.svg)](https://github.com/sweetrpg/game-room-api/actions/workflows/ci.yaml)
[![License](https://img.shields.io/github/license/sweetrpg/game-room-api.svg)](https://img.shields.io/github/license/sweetrpg/game-room-api.svg)
[![Issues](https://img.shields.io/github/issues/sweetrpg/game-room-api.svg)](https://img.shields.io/github/issues/sweetrpg/game-room-api.svg)
[![PRs](https://img.shields.io/github/issues-pr/sweetrpg/game-room-api.svg)](https://img.shields.io/github/issues-pr/sweetrpg/game-room-api.svg)
[![Dependabot](https://badgen.net/github/dependabot/sweetrpg/game-room-api)](https://badgen.net/github/dependabot/sweetrpg/game-room-api)

[![Go](https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://go.dev)

HTTP microservice for the SweetRPG Game Room domain (library, wishlist, tables, visibility).
Migrated from Python to Go to match the platform's Go microservice baseline. A thin Gin-based
layer: `server/*.go` wires JSON:API routes to `game-room-data.go`'s data-access functions.

## Building

```bash
go build -v ./...
```

## Testing

```bash
go test -v -coverprofile coverage.out ./...
```

## Documentation

API documentation is available at `http://localhost:8000/swagger/index.html` when the server is
running. See `service-conventions.md` in the platform repo for more details on the Go
microservice baseline.
