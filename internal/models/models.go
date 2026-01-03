package models

type Image struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type Artist struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Followers  int64    `json:"followers"`
	Popularity int      `json:"popularity"`
	Genres     []string `json:"genres,omitempty"`
	Images     []Image  `json:"images,omitempty"`
}

type Album struct {
	ID                   string   `json:"id"`
	Name                 string   `json:"name"`
	Type                 string   `json:"type"`
	Label                string   `json:"label"`
	ReleaseDate          string   `json:"release_date"`
	ReleaseDatePrecision string   `json:"release_date_precision"`
	UPC                  string   `json:"upc,omitempty"`
	TotalTracks          int      `json:"total_tracks"`
	CopyrightC           string   `json:"copyright,omitempty"`
	CopyrightP           string   `json:"copyright_p,omitempty"`
	Images               []Image  `json:"images,omitempty"`
	Artists              []Artist `json:"artists,omitempty"`
}

type Track struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	ISRC          string   `json:"isrc,omitempty"`
	DurationMs    int64    `json:"duration_ms"`
	Explicit      bool     `json:"explicit"`
	TrackNum      int      `json:"track_number"`
	DiscNum       int      `json:"disc_number"`
	Popularity    int      `json:"popularity"`
	PreviewURL    string   `json:"preview_url,omitempty"`
	Album         *Album   `json:"album,omitempty"`
	Artists       []Artist `json:"artists,omitempty"`
	OriginalTitle string   `json:"original_title,omitempty"`
	VersionTitle  string   `json:"version_title,omitempty"`
	HasLyrics     *bool    `json:"has_lyrics,omitempty"`
	Languages     []string `json:"languages,omitempty"`
	ArtistRoles   []string `json:"artist_roles,omitempty"`
}

type BatchLookupRequest struct {
	Tracks  []string `json:"tracks,omitempty"`  // Spotify track IDs
	Artists []string `json:"artists,omitempty"` // Spotify artist IDs
	Albums  []string `json:"albums,omitempty"`  // Spotify album IDs
	ISRCs   []string `json:"isrcs,omitempty"`   // ISRCs
}

type BatchLookupResponse struct {
	Tracks  map[string]*Track  `json:"tracks,omitempty"`
	Artists map[string]*Artist `json:"artists,omitempty"`
	Albums  map[string]*Album  `json:"albums,omitempty"`
	ISRCs   map[string][]Track `json:"isrcs,omitempty"`
	Errors  map[string]string  `json:"errors,omitempty"`
}
