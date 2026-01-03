package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"

	"metadata-api/internal/models"

	_ "modernc.org/sqlite"
)

type DB struct {
	main       *sql.DB
	trackFiles *sql.DB
}

func Open(dbPath string) (*DB, error) {
	main, err := sql.Open("sqlite", dbPath+"?mode=ro&_journal_mode=off")
	if err != nil {
		return nil, fmt.Errorf("open main db: %w", err)
	}
	main.SetMaxOpenConns(4)

	dir := filepath.Dir(dbPath)
	trackFilesPath := filepath.Join(dir, "spotify_clean_track_files.sqlite3")
	trackFiles, err := sql.Open("sqlite", trackFilesPath+"?mode=ro&_journal_mode=off")
	if err != nil {
		main.Close()
		return nil, fmt.Errorf("open track_files db: %w", err)
	}
	trackFiles.SetMaxOpenConns(4)

	return &DB{main: main, trackFiles: trackFiles}, nil
}

func (d *DB) Close() error {
	d.trackFiles.Close()
	return d.main.Close()
}

func (d *DB) LookupISRC(ctx context.Context, isrc string) ([]models.Track, error) {
	rows, err := d.main.QueryContext(ctx, `
		SELECT t.id, t.name, t.external_id_isrc, t.duration_ms, t.explicit,
		       t.track_number, t.disc_number, t.popularity, t.preview_url,
		       a.id, a.name, a.album_type, a.label, a.release_date, a.release_date_precision,
		       a.external_id_upc, a.total_tracks, a.copyright_c, a.copyright_p, a.rowid
		FROM tracks t
		JOIN albums a ON t.album_rowid = a.rowid
		WHERE t.external_id_isrc = ?
		ORDER BY t.popularity DESC
	`, isrc)
	if err != nil {
		return nil, fmt.Errorf("query isrc: %w", err)
	}
	defer rows.Close()

	var tracks []models.Track
	for rows.Next() {
		t, err := d.scanTrackWithAlbum(ctx, rows)
		if err != nil {
			return nil, err
		}
		tracks = append(tracks, *t)
	}
	return tracks, rows.Err()
}

func (d *DB) LookupTrack(ctx context.Context, id string) (*models.Track, error) {
	rows, err := d.main.QueryContext(ctx, `
		SELECT t.id, t.name, t.external_id_isrc, t.duration_ms, t.explicit,
		       t.track_number, t.disc_number, t.popularity, t.preview_url,
		       a.id, a.name, a.album_type, a.label, a.release_date, a.release_date_precision,
		       a.external_id_upc, a.total_tracks, a.copyright_c, a.copyright_p, a.rowid
		FROM tracks t
		JOIN albums a ON t.album_rowid = a.rowid
		WHERE t.id = ?
	`, id)
	if err != nil {
		return nil, fmt.Errorf("query track: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil
	}
	return d.scanTrackWithAlbum(ctx, rows)
}

func (d *DB) scanTrackWithAlbum(ctx context.Context, rows *sql.Rows) (*models.Track, error) {
	var t models.Track
	var alb models.Album
	var isrcNull, upcNull, copyCNull, copyPNull, previewNull sql.NullString
	var albumRowID int64

	err := rows.Scan(
		&t.ID, &t.Name, &isrcNull, &t.DurationMs, &t.Explicit,
		&t.TrackNum, &t.DiscNum, &t.Popularity, &previewNull,
		&alb.ID, &alb.Name, &alb.Type, &alb.Label, &alb.ReleaseDate, &alb.ReleaseDatePrecision,
		&upcNull, &alb.TotalTracks, &copyCNull, &copyPNull, &albumRowID,
	)
	if err != nil {
		return nil, fmt.Errorf("scan track: %w", err)
	}

	t.ISRC = isrcNull.String
	t.PreviewURL = previewNull.String
	alb.UPC = upcNull.String
	alb.CopyrightC = copyCNull.String
	alb.CopyrightP = copyPNull.String

	albumImages, err := d.getAlbumImages(ctx, albumRowID)
	if err != nil {
		slog.Error("get album images", "err", err, "rowid", albumRowID)
	}
	alb.Images = albumImages

	albumArtists, err := d.getAlbumArtists(ctx, albumRowID)
	if err != nil {
		slog.Error("get album artists", "err", err, "rowid", albumRowID)
	}
	alb.Artists = albumArtists

	t.Album = &alb

	artists, _ := d.getTrackArtists(ctx, t.ID)
	t.Artists = artists

	d.enrichTrackFromFiles(ctx, &t)

	return &t, nil
}

func (d *DB) enrichTrackFromFiles(ctx context.Context, t *models.Track) {
	row := d.trackFiles.QueryRowContext(ctx, `
		SELECT has_lyrics, original_title, version_title, language_of_performance, artist_roles
		FROM track_files WHERE track_id = ?
	`, t.ID)

	var hasLyrics sql.NullInt64
	var origTitle, versionTitle, langJSON, rolesJSON sql.NullString

	if err := row.Scan(&hasLyrics, &origTitle, &versionTitle, &langJSON, &rolesJSON); err != nil {
		return
	}

	if hasLyrics.Valid {
		val := hasLyrics.Int64 == 1
		t.HasLyrics = &val
	}
	t.OriginalTitle = origTitle.String
	t.VersionTitle = versionTitle.String

	if langJSON.String != "" {
		json.Unmarshal([]byte(langJSON.String), &t.Languages)
	}
	if rolesJSON.String != "" {
		json.Unmarshal([]byte(rolesJSON.String), &t.ArtistRoles)
	}
}

func (d *DB) LookupArtist(ctx context.Context, id string) (*models.Artist, error) {
	row := d.main.QueryRowContext(ctx, `
		SELECT id, name, followers_total, popularity, rowid FROM artists WHERE id = ?
	`, id)

	var a models.Artist
	var rowid int64
	err := row.Scan(&a.ID, &a.Name, &a.Followers, &a.Popularity, &rowid)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan artist: %w", err)
	}

	a.Genres, _ = d.getArtistGenres(ctx, rowid)
	images, err := d.getArtistImages(ctx, rowid)
	if err != nil {
		slog.Error("get artist images", "err", err, "rowid", rowid)
	}
	a.Images = images

	return &a, nil
}

func (d *DB) LookupAlbum(ctx context.Context, id string) (*models.Album, error) {
	row := d.main.QueryRowContext(ctx, `
		SELECT id, name, album_type, label, release_date, release_date_precision,
		       external_id_upc, total_tracks, copyright_c, copyright_p, rowid
		FROM albums WHERE id = ?
	`, id)

	var a models.Album
	var upcNull, copyCNull, copyPNull sql.NullString
	var rowid int64

	err := row.Scan(&a.ID, &a.Name, &a.Type, &a.Label, &a.ReleaseDate, &a.ReleaseDatePrecision,
		&upcNull, &a.TotalTracks, &copyCNull, &copyPNull, &rowid)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan album: %w", err)
	}

	a.UPC = upcNull.String
	a.CopyrightC = copyCNull.String
	a.CopyrightP = copyPNull.String
	a.Images, _ = d.getAlbumImages(ctx, rowid)
	a.Artists, _ = d.getAlbumArtists(ctx, rowid)

	return &a, nil
}

func (d *DB) GetAlbumTracks(ctx context.Context, albumID string) ([]models.Track, error) {
	rows, err := d.main.QueryContext(ctx, `
		SELECT t.id, t.name, t.external_id_isrc, t.duration_ms, t.explicit,
		       t.track_number, t.disc_number, t.popularity, t.preview_url
		FROM tracks t
		JOIN albums a ON t.album_rowid = a.rowid
		WHERE a.id = ?
		ORDER BY t.disc_number, t.track_number
	`, albumID)
	if err != nil {
		return nil, fmt.Errorf("get album tracks: %w", err)
	}
	defer rows.Close()

	var tracks []models.Track
	for rows.Next() {
		var t models.Track
		var isrcNull, previewNull sql.NullString
		err := rows.Scan(&t.ID, &t.Name, &isrcNull, &t.DurationMs, &t.Explicit,
			&t.TrackNum, &t.DiscNum, &t.Popularity, &previewNull)
		if err != nil {
			return nil, fmt.Errorf("scan track: %w", err)
		}
		t.ISRC = isrcNull.String
		t.PreviewURL = previewNull.String

		artists, _ := d.getTrackArtists(ctx, t.ID)
		t.Artists = artists

		d.enrichTrackFromFiles(ctx, &t)

		tracks = append(tracks, t)
	}
	return tracks, rows.Err()
}

func (d *DB) SearchArtist(ctx context.Context, query string, limit int) ([]models.Artist, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	rows, err := d.main.QueryContext(ctx, `
		SELECT id, name, followers_total, popularity, rowid FROM artists
		WHERE name LIKE ?
		ORDER BY followers_total DESC
		LIMIT ?
	`, "%"+query+"%", limit)
	if err != nil {
		return nil, fmt.Errorf("search artist: %w", err)
	}
	defer rows.Close()

	var artists []models.Artist
	for rows.Next() {
		var a models.Artist
		var rowid int64
		if err := rows.Scan(&a.ID, &a.Name, &a.Followers, &a.Popularity, &rowid); err != nil {
			return nil, fmt.Errorf("scan artist: %w", err)
		}
		a.Genres, _ = d.getArtistGenres(ctx, rowid)
		a.Images, _ = d.getArtistImages(ctx, rowid)
		artists = append(artists, a)
	}
	return artists, rows.Err()
}

func (d *DB) SearchTrack(ctx context.Context, query string, limit int) ([]models.Track, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	rows, err := d.main.QueryContext(ctx, `
		SELECT t.id, t.name, t.external_id_isrc, t.duration_ms, t.explicit,
		       t.track_number, t.disc_number, t.popularity, t.preview_url,
		       a.id, a.name, a.album_type, a.label, a.release_date, a.release_date_precision,
		       a.external_id_upc, a.total_tracks, a.copyright_c, a.copyright_p, a.rowid
		FROM tracks t
		JOIN albums a ON t.album_rowid = a.rowid
		WHERE t.name LIKE ?
		ORDER BY t.popularity DESC
		LIMIT ?
	`, "%"+query+"%", limit)
	if err != nil {
		return nil, fmt.Errorf("search track: %w", err)
	}
	defer rows.Close()

	var tracks []models.Track
	for rows.Next() {
		t, err := d.scanTrackWithAlbum(ctx, rows)
		if err != nil {
			return nil, err
		}
		tracks = append(tracks, *t)
	}
	return tracks, rows.Err()
}

func (d *DB) getTrackArtists(ctx context.Context, trackID string) ([]models.Artist, error) {
	rows, err := d.main.QueryContext(ctx, `
		SELECT a.id, a.name, a.followers_total, a.popularity, a.rowid
		FROM artists a
		JOIN track_artists ta ON a.rowid = ta.artist_rowid
		JOIN tracks t ON ta.track_rowid = t.rowid
		WHERE t.id = ?
	`, trackID)
	if err != nil {
		return nil, fmt.Errorf("get track artists: %w", err)
	}
	defer rows.Close()

	var artists []models.Artist
	for rows.Next() {
		var a models.Artist
		var rowid int64
		if err := rows.Scan(&a.ID, &a.Name, &a.Followers, &a.Popularity, &rowid); err != nil {
			return nil, fmt.Errorf("scan artist: %w", err)
		}
		a.Genres, _ = d.getArtistGenres(ctx, rowid)
		artists = append(artists, a)
	}
	return artists, rows.Err()
}

func (d *DB) getAlbumArtists(ctx context.Context, albumRowID int64) ([]models.Artist, error) {
	rows, err := d.main.QueryContext(ctx, `
		SELECT a.id, a.name, a.followers_total, a.popularity, a.rowid, MIN(aa.index_in_album) as idx
		FROM artists a
		JOIN artist_albums aa ON a.rowid = aa.artist_rowid
		WHERE aa.album_rowid = ? AND aa.index_in_album IS NOT NULL
		GROUP BY a.id
		ORDER BY idx
	`, albumRowID)
	if err != nil {
		return nil, fmt.Errorf("get album artists: %w", err)
	}
	defer rows.Close()

	var artists []models.Artist
	for rows.Next() {
		var a models.Artist
		var rowid int64
		var idx int
		if err := rows.Scan(&a.ID, &a.Name, &a.Followers, &a.Popularity, &rowid, &idx); err != nil {
			return nil, fmt.Errorf("scan artist: %w", err)
		}
		artists = append(artists, a)
	}
	return artists, rows.Err()
}

func (d *DB) getArtistGenres(ctx context.Context, artistRowID int64) ([]string, error) {
	rows, err := d.main.QueryContext(ctx, `
		SELECT genre FROM artist_genres WHERE artist_rowid = ?
	`, artistRowID)
	if err != nil {
		return nil, fmt.Errorf("get artist genres: %w", err)
	}
	defer rows.Close()

	var genres []string
	for rows.Next() {
		var g string
		if err := rows.Scan(&g); err != nil {
			return nil, fmt.Errorf("scan genre: %w", err)
		}
		genres = append(genres, g)
	}
	return genres, rows.Err()
}

func (d *DB) getAlbumImages(ctx context.Context, albumRowID int64) ([]models.Image, error) {
	rows, err := d.main.QueryContext(ctx, `
		SELECT DISTINCT url, width, height FROM album_images
		WHERE album_rowid = ? ORDER BY width DESC
	`, albumRowID)
	if err != nil {
		return nil, fmt.Errorf("get album images: %w", err)
	}
	defer rows.Close()

	var images []models.Image
	for rows.Next() {
		var img models.Image
		if err := rows.Scan(&img.URL, &img.Width, &img.Height); err != nil {
			return nil, fmt.Errorf("scan image: %w", err)
		}
		images = append(images, img)
	}
	return images, rows.Err()
}

func (d *DB) getArtistImages(ctx context.Context, artistRowID int64) ([]models.Image, error) {
	rows, err := d.main.QueryContext(ctx, `
		SELECT url, width, height FROM artist_images
		WHERE artist_rowid = ? ORDER BY width DESC
	`, artistRowID)
	if err != nil {
		return nil, fmt.Errorf("get artist images: %w", err)
	}
	defer rows.Close()

	var images []models.Image
	for rows.Next() {
		var img models.Image
		if err := rows.Scan(&img.URL, &img.Width, &img.Height); err != nil {
			return nil, fmt.Errorf("scan image: %w", err)
		}
		images = append(images, img)
	}
	return images, rows.Err()
}

func (d *DB) BatchLookupTracks(ctx context.Context, ids []string) (map[string]*models.Track, error) {
	result := make(map[string]*models.Track)

	for _, id := range ids {
		track, err := d.LookupTrack(ctx, id)
		if err != nil {
			slog.Error("batch lookup track", "id", id, "err", err)
			continue
		}
		if track != nil {
			result[id] = track
		}
	}

	return result, nil
}

func (d *DB) BatchLookupArtists(ctx context.Context, ids []string) (map[string]*models.Artist, error) {
	result := make(map[string]*models.Artist)

	for _, id := range ids {
		artist, err := d.LookupArtist(ctx, id)
		if err != nil {
			slog.Error("batch lookup artist", "id", id, "err", err)
			continue
		}
		if artist != nil {
			result[id] = artist
		}
	}

	return result, nil
}

func (d *DB) BatchLookupAlbums(ctx context.Context, ids []string) (map[string]*models.Album, error) {
	result := make(map[string]*models.Album)

	for _, id := range ids {
		album, err := d.LookupAlbum(ctx, id)
		if err != nil {
			slog.Error("batch lookup album", "id", id, "err", err)
			continue
		}
		if album != nil {
			result[id] = album
		}
	}

	return result, nil
}

func (d *DB) BatchLookupISRCs(ctx context.Context, isrcs []string) (map[string][]models.Track, error) {
	result := make(map[string][]models.Track)

	for _, isrc := range isrcs {
		tracks, err := d.LookupISRC(ctx, isrc)
		if err != nil {
			slog.Error("batch lookup isrc", "isrc", isrc, "err", err)
			continue
		}
		if len(tracks) > 0 {
			result[isrc] = tracks
		}
	}

	return result, nil
}
