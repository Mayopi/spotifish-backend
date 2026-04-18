# HTTP Responses & Contracts

Spotifish maintains a completely uniform JSON contract specification designed exclusively to be parseable effortlessly by external, strongly-typed frontends (like Kotlin Android representations or Typescript schemas).

## Standard Error Format

Internal service breakages or client logic validation faults consistently throw standardized error outputs avoiding arbitrary string injections inside root response bodies. All errors across generic bounds (whether an authentication missing signature, invalid paging bounds, or unhandled persistence faults) mutate into identical top-level properties binding an overarching `model.ErrorResponse`:

```json
{
  "error": {
    "code": "invalid_request",
    "message": "q parameter is required"
  }
}
```

*   **`code`:** Deterministic snake_case categorical tokens representing the discrete structural failure identity (e.g., `not_found`, `list_songs_error`, `token_expired`). These uniquely trigger deterministic programmatic fallbacks directly in the client layer without fuzzy text matching.
*   **`message`:** Human-readable context debug strings explaining the exact origin fault context.

## Success Responses

Successful state execution consistently returns `200 OK` bindings. Data isn't dumped arbitrarily at root structures but housed cohesively within standard hash bindings, most commonly rendered dynamically through generic `gin.H` representations returning mapping slices (`songs`, `albums`).

### Empty List Handling

Spotifish explicitly accounts for client-side serialization boundaries with nullable properties. Specifically, slice objects fetching completely empty results natively from PostgreSQL repositories (such as the `SongRepository` executing subset bounds lacking existing results) construct empty, un-allocated array types via standard initialization rules:
```go
songs := make([]*model.Song, 0, limit)
```

By ensuring zero-state initializations aren't kept inherently `nil`, the `gin` standard library encoder serializes arrays as `[]` rather than passing generic `null` JSON payloads. This directly stabilizes incremental client syncing procedures where strict languages (like Kotlin) inherently fail parsing non-nullable `List` properties upon receiving server `null` variables.

## Pagination Outputs

Paginated bounds append explicit token extensions to data responses when subsequent metadata objects successfully exist inside repository subsets:

```json
{
  "songs": [
    ... // Array mappings 
  ],
  "nextCursor": "de9ce982-f54e-41ce-abdf-14c45aeaa32f"
}
```
If boundaries resolve as comprehensive (last index mapping), `nextCursor` inherently omits the payload representation allowing clients to definitively cease fetching loops.

## Authentication Identity Extraction

Following validation bindings against active client metadata contexts natively through `internal/middleware` logic parsing token headers (`Authorization: Bearer <token>`), underlying structural mapping contexts abstract security context variables from domain HTTP Handlers natively through uniform extraction procedures:
```go
userID := middleware.GetUserID(c)
```
Handlers operate securely recognizing active authorization boundaries solely over the passed contextual token parameter.
