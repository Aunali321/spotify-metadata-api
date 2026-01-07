# Music Metadata API

A metadata enrichment API for music servers, backed by SQLite databases containing 256 million tracks.

## Disclaimer

**This project is not affiliated with, endorsed by, or connected to Spotify AB or any other music streaming service.** This is independent open-source software that provides API infrastructure for querying music metadata databases.

## Warning

**This repository does not include any databases or copyrighted data.** You must obtain the SQLite databases separately. This project only provides the API server code to query existing databases.

The author(s) of this project are not responsible for how you obtain or use the underlying data. Users are solely responsible for ensuring their use of any databases complies with applicable laws and terms of service. This software is provided "as is" without warranty of any kind.

## Features

- **Batch API** - Lookup up to 400 entities in a single request
- **Generous rate limits** - 100 req/s with burst capacity of 200
- ISRC lookup (primary identifier for recordings)
- Track/Artist/Album lookup by ID
- Search by name
- Full metadata: images, genres, labels, copyright, release dates
- OpenAPI 3.1 spec with Swagger UI

## Requirements

- Go 1.24+
- SQLite databases (two files required):
  - Main metadata database (~117GB)
  - Track files database (~99GB)

## Installation

```bash
go build -o metadata-api ./cmd/server
```

## Usage

```bash
./metadata-api -db /path/to/main_database.sqlite3
```

The server expects both database files to be in the same directory.

**Flags:**
- `-db` - Path to main database file (required)
- `-addr` - Listen address (default: `:8080`)

## Docker

### Using Pre-built Image (Recommended)

```bash
docker run -p 8080:8080 \
  -v /path/to/databases:/data:ro \
  ghcr.io/aunali321/music-metadata-api:latest \
  -db /data/main_database.sqlite3
```

### Docker Compose (Production)

```bash
# Edit docker-compose.yml and set your database path
# Change: /path/to/your/databases -> your actual path

docker-compose up -d
```

The compose file includes:
- Auto-restart policy
- Health checks
- Read-only database mounts
- Port mapping (8080:8080)

### Building Locally

```bash
# Build
docker build -t metadata-api .

# Run (mount your database directory)
docker run -p 8080:8080 -v /path/to/databases:/data metadata-api -db /data/main_database.sqlite3
```

## API Endpoints

| Endpoint | Description |
|----------|-------------|
| `POST /batch/lookup` | **Batch lookup multiple entities** |
| `GET /lookup/isrc/{isrc}` | Lookup tracks by ISRC |
| `GET /lookup/track/{id}` | Lookup track by ID |
| `GET /lookup/artist/{id}` | Lookup artist by ID |
| `GET /lookup/album/{id}` | Lookup album by ID |
| `GET /lookup/album/{id}/tracks` | Get all tracks in album |
| `GET /search/track?q=&limit=` | Search tracks by name (case-insensitive) |
| `GET /search/artist?q=&limit=` | Search artists by name (case-insensitive) |
| `GET /health` | Health check |
| `GET /docs` | Swagger UI |
| `GET /openapi.yaml` | OpenAPI spec |

### Search Behavior

Search endpoints use **case-insensitive substring matching**:
- ✅ `q=gaga` matches "Lady Gaga"
- ✅ `q=Bohemian` matches "Bohemian Rhapsody"
- ✅ `q=lady` matches "Lady Gaga", "Lady Antebellum"
- Minimum 2 characters required
- 10-second timeout for protection
- Results ordered by popularity/followers
- Default limit: 20, max: 50

## Rate Limits

This API has generous rate limits designed for high-volume usage:

- **100 requests per second** per IP address
- **Burst capacity of 200 requests** for handling traffic spikes
- Rate limits apply across all endpoints
- Returns HTTP 429 when exceeded

### Throughput Examples

**Individual endpoints:**
- 6,000 entities per minute (100 req/s × 60s)

**Batch API:**
- Up to 600,000 entities per minute (100 req/s × 100 items × 60s)
- Maximum 400 total items per batch request
- Can mix tracks, artists, albums, and ISRCs in single request

## Examples

### Individual Lookups

```bash
# Lookup by ISRC
curl http://localhost:8080/lookup/isrc/USUM72409273 | jq '.[0]'

# Lookup track by ID
curl http://localhost:8080/lookup/track/2plbrEY59IikOBgBGLjaoe

# Search
curl "http://localhost:8080/search/track?q=Bohemian%20Rhapsody&limit=5"
```

### Batch Lookup (Recommended for bulk operations)

```bash
curl -X POST http://localhost:8080/batch/lookup \
  -H "Content-Type: application/json" \
  -d '{
    "tracks": ["2plbrEY59IikOBgBGLjaoe", "3n3Ppam7vgaVa1iaRUc9Lp"],
    "artists": ["1HY2Jd0NmPuamShAr6KMms"],
    "albums": ["10FLjwfpbxLmW8c25Xyc2N"],
    "isrcs": ["USUM72409273"]
  }'
```

**Batch Response Format:**
```json
{
  "tracks": {
    "2plbrEY59IikOBgBGLjaoe": { "id": "2plbrEY59IikOBgBGLjaoe", "name": "Die With A Smile", ... },
    "3n3Ppam7vgaVa1iaRUc9Lp": { ... }
  },
  "artists": {
    "1HY2Jd0NmPuamShAr6KMms": { "id": "1HY2Jd0NmPuamShAr6KMms", "name": "Lady Gaga", ... }
  },
  "albums": {
    "10FLjwfpbxLmW8c25Xyc2N": { ... }
  },
  "isrcs": {
    "USUM72409273": [ { "id": "2plbrEY59IikOBgBGLjaoe", ... } ]
  }
}
```

## Individual Response Format

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