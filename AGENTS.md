# Repository Guidelines

## Project Structure & Module Organization

This repository is a Go-based retrieval-augmented generation service (`module rag`). The executable entry point is `cmd/main.go`. HTTP routing, handlers, middleware, services, repositories, models, and background processing live under `internal/`. Reusable integrations and processing components are under `pkg/`, including extraction, chunking, embeddings, Milvus access, JWT handling, and response utilities. Runtime configuration is in `config/config.ini` and its Go loader in `config/config.go`; Docker dependencies are defined in `deployments/docker-compose.yml`. Tests currently sit beside their package code, such as `pkg/chunker/chunker_test.go`.

## Build, Test, and Development Commands

- `docker compose -f deployments/docker-compose.yml up -d` starts Milvus and its etcd/MinIO dependencies; wait for `milvus-standalone` to become healthy.
- `go run cmd/main.go` runs the service locally using `config/config.ini`.
- `go build -o main cmd/main.go` builds the application binary.
- `go test ./...` runs the complete test suite; use `go test ./pkg/chunker` for focused iteration or `go test -v ./...` for verbose output.
- `go mod tidy` synchronizes module dependencies after dependency changes.

PostgreSQL and Milvus must be reachable before starting the service. Set `DASHSCOPE_API_KEY` for real embeddings, or use the `mock` embedding provider in local configuration for pipeline testing.

## Coding Style & Naming Conventions

Format Go code with `gofmt` (for example, `gofmt -w internal/service/query.go`). Follow standard Go naming: exported identifiers use `PascalCase`, unexported identifiers use `camelCase`, and package names are short lowercase words. Keep responsibilities aligned with the existing layers and return errors explicitly; avoid introducing global state or unrelated refactors.

## Testing Guidelines

Use Go’s standard `testing` package. Name tests with `Test<Type>_<Behavior>` or another descriptive `Test...` form, and keep unit tests close to the implementation. Add regression coverage for changes to chunking, extraction, repositories, services, or request handling. Run `go test ./...` before submitting changes; integration tests may require PostgreSQL, Milvus, and suitable credentials.

## Commit & Pull Request Guidelines

Recent history uses concise Conventional Commit-style subjects such as `feat: ...` and `refactor: ...`; follow that pattern (`fix: handle empty document`, for example). Keep commits focused. Pull requests should explain the behavior change, list validation commands, call out configuration or schema changes, and include request/response examples or screenshots when modifying API behavior.

## Security & Configuration Tips

Do not commit API keys, JWT secrets, database credentials, uploaded files, or generated binaries. Review changes to authentication and ownership checks carefully, and keep secrets in environment variables or local configuration outside version control.
