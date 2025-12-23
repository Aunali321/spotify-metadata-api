# Spotify Metadata API

A metadata enrichment API for music servers, backed by SQLite databases containing 256 million tracks.

## Warning

**This repository does not include any databases or copyrighted data.** You must obtain the SQLite databases separately. This project only provides the API server code to query existing databases.

The author(s) of this project are not responsible for how you obtain or use the underlying data. This software is provided "as is" without warranty of any kind.

## Features

- ISRC lookup (primary identifier for recordings)
- Track/Artist/Album lookup by Spotify ID
- Search by name
- Full metadata: images, genres, labels, copyright, release dates
- OpenAPI 3.1 spec with Swagger UI

## Requirements

- Go 1.24+
- SQLite databases:
  - `spotify_clean.sqlite3` (~117GB)
  - `spotify_clean_track_files.sqlite3` (~99GB)

## Installation

```bash
go build -o metadata-api ./cmd/server
```

## Usage

```bash
./metadata-api -db /path/to/spotify_clean.sqlite3
```

The server expects `spotify_clean_track_files.sqlite3` to be in the same directory as the main database.

**Flags:**
- `-db` - Path to `spotify_clean.sqlite3` (required)
- `-addr` - Listen address (default: `:8080`)

## API Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /lookup/isrc/{isrc}` | Lookup tracks by ISRC |
| `GET /lookup/track/{id}` | Lookup track by Spotify ID |
| `GET /lookup/artist/{id}` | Lookup artist by Spotify ID |
| `GET /lookup/album/{id}` | Lookup album by Spotify ID |
| `GET /lookup/album/{id}/tracks` | Get all tracks in album |
| `GET /search/track?q=&limit=` | Search tracks by name |
| `GET /search/artist?q=&limit=` | Search artists by name |
| `GET /health` | Health check |
| `GET /docs` | Swagger UI |
| `GET /openapi.yaml` | OpenAPI spec |

## Example

```bash
# Lookup by ISRC
curl http://localhost:8080/lookup/isrc/USUM72409273 | jq '.[0]'

# Search
curl "http://localhost:8080/search/track?q=Bohemian%20Rhapsody&limit=5"
```

## Response Format

```json
{
  "id": "2plbrEY59IikOBgBGLjaoe",
  "name": "Die With A Smile",
  "isrc": "USUM72409273",
  "duration_ms": 251667,
  "explicit": false,
  "track_number": 1,
  "disc_number": 1,
  "popularity": 100,
  "album": {
    "id": "10FLjwfpbxLmW8c25Xyc2N",
    "name": "Die With A Smile",
    "type": "single",
    "label": "Interscope",
    "release_date": "2024-08-16",
    "upc": "00602475093060",
    "images": [
      {"url": "https://i.scdn.co/image/...", "width": 640, "height": 640}
    ]
  },
  "artists": [
    {"id": "1HY2Jd0NmPuamShAr6KMms", "name": "Lady Gaga", "genres": ["art pop", "pop"]}
  ],
  "languages": ["en"]
}
```

## License

MIT

