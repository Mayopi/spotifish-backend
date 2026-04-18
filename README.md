# 🎧 Spotifish Backend

A self-hosted Go backend service for the Spotifish Android music app. Handles Google Drive library sync, music streaming, playlists, favorites, and cross-device library state — so the Android client only needs to talk HTTP.

## Tech Stack

| Component | Technology |
|-----------|------------|
| API | Go + Gin |
| Database | PostgreSQL 16 |
| Auth | Google OAuth 2.0 + JWT (HS256) |
| Drive Integration | Google Drive API v3 |
| Metadata Extraction | `dhowden/tag` (ID3, FLAC, Vorbis) |
| Album Art | Local disk storage |
| Migrations | `golang-migrate` |
| Scheduler | `robfig/cron` |
| Logging | `zerolog` (structured JSON) |

---

## Prerequisites

- Docker + Docker Compose
- A [Google Cloud project](https://console.cloud.google.com/) with:
  - **Google Drive API** enabled
  - An **OAuth 2.0 Web Client** credential created

---

## Getting Started

### 1. Clone & configure

```bash
git clone https://github.com/spotifish/backend.git
cd spotifish-backend
cp .env.example .env
```

Edit `.env` and fill in your values:

```env
JWT_SIGNING_KEY=your-random-64-char-secret
GOOGLE_CLIENT_ID=your-client-id.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=your-client-secret
GOOGLE_REDIRECT_URI=http://localhost:8080/v1/drive/callback
DRIVE_ENCRYPTION_KEY=64-hex-chars-random-32-byte-key-here
```

> **Generating keys:**
> ```bash
> # JWT signing key (any long random string)
> openssl rand -hex 32
>
> # Drive encryption key (must be exactly 64 hex chars = 32 bytes)
> openssl rand -hex 32
> ```

### 2. Start with Docker Compose

```bash
docker compose up -d
```

This starts:
- `app` — the Go backend on port `8080`
- `postgres` — PostgreSQL 16 (migrations run automatically on boot)

### 3. Verify it's running

```bash
curl http://localhost:8080/healthz   # → {"status":"ok"}
curl http://localhost:8080/readyz    # → {"status":"ready"}
```

---

## Google OAuth Setup

You need a single **Web application** OAuth client from [Google Cloud Console](https://console.cloud.google.com/apis/credentials):

1. Go to **APIs & Services → Credentials → Create Credentials → OAuth client ID**
2. Choose **Web application**
3. Add your redirect URI under **Authorized redirect URIs** (e.g. `http://your-server/v1/drive/callback`)
4. Copy **Client ID** → `GOOGLE_CLIENT_ID`
5. Copy **Client secret** → `GOOGLE_CLIENT_SECRET`

> **Note for Android:** In your Android app set `requestServerAuthCode(serverClientId)` using this same **Web** client ID. Android-type OAuth clients don't have a secret, so they can't be used here.

---

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | ✅ | — | PostgreSQL connection string |
| `JWT_SIGNING_KEY` | ✅ | — | HMAC key for signing JWTs |
| `GOOGLE_CLIENT_ID` | ✅ | — | OAuth Web client ID |
| `GOOGLE_CLIENT_SECRET` | — | — | OAuth Web client secret |
| `GOOGLE_REDIRECT_URI` | — | — | OAuth redirect URL |
| `DRIVE_ENCRYPTION_KEY` | — | — | 64 hex chars (32 bytes) for AES-256-GCM |
| `ALBUM_ART_PATH` | — | `./art` | Directory to store album art |
| `SYNC_INTERVAL_HOURS` | — | `6` | How often to auto-sync Drive libraries |
| `RATE_LIMIT_PER_MIN` | — | `200` | Max requests per user per minute |
| `PORT` | — | `8080` | HTTP server port |
| `LOG_LEVEL` | — | `info` | `debug` / `info` / `warn` / `error` |

---

## API Reference

Base URL: `https://<host>/v1`

All endpoints except `/auth/*` require: `Authorization: Bearer <accessToken>`

### Authentication

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/v1/auth/google` | Sign in with Google ID token, returns JWT pair |
| `POST` | `/v1/auth/refresh` | Rotate refresh token, returns new JWT pair |
| `POST` | `/v1/auth/sign-out` | Revoke refresh token |

**Sign in request:**
```json
POST /v1/auth/google
Content-Type: application/json

{ "idToken": "<Google ID token from Android Credential Manager>" }
```

**Response:**
```json
{
  "accessToken": "eyJ...",
  "refreshToken": "abc123...",
  "user": { "id": "...", "email": "...", "displayName": "..." }
}
```

### User

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/v1/me` | Current user profile |
| `GET` | `/v1/me/settings` | User settings |
| `PATCH` | `/v1/me/settings` | Update settings |
| `GET` | `/v1/me/stats` | User library + playback statistics |

### Google Drive

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/v1/drive/connect` | Connect Drive (exchange auth code) |
| `GET` | `/v1/drive/folders?parentId={id}` | List subfolders |
| `POST` | `/v1/drive/connection` | Set active sync folder |
| `DELETE` | `/v1/drive/connection` | Disconnect Drive |

### Library Sync

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/v1/sync/run` | Trigger a manual sync |
| `GET` | `/v1/sync/status` | Check sync job status |

### Songs

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/v1/songs` | List songs (paginated, sortable) |
| `GET` | `/v1/songs/:id` | Song detail |
| `GET` | `/v1/songs/search?q=` | Full-text search (title, artist, album) |
| `GET` | `/v1/songs/:id/stream` | Stream audio (byte-proxy, Range supported) |

**List query params:** `?cursor=&limit=50&sort=title&dir=asc`

### Artists & Albums

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/v1/artists` | List artists with song counts |
| `GET` | `/v1/artists/:id/songs` | Songs by artist |
| `GET` | `/v1/albums` | List albums with song counts |
| `GET` | `/v1/albums/:id/songs` | Songs by album |
| `GET` | `/v1/home` | Curated home screen sections |

### Playlists

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/v1/playlists` | List playlists |
| `POST` | `/v1/playlists` | Create playlist `{"name":"..."}` |
| `PATCH` | `/v1/playlists/:id` | Rename playlist |
| `DELETE` | `/v1/playlists/:id` | Delete playlist |
| `POST` | `/v1/playlists/:id/songs` | Add song `{"songId":"..."}` |
| `DELETE` | `/v1/playlists/:id/songs/:songId` | Remove song |
| `PUT` | `/v1/playlists/:id/songs` | Replace full ordered list `{"songIds":[...]}` |

### Favorites

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/v1/favorites` | List liked songs |
| `PUT` | `/v1/favorites/:songId` | Like a song |
| `DELETE` | `/v1/favorites/:songId` | Unlike a song |

### Playback Events

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/v1/playback/events` | Record playback event |
| `GET` | `/v1/playback/recent?limit=20` | List recently played songs |

```json
{ "songId": "...", "eventType": "started|completed|skipped", "positionMs": 0 }
```

### Infrastructure

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/healthz` | Liveness probe |
| `GET` | `/readyz` | Readiness probe (checks DB) |
| `GET` | `/v1/art/:key` | Serve album art |

---

## Error Format

All errors follow a consistent shape:

```json
{
  "error": {
    "code": "token_expired",
    "message": "invalid or expired token"
  }
}
```

---

## Auth Flow (for Android client)

```
1. Android → Google Credential Manager → Google ID Token
2. Android → POST /v1/auth/google { idToken }
3. Backend verifies token, finds/creates user
4. Backend → { accessToken (15 min), refreshToken (30 days) }
5. Android attaches: Authorization: Bearer <accessToken> on every request
6. On 401 → Android calls POST /v1/auth/refresh to rotate tokens
```

## Drive Sync Flow

```
1. Android opens browser → Google OAuth consent screen (Drive readonly)
2. Google redirects to /v1/drive/callback with auth code
3. POST /v1/drive/connect { authCode } → backend exchanges & stores tokens (AES encrypted)
4. POST /v1/drive/connection { folderId, folderName } → set which folder to sync
5. POST /v1/sync/run → triggers immediate sync
6. Sync worker auto-runs every SYNC_INTERVAL_HOURS
```

---

## Development

```bash
# Run locally (needs PostgreSQL running)
make run

# Build binary
make build

# Run tests
make test

# Docker
make docker-up
make docker-down
```

---

## Database Schema

8 migration files under `migrations/` creating these tables:

`users` · `auth_refresh_tokens` · `drive_credentials` · `drive_folders` · `songs` · `playlists` · `playlist_songs` · `favorites` · `sync_jobs` · `user_settings` · `playback_events`

---

## Project Structure

```
spotifish-backend/
├── cmd/server/main.go       # Entry point
├── internal/
│   ├── config/              # Env var configuration
│   ├── crypto/              # AES-256-GCM encryption
│   ├── database/            # DB pool + migrations
│   ├── middleware/          # Auth, CORS, Logger, Rate limiter
│   ├── model/               # Domain models
│   ├── repository/          # Database access layer
│   ├── service/             # Business logic layer
│   ├── handler/             # HTTP handlers
│   ├── worker/              # Background sync worker
│   └── server/              # Router + DI wiring
├── migrations/              # SQL migration files
├── Dockerfile
├── docker-compose.yml
└── .env.example
```
