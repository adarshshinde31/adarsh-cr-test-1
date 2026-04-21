# feature-flags

A minimal HTTP CRUD service for managing feature flags, backed by the local filesystem. The storage layer is defined by the `flags.Store` interface so alternative backends (Postgres, Redis, remote config, etc.) can be added without touching the API handlers.

## Layout

```
main.go                       # wiring: flag parsing, HTTP server, signal handling
internal/flags/flag.go        # FeatureFlag domain type
internal/flags/store.go       # Store interface + sentinel errors
internal/flags/filestore.go   # filesystem-backed implementation
internal/api/handlers.go      # HTTP handlers / routing
```

## Run

```sh
go run . --addr :8080 --data-dir ./data/flags
```

## API

| Method | Path           | Body                                      | Response |
|--------|----------------|-------------------------------------------|----------|
| GET    | `/flags`       | —                                         | `200` list |
| POST   | `/flags`       | `{"key","enabled","description"}`         | `201` flag / `409` exists |
| GET    | `/flags/{key}` | —                                         | `200` flag / `404` |
| PUT    | `/flags/{key}` | `{"enabled","description"}`               | `200` flag / `404` |
| DELETE | `/flags/{key}` | —                                         | `204` / `404` |

Keys must match `^[a-zA-Z0-9_.-]{1,128}$`.

## Adding a new backend

Implement the `flags.Store` interface and swap it in `main.go`:

```go
type Store interface {
    Create(ctx context.Context, flag FeatureFlag) error
    Get(ctx context.Context, key string) (FeatureFlag, error)
    List(ctx context.Context) ([]FeatureFlag, error)
    Update(ctx context.Context, flag FeatureFlag) error
    Delete(ctx context.Context, key string) error
}
```
