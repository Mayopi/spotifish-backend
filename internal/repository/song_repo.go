package repository

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spotifish/backend/internal/model"
)

// SongRepository handles database operations for songs.
type SongRepository struct {
	pool *pgxpool.Pool
}

// NewSongRepository creates a new SongRepository.
func NewSongRepository(pool *pgxpool.Pool) *SongRepository {
	return &SongRepository{pool: pool}
}

// Upsert inserts or updates a song by (user_id, source, source_file_id).
func (r *SongRepository) Upsert(ctx context.Context, song *model.Song) (*model.Song, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO songs (user_id, source, source_file_id, title, artist, album, duration_ms, mime_type, album_art_object_key, drive_modified_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 ON CONFLICT (user_id, source, source_file_id) DO UPDATE SET
		     title = EXCLUDED.title,
		     artist = EXCLUDED.artist,
		     album = EXCLUDED.album,
		     duration_ms = EXCLUDED.duration_ms,
		     mime_type = EXCLUDED.mime_type,
		     album_art_object_key = COALESCE(EXCLUDED.album_art_object_key, songs.album_art_object_key),
		     drive_modified_at = EXCLUDED.drive_modified_at
		 RETURNING id, added_at`,
		song.UserID, song.Source, song.SourceFileID, song.Title, song.Artist, song.Album,
		song.DurationMs, song.MimeType, song.AlbumArtObjectKey, song.DriveModifiedAt,
	).Scan(&song.ID, &song.AddedAt)

	if err != nil {
		return nil, fmt.Errorf("upsert song: %w", err)
	}
	return song, nil
}

// GetByID retrieves a song by ID, scoped to a user.
func (r *SongRepository) GetByID(ctx context.Context, userID, songID string) (*model.Song, error) {
	var s model.Song
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, source, source_file_id, title, artist, album, duration_ms, mime_type,
		        album_art_object_key, drive_modified_at, added_at
		 FROM songs WHERE id = $1 AND user_id = $2`, songID, userID,
	).Scan(&s.ID, &s.UserID, &s.Source, &s.SourceFileID, &s.Title, &s.Artist, &s.Album,
		&s.DurationMs, &s.MimeType, &s.AlbumArtObjectKey, &s.DriveModifiedAt, &s.AddedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get song by id: %w", err)
	}
	return &s, nil
}

// GetBySourceFileID retrieves a song by its source file ID.
func (r *SongRepository) GetBySourceFileID(ctx context.Context, userID, source, sourceFileID string) (*model.Song, error) {
	var s model.Song
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, source, source_file_id, title, artist, album, duration_ms, mime_type,
		        album_art_object_key, drive_modified_at, added_at
		 FROM songs WHERE user_id = $1 AND source = $2 AND source_file_id = $3`,
		userID, source, sourceFileID,
	).Scan(&s.ID, &s.UserID, &s.Source, &s.SourceFileID, &s.Title, &s.Artist, &s.Album,
		&s.DurationMs, &s.MimeType, &s.AlbumArtObjectKey, &s.DriveModifiedAt, &s.AddedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get song by source file id: %w", err)
	}
	return &s, nil
}

// List returns a paginated list of songs for a user.
func (r *SongRepository) List(ctx context.Context, userID, cursor string, limit int, sortBy, sortDir string) ([]*model.Song, string, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	// Validate sort params
	validSorts := map[string]string{
		"title": "lower(title)", "artist": "lower(artist)", "album": "lower(album)",
		"added_at": "added_at", "duration_ms": "duration_ms",
	}
	orderCol, ok := validSorts[sortBy]
	if !ok {
		orderCol = "lower(title)"
	}
	if sortDir != "desc" {
		sortDir = "asc"
	}

	// Build query with cursor-based pagination using ID as tiebreaker
	query := fmt.Sprintf(
		`SELECT id, user_id, source, source_file_id, title, artist, album, duration_ms, mime_type,
		        album_art_object_key, drive_modified_at, added_at
		 FROM songs WHERE user_id = $1`)

	args := []interface{}{userID}
	if cursor != "" {
		// Cursor is the last song's ID
		query += ` AND id > $2`
		args = append(args, cursor)
	}
	query += fmt.Sprintf(` ORDER BY %s %s, id ASC LIMIT $%d`, orderCol, sortDir, len(args)+1)
	args = append(args, limit+1) // fetch one extra to determine nextCursor

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, "", fmt.Errorf("list songs: %w", err)
	}
	defer rows.Close()

	var songs []*model.Song
	for rows.Next() {
		var s model.Song
		if err := rows.Scan(&s.ID, &s.UserID, &s.Source, &s.SourceFileID, &s.Title, &s.Artist,
			&s.Album, &s.DurationMs, &s.MimeType, &s.AlbumArtObjectKey, &s.DriveModifiedAt, &s.AddedAt); err != nil {
			return nil, "", fmt.Errorf("scan song: %w", err)
		}
		songs = append(songs, &s)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("iterate songs: %w", err)
	}

	var nextCursor string
	if len(songs) > limit {
		nextCursor = songs[limit-1].ID
		songs = songs[:limit]
	}

	return songs, nextCursor, nil
}

// Search performs a full-text search across title, artist, and album.
func (r *SongRepository) Search(ctx context.Context, userID, query string, limit int) ([]*model.Song, error) {
	if limit <= 0 || limit > 100 {
		limit = 100
	}

	// Convert the user's query into a tsquery — split on spaces, join with &
	words := strings.Fields(query)
	for i, w := range words {
		words[i] = w + ":*" // prefix matching
	}
	tsQuery := strings.Join(words, " & ")

	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, source, source_file_id, title, artist, album, duration_ms, mime_type,
		        album_art_object_key, drive_modified_at, added_at
		 FROM songs
		 WHERE user_id = $1
		   AND to_tsvector('simple', title || ' ' || artist || ' ' || album) @@ to_tsquery('simple', $2)
		 ORDER BY ts_rank(to_tsvector('simple', title || ' ' || artist || ' ' || album), to_tsquery('simple', $2)) DESC
		 LIMIT $3`,
		userID, tsQuery, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("search songs: %w", err)
	}
	defer rows.Close()

	var songs []*model.Song
	for rows.Next() {
		var s model.Song
		if err := rows.Scan(&s.ID, &s.UserID, &s.Source, &s.SourceFileID, &s.Title, &s.Artist,
			&s.Album, &s.DurationMs, &s.MimeType, &s.AlbumArtObjectKey, &s.DriveModifiedAt, &s.AddedAt); err != nil {
			return nil, fmt.Errorf("scan song: %w", err)
		}
		songs = append(songs, &s)
	}
	return songs, rows.Err()
}

// GetArtists returns artist groups with song count and sample album art.
func (r *SongRepository) GetArtists(ctx context.Context, userID string) ([]model.ArtistGroup, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT artist, COUNT(*) as song_count,
		        (SELECT album_art_object_key FROM songs s2 WHERE s2.user_id = $1 AND s2.artist = songs.artist AND s2.album_art_object_key IS NOT NULL LIMIT 1) as sample_art
		 FROM songs WHERE user_id = $1
		 GROUP BY artist
		 ORDER BY lower(artist)`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("get artists: %w", err)
	}
	defer rows.Close()

	var artists []model.ArtistGroup
	for rows.Next() {
		var a model.ArtistGroup
		var sampleArt *string
		if err := rows.Scan(&a.Name, &a.SongCount, &sampleArt); err != nil {
			return nil, fmt.Errorf("scan artist: %w", err)
		}
		a.ID = base64.URLEncoding.EncodeToString([]byte(a.Name))
		if sampleArt != nil {
			a.ArtURL = *sampleArt
		}
		artists = append(artists, a)
	}
	return artists, rows.Err()
}

// GetAlbums returns album groups with song count and sample album art.
func (r *SongRepository) GetAlbums(ctx context.Context, userID string) ([]model.AlbumGroup, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT album, artist, COUNT(*) as song_count,
		        (SELECT album_art_object_key FROM songs s2 WHERE s2.user_id = $1 AND s2.album = songs.album AND s2.artist = songs.artist AND s2.album_art_object_key IS NOT NULL LIMIT 1) as sample_art
		 FROM songs WHERE user_id = $1
		 GROUP BY album, artist
		 ORDER BY lower(album)`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("get albums: %w", err)
	}
	defer rows.Close()

	var albums []model.AlbumGroup
	for rows.Next() {
		var a model.AlbumGroup
		var sampleArt *string
		if err := rows.Scan(&a.Name, &a.Artist, &a.SongCount, &sampleArt); err != nil {
			return nil, fmt.Errorf("scan album: %w", err)
		}
		a.ID = base64.URLEncoding.EncodeToString([]byte(a.Name + "|" + a.Artist))
		if sampleArt != nil {
			a.ArtURL = *sampleArt
		}
		albums = append(albums, a)
	}
	return albums, rows.Err()
}

// ListByArtist returns all songs by a specific artist.
func (r *SongRepository) ListByArtist(ctx context.Context, userID, artistName string) ([]*model.Song, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, source, source_file_id, title, artist, album, duration_ms, mime_type,
		        album_art_object_key, drive_modified_at, added_at
		 FROM songs WHERE user_id = $1 AND artist = $2
		 ORDER BY lower(album), added_at`, userID, artistName,
	)
	if err != nil {
		return nil, fmt.Errorf("list by artist: %w", err)
	}
	defer rows.Close()
	return scanSongs(rows)
}

// ListByAlbum returns all songs in a specific album by a specific artist.
func (r *SongRepository) ListByAlbum(ctx context.Context, userID, albumName, artistName string) ([]*model.Song, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, source, source_file_id, title, artist, album, duration_ms, mime_type,
		        album_art_object_key, drive_modified_at, added_at
		 FROM songs WHERE user_id = $1 AND album = $2 AND artist = $3
		 ORDER BY added_at`, userID, albumName, artistName,
	)
	if err != nil {
		return nil, fmt.Errorf("list by album: %w", err)
	}
	defer rows.Close()
	return scanSongs(rows)
}

// RecentlyAdded returns the most recently added songs.
func (r *SongRepository) RecentlyAdded(ctx context.Context, userID string, limit int) ([]*model.Song, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, source, source_file_id, title, artist, album, duration_ms, mime_type,
		        album_art_object_key, drive_modified_at, added_at
		 FROM songs WHERE user_id = $1
		 ORDER BY added_at DESC LIMIT $2`, userID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("recently added: %w", err)
	}
	defer rows.Close()
	return scanSongs(rows)
}

// CountByUserID returns the total song count for a user.
func (r *SongRepository) CountByUserID(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM songs WHERE user_id = $1`, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count songs: %w", err)
	}
	return count, nil
}

// GetDriveFileIDs returns all Drive source file IDs for a user.
func (r *SongRepository) GetDriveFileIDs(ctx context.Context, userID string) (map[string]bool, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT source_file_id FROM songs WHERE user_id = $1 AND source = 'drive'`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("get drive file ids: %w", err)
	}
	defer rows.Close()

	ids := make(map[string]bool)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan file id: %w", err)
		}
		ids[id] = true
	}
	return ids, rows.Err()
}

// DeleteBySourceFileIDs deletes songs by their source file IDs.
func (r *SongRepository) DeleteBySourceFileIDs(ctx context.Context, userID string, fileIDs []string) (int, error) {
	if len(fileIDs) == 0 {
		return 0, nil
	}
	ct, err := r.pool.Exec(ctx,
		`DELETE FROM songs WHERE user_id = $1 AND source = 'drive' AND source_file_id = ANY($2)`,
		userID, fileIDs,
	)
	if err != nil {
		return 0, fmt.Errorf("delete songs: %w", err)
	}
	return int(ct.RowsAffected()), nil
}

// DeleteAllDriveSongs deletes all Drive-sourced songs for a user.
func (r *SongRepository) DeleteAllDriveSongs(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM songs WHERE user_id = $1 AND source = 'drive'`, userID,
	)
	if err != nil {
		return fmt.Errorf("delete all drive songs: %w", err)
	}
	return nil
}

// scanSongs is a helper that scans rows into a slice of Song pointers.
func scanSongs(rows pgx.Rows) ([]*model.Song, error) {
	var songs []*model.Song
	for rows.Next() {
		var s model.Song
		if err := rows.Scan(&s.ID, &s.UserID, &s.Source, &s.SourceFileID, &s.Title, &s.Artist,
			&s.Album, &s.DurationMs, &s.MimeType, &s.AlbumArtObjectKey, &s.DriveModifiedAt, &s.AddedAt); err != nil {
			return nil, fmt.Errorf("scan song: %w", err)
		}
		songs = append(songs, &s)
	}
	return songs, rows.Err()
}
