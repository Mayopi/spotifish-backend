# Code Patterns & Best Practices

The Spotifish codebase leverages several canonical Go patterns to maintain clean boundaries between HTTP routing, business logic, persistence, and error handling.

## Constructor-Based Dependency Injection (DI)

Spotifish avoids mutable global variables or singletons. Instead, instances are passed down through the codebase entirely via constructors.

*   Every handler, service, or repository possesses a function typically named `New[Target]()`.
*   Example inside `service/library_service.go`:
    ```go
    type LibraryService struct {
    	songRepo     *repository.SongRepository
    	playbackRepo *repository.PlaybackEventRepository
    }

    func NewLibraryService(songRepo *repository.SongRepository, playbackRepo *repository.PlaybackEventRepository) *LibraryService {
    	return &LibraryService{songRepo: songRepo, playbackRepo: playbackRepo}
    }
    ```
This enforces explicitness on what dependencies a specific logical unit requires.

## `context.Context` Propagation

Go's `context.Context` is the foundational tool for carrying cancellation signals and request-scoped values across application boundaries. Spotifish enforces contexts rigorously.

1.  **Request Start:** The active request's context is extracted at the Gin Handler level by retrieving `c.Request.Context()`.
2.  **Propagation:** Context travels sequentially: `Handler -> Service -> Repository`.
3.  **Cancellation enforcement:** Inside `internal/repository`, standard pgx endpoints (`QueryRow`, `Query`, `Exec`) bind directly to the passed Context. If a request terminates or disconnects early, the database layer aborts execution automatically.

**Example in `handler/song_handler.go`:**
```go
// The context passes seamlessly from HTTP boundaries straight to the persistence layer.
song, err := h.libSvc.GetSong(c.Request.Context(), userID, songID)
```

## Configuration Management

Spotifish consolidates configuration to strict typing via `internal/config`, relying on POSIX environment variables. Default `.env` configurations are only pulled via `godotenv` during non-containerized debug sessions inside `cmd/server/main.go`. 

Variables bind explicitly to a `Config` struct. Failure to provide essential secrets (`JWT_SIGNING_KEY`, `DATABASE_URL`) triggers startup fatals, adhering to the fail-fast philosophy rather than failing on first-request.

## Consistent Structured Logging (`zerolog`)

Rather than relying on `log.Printf` unstructured formatting, Spotifish implements globally configured structured logging through `github.com/rs/zerolog`.
Structured JSON output ensures logs aggregate beautifully through indexing suites, attributing log points automatically without messy string parsing.

*   **Setup:** Global parameters (Log Level) are established entirely during initial server hydration inside `main.go`.
*   **Usage:** Anywhere across the application, metadata gets stitched immediately onto the output payload safely.
    ```go
    log.Info().Str("userID", user.ID).Int("count", newSongs).Msg("metadata extraction complete")
    ```

## Custom Middleware Injection

Standard boilerplate concerns affecting multiple endpoints are stripped from handlers completely and abstracted linearly in `internal/middleware/`:

*   **Recovery:** Intercepts runtime panics from Gin execution layers gracefully formatting them as standard internal HTTP server error responses, keeping node integrity intact.
*   **Rate Limits:** Standard client throttling applied via declarative memory blocks avoiding external storage caching when possible for performance.
*   **Auth Extraction (`middleware.GetUserID(c)`):** Decodes JWT signatures attached on HTTP headers. Successfully decoupled users store deterministic identifying metadata embedded directly inside the `*gin.Context` instance avoiding extraneous round trips into databases.
