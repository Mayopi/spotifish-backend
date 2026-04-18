# Spotifish Backend Architecture

Spotifish is a self-hosted Go backend service that bridges a PostgreSQL database, Google Drive APIs, and an Android client. The architecture strictly adheres to a layered design utilizing Dependency Injection in Go, allowing clear separation of concerns, testability, and explicit dependencies.

## Overall Stack

*   **API Framework:** Go standard library `net/http` combined with `github.com/gin-gonic/gin` for fast routing, middleware chaining, and JSON responses.
*   **Database:** PostgreSQL 16.
*   **Database Driver:** `github.com/jackc/pgx/v5` (and `pgxpool`), optimized for Postgres over standard `database/sql`, omitting ORMs in favor of typed raw SQL.
*   **Migrations:** Schema management done natively via `github.com/golang-migrate/migrate/v4`.
*   **Background Jobs:** Basic scheduling via `github.com/robfig/cron/v3` (e.g., standard background sync routines).
*   **Authentication:** Google OAuth 2.0 Web ID flows passing to custom local HS256 JWT tokens.

## Project Structure

The project conforms to standard Go layout patterns, restricting executable setups from shared library/domain logic:

```
spotifish-backend/
├── cmd/
│   └── server/          # Executable entrypoints
│       └── main.go      # Ties the configuration, database, routing, and DI graph together
├── internal/            # Private application and domain code
│   ├── config/          # Environment variable ingestion
│   ├── crypto/          # Internal helpers (e.g. AES-256-GCM encryption for Drive credentials)
│   ├── database/        # pgxpool connection generation and automated schema migration runner
│   ├── handler/         # Presentation layer: HTTP Gin Handlers, request unpacking / response packing
│   ├── middleware/      # Gin middleware for cross-cutting concepts (Auth checking, CORS, Rate Limit)
│   ├── model/           # Domain entities (Song, User, Playlists, API Responses) shared across layers
│   ├── repository/      # Persistence layer: Data access interfaces mapped with pgxpool raw SQL
│   ├── server/          # Setup scripts to initiate dependencies via Dependency Injection and map routers
│   ├── service/         # Business layer: Orchestration logic and algorithms crossing multiple Repositories
│   └── worker/          # Background worker instances (e.g., SyncWorker for routine cloud syncs)
├── migrations/          # Uncompiled `.sql` up/down scripts
├── Dockerfile           # Production and build container specification
└── docker-compose.yml   # Dev runtime dependencies bindings (Postgres + Network)
```

## Layered Design Concepts

The layers only depend strictly downwards: `Handler -> Service -> Repository`.
The `model` directory contains cross-boundary structs (domain bounds), which are freely used across all tiers.

1.  **Handler Layer (`internal/handler`)**:
    *   Injects dependencies from `service`s via constructor blocks (`NewSongHandler(libSvc *service.LibraryService)`).
    *   No business logic. Extracts path parameters, query parameters, bindings, Context user identification.
    *   Standardizes JSON outputs, pushing domain failures to uniform `model.ErrorResponse`.
2.  **Service Layer (`internal/service`)**:
    *   Injects dependencies from `repository` instances.
    *   Contains the application's unique business logic (e.g., handling cross-entity tasks: Fetch Album, Decrypt Key, Perform Sync with Drive).
    *   Abstracts the data fetching source from the Handler layer.
3.  **Repository Layer (`internal/repository`)**:
    *   Injects the `*pgxpool.Pool`.
    *   Only concerns itself with translating memory Structs (`*model.Song`, `*model.User`) into SQL statements and marshaling SQL rows back into model representations.
    *   Never operates on complex "business logic", solely data mapping and constrained integrity rules (like Upserts).

## Dependency Injection and Bootstrapping

Dependencies are not declared statically or globally, preserving pure function testability.
Instead, components are explicitly registered and linked during bootstrapping inside `internal/server/dependencies.go` and `cmd/server/main.go`.

*   `main.go` parses the system environment and establishes external data channels (Logging singleton config, PostgreSQL pooling).
*   `InitDependencies` (in `internal/server/dependencies.go`) cascades the initializations, spawning Repositories, passing them to Services, and packaging the outputs.
*   `SetupRouter` binds the Handler methods to specific Gin API paths.

This isolates tests, as replacing a given module interface or bypassing network dependencies is done purely at initialization.
