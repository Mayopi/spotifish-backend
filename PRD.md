# 📄 Backend Product Requirements Document (PRD)

## 🎧 Project: Spotifish Backend Service

**Status:** v1 — decisions locked in (see [§14 Decisions](#14-decisions-locked-in))
**Supersedes (in part):** [`PRD.md`](./PRD.md) — sections 3 (Drive), 7 (Favorites), 8 (Playlists), 12 (Data Management), 14 (Security)
**Out of scope vs old PRD:** local-only file scanning, on-device sync, local JSON persistence

> **Repo layout:** the backend lives in a **separate repository**. This repo only contains the rewritten Android client. This document specifies the API contract that both repos must agree on.

---

## 1. Why a Backend?

The current app is a fat client: Drive sync, metadata extraction, persistence, and library state all live on-device. That made sense as an MVP but creates real problems as the library and feature set grow:

| Pain point on the fat client today                                                                     | What a backend gives us                                      |
| ------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------ |
| Re-syncing 1k+ Drive files burns minutes of mobile bandwidth and battery on every install / re-install | Sync runs once, server-side; clients just pull a small delta |
| Metadata extraction (FLAC tags, album art) re-runs per-device                                          | Extract once, serve to every client                          |
| No way to share a library between phone and tablet, or hand off to a future web client                 | Server is the single source of truth                         |
| OAuth refresh is fragile across process death and Identity Services quirks                             | Refresh tokens stored server-side, never expire silently     |
| Playlists / favorites only exist on the device that created them                                       | Sync across clients automatically                            |
| Adding any social feature later (sharing, following, listening history) is impossible without a server | Built into the foundation                                    |
| No analytics, no observability, no way to fix bugs without shipping a new APK                          | Server-side logs + metrics                                   |

---

## 2. Goals & Non-Goals

### 2.1 Goals (MVP)

1. **Single source of truth for the music library.** Server owns the metadata, the client renders it.
2. **Server-side Drive sync.** Server holds Google OAuth refresh tokens, scans the user's Drive folder, extracts metadata + album art, persists to a database.
3. **Streaming playback.** Client gets a short-lived signed URL or proxied stream URL for each track. Bearer-token plumbing disappears from the client.
4. **Cross-device library state.** Playlists, favorites, recently-played, and now-playing position sync across devices for the same user.
5. **Account-based auth.** Sign in with Google → backend issues a session JWT. The Android app talks to the backend with the JWT, not directly with Drive.
6. **Backwards-compatible feature set.** Every feature that exists in the current app keeps working from the user's POV.

### 2.2 Non-Goals (Phase 1)

- Multi-tenant SaaS / public signups. Single-user or small invite-list only.
- Music recommendations / ML.
- Social features (sharing, following, comments).
- Audio transcoding (clients play whatever Drive returns).
- Lyrics, podcast support, video.
- Offline mode for the Android client beyond a thin "last known library" cache.
- Web client (designed for, not built in Phase 1).
- Native iOS client.

---

## 3. Personas

| Persona                | Description                                                             | Needs                                                 |
| ---------------------- | ----------------------------------------------------------------------- | ----------------------------------------------------- |
| **Owner**              | The person hosting the backend (initially: just you). Has admin access. | One-click install, low maintenance, low cost          |
| **Authenticated user** | Someone signed in via Google. In Phase 1, only the owner.               | Fast library, reliable playback, multi-device sync    |
| **Listener device**    | Android phone/tablet running the rewritten client                       | Small payloads, smooth streaming, background playback |

---

## 4. High-Level Architecture

```
┌──────────────────────┐         ┌──────────────────────────────────────────┐
│  Android Client      │         │  Backend Service                         │
│                      │         │                                          │
│  ┌────────────────┐  │  HTTPS  │  ┌────────────┐    ┌──────────────────┐ │
│  │  Compose UI    │  │ <─────> │  │  REST API  │    │  Auth Service    │ │
│  └────────────────┘  │   JWT   │  │  (Ktor)    │<──>│  (Google OAuth   │ │
│         │            │         │  └────────────┘    │   + JWT issuer)  │ │
│  ┌────────────────┐  │         │        │           └──────────────────┘ │
│  │  ViewModels    │  │         │        │                                 │
│  └────────────────┘  │         │  ┌─────▼──────┐    ┌──────────────────┐ │
│         │            │         │  │  Library   │<──>│  PostgreSQL      │ │
│  ┌────────────────┐  │         │  │  Service   │    │  (songs, users,  │ │
│  │  Repositories  │  │         │  └────────────┘    │   playlists,...) │ │
│  │  (HTTP only)   │  │         │        │           └──────────────────┘ │
│  └────────────────┘  │         │  ┌─────▼──────┐    ┌──────────────────┐ │
│         │            │         │  │  Sync      │───>│  Object Storage  │ │
│  ┌────────────────┐  │         │  │  Worker    │    │  (album art)     │ │
│  │  Media3 Player │  │         │  └────────────┘    └──────────────────┘ │
│  │  (HTTP source) │  │         │        │                                 │
│  └────────────────┘  │         │        ▼                                 │
│         ▲            │         │  ┌────────────┐                          │
└─────────┼────────────┘         │  │  Google    │                          │
          │                      │  │  Drive API │                          │
          │  Signed stream URL   │  └────────────┘                          │
          └──────────────────────┴──────────────────────────────────────────┘
```

### 4.1 Component breakdown

| Component              | Responsibility                                            | Technology                                                                         |
| ---------------------- | --------------------------------------------------------- | ---------------------------------------------------------------------------------- |
| **REST API**           | HTTP endpoints, request validation, response shaping      | **Go + Gin**                                                                       |
| **Auth Service**       | Google OAuth flow, refresh-token storage, JWT issuance    | Go + `golang.org/x/oauth2` + `golang-jwt/jwt`                                      |
| **Library Service**    | CRUD on songs/playlists/favorites, search, browse         | Go + `sqlc` or `gorm`                                                              |
| **Sync Worker**        | Background Drive scanning, metadata extraction, art write | Go goroutines + `robfig/cron`                                                      |
| **Database**           | Source of truth                                           | **PostgreSQL 16**                                                                  |
| **Album Art Storage**  | Cover art bytes                                           | **Local disk** (`/var/lib/spotifish/art/`) — migration path to S3-compatible later |
| **Cache** _(optional)_ | Hot library reads, signed URL TTLs                        | Redis 7 (deferred until needed)                                                    |

> **Why Go?** Lowest memory footprint, single static binary, trivial Docker image. The trade-off is no shared types with the Kotlin Android client — DTOs are defined twice (once in OpenAPI/Go structs, once in Kotlin DTOs). Acceptable cost given the smaller deployment surface.

---

## 5. Functional Requirements

### 5.1 Authentication & Accounts

**FR-AUTH-1** — Users sign in with Google via the existing Identity Services flow on the Android client.
**FR-AUTH-2** — The Android client sends the resulting Google ID token to `POST /v1/auth/google` on the backend.
**FR-AUTH-3** — The backend verifies the ID token against Google's JWKS, finds-or-creates a user row, and issues a signed JWT (15 min) plus a refresh token (30 days, rotating).
**FR-AUTH-4** — All subsequent requests use `Authorization: Bearer <jwt>`. JWT carries `userId` and `email`.
**FR-AUTH-5** — `POST /v1/auth/refresh` rotates the refresh token and returns a new JWT pair.
**FR-AUTH-6** — `POST /v1/auth/sign-out` revokes the current refresh token.

### 5.2 Drive Connection

**FR-DRIVE-1** — `POST /v1/drive/connect` accepts a Google authorization code (server-side OAuth flow) and exchanges it for an access + refresh token, then stores them encrypted in the database keyed by `userId`.
**FR-DRIVE-2** — The backend handles all token refresh on its own. Clients never see Drive bearer tokens again.
**FR-DRIVE-3** — `GET /v1/drive/folders?parentId={id}` lists subfolders of the given parent (default `root`). Used by the in-app folder picker.
**FR-DRIVE-4** — `POST /v1/drive/connection` sets the active folder (`{folderId, folderName}`).
**FR-DRIVE-5** — `DELETE /v1/drive/connection` revokes the Drive grant, deletes stored tokens, and (optionally) the synced library rows.

### 5.3 Library Sync

**FR-SYNC-1** — `POST /v1/sync/run` enqueues a sync job for the authenticated user. Returns a `syncJobId`.
**FR-SYNC-2** — The Sync Worker walks the connected Drive folder recursively, comparing every file's `modifiedTime` against the cached `addedAtEpochMillis` in the DB.
**FR-SYNC-3** — For each new or modified file, the worker streams the prefix into `MediaMetadataRetriever` (server-side), extracts title/artist/album/duration + embedded picture, and uploads the picture to object storage.
**FR-SYNC-4** — Files deleted from Drive between syncs are removed from the DB.
**FR-SYNC-5** — `GET /v1/sync/status` returns `{state, progress, lastError, lastSyncedAt}` for the latest job.
**FR-SYNC-6** — Sync runs are idempotent and safe to enqueue multiple times. Concurrent runs for the same user are de-duplicated.
**FR-SYNC-7** — A scheduled cron triggers a sync every 6 hours per active user (configurable).

### 5.4 Library Browsing & Search

**FR-LIB-1** — `GET /v1/songs?cursor={c}&limit={n}` returns a paginated, sorted song list. Default sort: title asc.
**FR-LIB-2** — `GET /v1/songs/{songId}` returns full metadata for one song.
**FR-LIB-3** — `GET /v1/songs/search?q={query}` runs a full-text search across `title`, `artist`, `album`. Returns up to 100 hits.
**FR-LIB-4** — `GET /v1/artists` returns artist groups with song counts and a sample album-art URL.
**FR-LIB-5** — `GET /v1/artists/{artistId}/songs` returns the songs for one artist.
**FR-LIB-6** — `GET /v1/albums` returns album groups (keyed by album+artist).
**FR-LIB-7** — `GET /v1/albums/{albumId}/songs` returns the songs for one album.
**FR-LIB-8** — `GET /v1/home` returns the curated home sections (`Recently Added`, `Drive Library`, `Local Library` → may be retired post-migration, see §6, etc.).

### 5.5 Playback

**FR-PLAY-1** — `GET /v1/songs/{songId}/stream` is a **byte-proxy endpoint**. The backend opens a Drive download for the user's file and pipes the bytes straight back to the client with full HTTP `Range` support for ExoPlayer seeking. The Drive bearer token never leaves the backend.

The client treats this URL as opaque — it just hands `https://<backend>/v1/songs/{id}/stream` (with the Authorization header attached by `AuthInterceptor`) to ExoPlayer.

**FR-PLAY-2** — Stream URLs are valid for 1 hour, then the client must re-request `GET /v1/songs/{songId}/stream`.
**FR-PLAY-3** — Optional: the backend records `POST /v1/playback/events` (track started / completed / skipped) for future "recently played" support.

### 5.6 Playlists

**FR-PLST-1** — `GET /v1/playlists` returns the user's playlists.
**FR-PLST-2** — `POST /v1/playlists` creates a new playlist. Body: `{name}`.
**FR-PLST-3** — `PATCH /v1/playlists/{id}` renames.
**FR-PLST-4** — `DELETE /v1/playlists/{id}`.
**FR-PLST-5** — `POST /v1/playlists/{id}/songs` adds a song. Body: `{songId}`.
**FR-PLST-6** — `DELETE /v1/playlists/{id}/songs/{songId}` removes a song.
**FR-PLST-7** — `PUT /v1/playlists/{id}/songs` replaces the full ordered song list (used for reordering).

### 5.7 Favorites

**FR-FAV-1** — `GET /v1/favorites` returns the user's liked song IDs.
**FR-FAV-2** — `PUT /v1/favorites/{songId}` likes a song.
**FR-FAV-3** — `DELETE /v1/favorites/{songId}` unlikes.

### 5.8 Settings

**FR-SET-1** — `GET /v1/me/settings` returns user-scoped settings (theme, default sort, etc.).
**FR-SET-2** — `PATCH /v1/me/settings` updates settings.

---

## 6. What Happens to Local Music Scanning?

The current app also scans device-local audio via `MediaStore`. Three options for the backend migration:

| Option                                                                                            | Pros                                   | Cons                                         |
| ------------------------------------------------------------------------------------------------- | -------------------------------------- | -------------------------------------------- |
| **A. Drop local scanning entirely** — backend-only                                                | Simpler client, single source of truth | Users with on-device music lose it           |
| **B. Keep local scanning client-side, mix into the UI alongside backend results**                 | No regression, hybrid library          | Two repositories on the client, harder dedup |
| **C. Upload local files to Drive (or backend storage) so they become part of the synced library** | Clean architecture                     | Heavy migration, storage cost                |

**Decision: Option B.** `MediaStoreLocalMusicDataSource` stays on the client unchanged. The new `CompositeMusicRepository` merges its results with the backend-driven `RemoteMusicRepository`.

---

## 7. Non-Functional Requirements

| Category            | Requirement                                                                                                         |
| ------------------- | ------------------------------------------------------------------------------------------------------------------- |
| **Performance**     | Library list endpoint p95 < 200ms for libraries up to 5k songs. Stream URL endpoint p95 < 300ms.                    |
| **Sync throughput** | Backend should process at least 10 new files / second during a sync (mostly bound by Drive metadata HEAD requests). |
| **Availability**    | 99% (single VPS, single instance acceptable for personal use).                                                      |
| **Recovery**        | Database backups daily, retained 14 days. Object storage versioned.                                                 |
| **Security**        | Refresh tokens encrypted at rest with a key from env. JWT signing key rotated annually. HTTPS-only.                 |
| **Privacy**         | No third-party analytics. Logs scrub access tokens and email addresses.                                             |
| **Cost**            | < $10/month for a single user, hostable on a 1 vCPU / 1 GB VPS or Fly.io free tier.                                 |

---

## 8. Data Model (PostgreSQL)

```sql
-- Users
CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    google_sub      TEXT UNIQUE NOT NULL,        -- Google's stable subject id
    email           TEXT NOT NULL,
    display_name    TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Refresh tokens for the backend's own JWT auth
CREATE TABLE auth_refresh_tokens (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash      TEXT NOT NULL,               -- sha256, never plaintext
    expires_at      TIMESTAMPTZ NOT NULL,
    revoked_at      TIMESTAMPTZ
);

-- Drive OAuth credentials per user (encrypted at rest with app key)
CREATE TABLE drive_credentials (
    user_id           UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    encrypted_access  BYTEA NOT NULL,
    encrypted_refresh BYTEA NOT NULL,
    expires_at        TIMESTAMPTZ NOT NULL,
    scope             TEXT NOT NULL,
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Connected Drive folder
CREATE TABLE drive_folders (
    user_id        UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    folder_id      TEXT NOT NULL,
    folder_name    TEXT NOT NULL,
    last_synced_at TIMESTAMPTZ
);

-- Songs (the canonical library)
CREATE TABLE songs (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id              UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    source               TEXT NOT NULL CHECK (source IN ('drive', 'local')),
    source_file_id       TEXT NOT NULL,           -- Drive file id, or device path
    title                TEXT NOT NULL,
    artist               TEXT NOT NULL,
    album                TEXT NOT NULL,
    duration_ms          BIGINT NOT NULL DEFAULT 0,
    mime_type            TEXT,
    album_art_object_key TEXT,                   -- key into object storage
    drive_modified_at    TIMESTAMPTZ,            -- for change detection
    added_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, source, source_file_id)
);

CREATE INDEX songs_user_title_idx     ON songs (user_id, lower(title));
CREATE INDEX songs_user_artist_idx    ON songs (user_id, lower(artist));
CREATE INDEX songs_user_album_idx     ON songs (user_id, lower(album));
CREATE INDEX songs_fts_idx            ON songs USING gin (to_tsvector('simple', title || ' ' || artist || ' ' || album));

-- Playlists
CREATE TABLE playlists (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE playlist_songs (
    playlist_id UUID NOT NULL REFERENCES playlists(id) ON DELETE CASCADE,
    song_id     UUID NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
    position    INTEGER NOT NULL,
    PRIMARY KEY (playlist_id, song_id)
);

-- Favorites
CREATE TABLE favorites (
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    song_id    UUID NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, song_id)
);

-- Sync jobs
CREATE TABLE sync_jobs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    state           TEXT NOT NULL CHECK (state IN ('queued', 'running', 'succeeded', 'failed')),
    processed_count INTEGER NOT NULL DEFAULT 0,
    total_count     INTEGER,
    last_error      TEXT,
    started_at      TIMESTAMPTZ,
    finished_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

---

## 9. API Contract (REST, JSON)

### 9.1 Conventions

- Base URL: `https://<host>/v1`
- All requests except `/auth/*` require `Authorization: Bearer <jwt>`.
- Errors: `{ "error": { "code": "drive_unauthorized", "message": "..." } }` with appropriate HTTP status.
- Pagination: cursor-based via `?cursor=` query param. Responses include `nextCursor: string | null`.
- Timestamps: RFC3339 strings.

### 9.2 Endpoint summary

| Method   | Path                                | Purpose                                   |
| -------- | ----------------------------------- | ----------------------------------------- |
| `POST`   | `/v1/auth/google`                   | Exchange Google ID token for our JWT pair |
| `POST`   | `/v1/auth/refresh`                  | Rotate refresh token                      |
| `POST`   | `/v1/auth/sign-out`                 | Revoke refresh token                      |
| `GET`    | `/v1/me`                            | Current user profile                      |
| `GET`    | `/v1/me/settings`                   | User settings                             |
| `PATCH`  | `/v1/me/settings`                   | Update settings                           |
| `POST`   | `/v1/drive/connect`                 | Begin Drive OAuth (server-side)           |
| `GET`    | `/v1/drive/folders`                 | List subfolders for the picker            |
| `POST`   | `/v1/drive/connection`              | Set active folder                         |
| `DELETE` | `/v1/drive/connection`              | Disconnect Drive                          |
| `POST`   | `/v1/sync/run`                      | Enqueue a sync                            |
| `GET`    | `/v1/sync/status`                   | Latest sync status                        |
| `GET`    | `/v1/songs`                         | List songs (paginated)                    |
| `GET`    | `/v1/songs/{id}`                    | Song detail                               |
| `GET`    | `/v1/songs/search`                  | Search                                    |
| `GET`    | `/v1/songs/{id}/stream`             | Get streamable URL                        |
| `GET`    | `/v1/artists`                       | List artists                              |
| `GET`    | `/v1/artists/{id}/songs`            | Songs by artist                           |
| `GET`    | `/v1/albums`                        | List albums                               |
| `GET`    | `/v1/albums/{id}/songs`             | Songs by album                            |
| `GET`    | `/v1/home`                          | Curated home sections                     |
| `GET`    | `/v1/playlists`                     | List playlists                            |
| `POST`   | `/v1/playlists`                     | Create playlist                           |
| `PATCH`  | `/v1/playlists/{id}`                | Rename playlist                           |
| `DELETE` | `/v1/playlists/{id}`                | Delete playlist                           |
| `POST`   | `/v1/playlists/{id}/songs`          | Add song to playlist                      |
| `DELETE` | `/v1/playlists/{id}/songs/{songId}` | Remove song                               |
| `PUT`    | `/v1/playlists/{id}/songs`          | Replace ordered list                      |
| `GET`    | `/v1/favorites`                     | List favorites                            |
| `PUT`    | `/v1/favorites/{songId}`            | Like                                      |
| `DELETE` | `/v1/favorites/{songId}`            | Unlike                                    |

---

## 10. Authentication & Security

### 10.1 Auth flow

1. Android client signs in with Google via Credential Manager (existing flow, unchanged).
2. Client receives a Google ID token.
3. Client sends it to `POST /v1/auth/google`.
4. Backend verifies ID token signature against Google JWKS, extracts `sub` + `email`, finds-or-creates a `users` row.
5. Backend mints `accessToken` (JWT, 15 min) + `refreshToken` (opaque, 30 days, sha256-hashed in DB).
6. Client stores both in `EncryptedSharedPreferences`.
7. Client sends `Authorization: Bearer <accessToken>` on every API call.
8. On 401, client calls `POST /v1/auth/refresh` to get a fresh pair, retries the original request.

### 10.2 Drive auth flow

1. Client opens an in-app web view (or system browser) at `https://accounts.google.com/o/oauth2/auth?...`.
2. After consent, Google redirects to a backend callback URL with an authorization code.
3. Backend exchanges the code for `access_token` + `refresh_token`, encrypts both with the `DRIVE_ENCRYPTION_KEY` env var (AES-GCM), stores in `drive_credentials`.
4. From here on, the backend is the only thing that ever talks to Drive on the user's behalf.

### 10.3 Encryption at rest

- `drive_credentials.encrypted_access` / `encrypted_refresh`: AES-GCM with a key from `DRIVE_ENCRYPTION_KEY` env var.
- JWTs signed with `JWT_SIGNING_KEY` env var (HS256 acceptable for single-instance deploys).
- Database backups: encrypted at the storage layer (provider-managed).

### 10.4 HTTPS

- TLS terminated at a reverse proxy (Caddy / nginx) or at the Fly.io / Railway edge.
- HSTS enabled.

---

## 11. Deployment & Operations

### 11.1 Topology (single-user deployment)

```
[Cloudflare DNS] → [Fly.io app] → [Postgres (Fly Postgres)] + [Cloudflare R2 (album art)]
                                        │
                                        └─→ [Drive API]
```

A single Fly.io machine (shared 1 vCPU, 256 MB RAM) handles both API and background sync. Postgres on Fly's managed offering. Album art on R2 (cheap egress).

### 11.2 Configuration (env vars)

| Var                              | Purpose                                |
| -------------------------------- | -------------------------------------- |
| `DATABASE_URL`                   | Postgres connection string             |
| `JWT_SIGNING_KEY`                | HMAC key for JWTs                      |
| `DRIVE_ENCRYPTION_KEY`           | AES-GCM key for Drive token encryption |
| `GOOGLE_CLIENT_ID`               | OAuth client id                        |
| `GOOGLE_CLIENT_SECRET`           | OAuth client secret                    |
| `GOOGLE_REDIRECT_URI`            | OAuth callback URL                     |
| `OBJECT_STORAGE_ENDPOINT`        | S3-compatible endpoint                 |
| `OBJECT_STORAGE_BUCKET`          | Bucket name                            |
| `OBJECT_STORAGE_KEY` / `_SECRET` | Bucket credentials                     |
| `SYNC_INTERVAL_HOURS`            | Default 6                              |
| `LOG_LEVEL`                      | `info` / `debug`                       |

### 11.3 Migrations

Flyway-style numbered SQL migrations under `backend/migrations/`. Run automatically on boot.

### 11.4 Observability

- Structured JSON logs to stdout (Fly.io captures them).
- `/healthz` for liveness, `/readyz` for readiness (DB ping).
- Prometheus `/metrics` endpoint exposing request rate, p95 latency, sync job queue depth.

### 11.5 Backup & Recovery

- Postgres: provider snapshot daily, retained 14 days.
- Album art: R2 versioning enabled.
- Refresh tokens are hashed; if the DB is dumped, replays are not directly possible.

---

## 12. Android Client Rewrite

### 12.1 What disappears from the client

- ❌ `data/drive/*` (entire package — sync moves to backend)
- ❌ `data/repository/DriveLibraryStore.kt`
- ❌ `data/repository/DriveTokenStore.kt`
- ❌ `core/DriveAuthSessionStore.kt`
- ❌ `app/ui/settings/GoogleDriveAuthManager.kt` (replaced by backend OAuth)
- ❌ `data/repository/DefaultMusicRepository.kt`'s sync logic
- ❌ Any in-process token refresh logic

### 12.2 What appears

- ➕ `data/network/SpotifishApi.kt` — Retrofit/Ktor client interface
- ➕ `data/network/AuthInterceptor.kt` — attaches `Authorization` header, handles 401 → refresh → retry
- ➕ `data/network/dto/*.kt` — request/response DTOs (or reuse types from a `shared` Kotlin module — see §13)
- ➕ `data/repository/RemoteMusicRepository.kt` — implements `MusicRepository` against the API
- ➕ `data/repository/RemotePlaylistRepository.kt`
- ➕ `data/repository/RemoteFavoritesRepository.kt`
- ➕ `data/auth/SessionStore.kt` — `EncryptedSharedPreferences` for our JWT pair
- ➕ `data/auth/AuthRepository.kt` — sign-in / refresh / sign-out
- ➕ `app/ui/auth/SignInScreen.kt` — first-run sign-in UX

### 12.3 What stays

- ✅ All Compose UI in `app/ui/` — operates against the same `MusicRepository` interface
- ✅ ViewModels — unchanged contracts
- ✅ `MediaStoreLocalMusicDataSource` — local-only library still works (per [§6 Option B](#6-what-happens-to-local-music-scanning))
- ✅ `Media3PlaybackController` and `PlaybackService` — but the data source factory drops the bearer-token logic; stream URLs from the backend are pre-signed
- ✅ Domain models (`Song`, `Playlist`, …) — possibly moved to a shared module
- ✅ Use cases — they still depend on the same interfaces

### 12.4 Streaming integration

The Media3 `ResolvingDataSource.Factory` becomes much simpler:

```kotlin
ResolvingDataSource.Factory(httpFactory) { dataSpec ->
    if (dataSpec.uri.scheme == "spotifish") {
        // "spotifish:song:<id>" → resolve to a freshly signed stream URL
        val resolvedUri = streamUrlResolver.resolve(songId)
        dataSpec.buildUpon().setUri(resolvedUri).build()
    } else {
        dataSpec
    }
}
```

`StreamUrlResolver` calls `GET /v1/songs/{id}/stream` synchronously (acceptable on the player thread for a sub-300ms call) and caches the result for the URL's TTL.

### 12.5 Repository contract migration

| Current                                                   | After                                                                                |
| --------------------------------------------------------- | ------------------------------------------------------------------------------------ |
| `DefaultMusicRepository` (mixes local + Drive sync state) | `CompositeMusicRepository` (merges `LocalMusicRepository` + `RemoteMusicRepository`) |
| `DefaultPlaylistRepository` (JSON files)                  | `RemotePlaylistRepository` (HTTP)                                                    |
| `DefaultFavoritesRepository` (JSON files)                 | `RemoteFavoritesRepository` (HTTP)                                                   |
| `DefaultSettingsRepository` (DataStore)                   | `RemoteSettingsRepository` (HTTP) — local DataStore retained for theme only          |

The domain `MusicRepository` interface stays the same, so ViewModels and use cases don't change.

### 12.6 Offline behavior

- The client caches the last successful library response (paginated, rolled into a single Room table or a simple JSON file).
- If the network is unreachable, repos return cached data and the UI shows an "offline" badge.
- Stream URLs cannot be cached → playback of Drive songs requires connectivity. Local songs continue to work offline via `LocalMusicRepository`.

---

## 13. API Contract Source of Truth

Because the backend is Go and the client is Kotlin, we can't share types via a Gradle module. Instead, the **API contract is owned by this PRD plus an OpenAPI 3 spec** that lives in the backend repo (`backend/openapi.yaml`). Both sides generate code from it:

- Backend: generated handlers + request/response structs via `oapi-codegen`
- Client (this repo): hand-written Kotlin DTOs that mirror the OpenAPI shapes, validated by integration tests against a running backend

Any change to a request or response shape must update the OpenAPI spec **and** this PRD's [§9 API Contract](#9-api-contract-rest-json) section in the same PR.

---

## 14. Decisions (locked in)

| Decision                    | Choice                         | Notes                                                                                                                                                                      |
| --------------------------- | ------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Backend tech stack**      | **Go + Gin**                   | Single static binary, low memory, fast cold start                                                                                                                          |
| **Repo layout**             | **Backend in a separate repo** | This repo only contains the rewritten Android client                                                                                                                       |
| **User model**              | **Multi-user from day one**    | Real Google sign-in, JWT pair, per-user scoping in the DB                                                                                                                  |
| **Database**                | **PostgreSQL 16**              | tsvector full-text search, gen_random_uuid(), trivial in Docker                                                                                                            |
| **Album art**               | **Local disk on backend host** | `/var/lib/spotifish/art/<uuid>.img`, migrate to S3 later                                                                                                                   |
| **Streaming model**         | **Backend proxies bytes**      | `GET /v1/songs/{id}/stream` opens Drive download, pipes bytes with `Range` support                                                                                         |
| **Local music scanning**    | **Stays on the client**        | `MediaStoreLocalMusicDataSource` unchanged, results merged with remote in `CompositeMusicRepository`                                                                       |
| **Existing-data migration** | **Upload on first sign-in**    | Client posts existing playlists + favorites to the backend on first launch; the local Drive library cache is discarded                                                     |
| **Sync trigger**            | **Cron + on-demand**           | Default cron interval 6 hours, plus a manual `POST /v1/sync/run` button                                                                                                    |
| **OAuth client model**      | **Single Google client**       | Android client uses Identity Services with `requestServerAuthCode(serverClientId)`, posts the auth code to `/v1/auth/google`, backend exchanges for both ID + Drive tokens |

---

## 15. Phased Rollout Plan

| Phase | Scope                                                                         | Done when                                                                                |
| ----- | ----------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- | -------------------- |
| **0** | This PRD approved + open questions answered                                   | You sign off on §14                                                                      |
| **1** | Backend skeleton: Ktor app, Postgres schema, auth endpoints, `/me`            | `curl /v1/me` returns user JSON after Google sign-in                                     |
| **2** | Drive integration: connect, list folders, sync worker, songs API              | Sync produces > 0 rows in `songs` table for your Drive folder                            |
| **3** | Library + search + home + playlists + favorites endpoints                     | Postman collection of all endpoints passes                                               |
| **4** | Streaming endpoint                                                            | `curl -L /v1/songs/{id}/stream                                                           | mpv -` plays a track |
| **5** | Android client refactor: Retrofit DI, `RemoteMusicRepository`, sign-in screen | App talks only to backend for Drive content; local scanning still works                  |
| **6** | Decommission client-side Drive code, ship                                     | All code listed in [§12.1](#121-what-disappears-from-the-client) deleted, all tests pass |

Each phase is independently shippable in the sense that the previous phase's behavior doesn't regress.

---

## 16. Success Metrics

- **First sync time** for a 500-song Drive folder ≤ 5 minutes (server-side, vs. ~20 minutes on the current device-side flow).
- **Re-sync time** when nothing has changed ≤ 5 seconds.
- **Cold-start time to first frame** of the home screen ≤ 1.5 seconds (no client-side sync to wait on).
- **Library list payload** ≤ 50 KB for the first page of 50 songs.
- **Backend monthly cost** ≤ $10 for a single user.
