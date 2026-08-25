# SweetRPG Shelf API

[![Unit tests](https://github.com/sweetrpg/shelf-api/actions/workflows/go-ci.yml/badge.svg)](https://github.com/sweetrpg/shelf-api/actions/workflows/go-ci.yml)
[![Docker Build](https://github.com/sweetrpg/shelf-api/actions/workflows/docker-build.yml/badge.svg)](https://github.com/sweetrpg/shelf-api/actions/workflows/docker-build.yml)
[![License](https://img.shields.io/github/license/sweetrpg/shelf-api.svg)](https://img.shields.io/github/license/sweetrpg/shelf-api.svg)
[![Issues](https://img.shields.io/github/issues/sweetrpg/shelf-api.svg)](https://img.shields.io/github/issues/sweetrpg/shelf-api.svg)
[![PRs](https://img.shields.io/github/issues-pr/sweetrpg/shelf-api.svg)](https://img.shields.io/github/issues-pr/sweetrpg/shelf-api.svg)
[![Dependabot](https://badgen.net/github/dependabot/sweetrpg/shelf-api)](https://badgen.net/github/dependabot/sweetrpg/shelf-api)

[![Go](https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://go.dev)

HTTP microservice for the SweetRPG Shelf domain (library, wishlist, tables, visibility). Migrated from Python to Go to match the platform's Go microservice baseline. A thin Gin-based layer: `server/*.go` wires JSON:API routes to `shelf-data.go`'s data-access functions.

## Building

```bash
go build -v ./...
```

## Testing

```bash
go test -v -coverprofile coverage.out ./...
```

## Documentation

API documentation is available at `http://localhost:8000/swagger/index.html` when the server is running. See `service-conventions.md` in the platform repo for more details on the Go microservice baseline.
