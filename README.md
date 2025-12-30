# Vocabulator

A Go REST API for managing vocabulary words with SQLite persistence, dictionary lookup integration, and CSV import/export.

## Features

- CRUD operations for vocabulary words
- Random word retrieval
- Dictionary lookup via [Free Dictionary API](https://dictionaryapi.dev/)
- CSV import/export
- Filtering by source, tag, date range, and search
- Docker support for easy deployment

## Quick Start

### Local Development

```bash
# Run the server
make run

# Run tests
make test

# Build binary
make build
```

### Docker

```bash
# Build and run with Docker Compose
make docker-run

# Stop
make docker-stop
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| GET | `/api/v1/words` | List words (with filtering) |
| GET | `/api/v1/words/{id}` | Get word by ID |
| POST | `/api/v1/words` | Create word |
| PUT | `/api/v1/words/{id}` | Update word |
| DELETE | `/api/v1/words/{id}` | Delete word |
| GET | `/api/v1/words/random` | Get random word |
| GET | `/api/v1/words/{id}/definition` | Fetch definition from dictionary |
| POST | `/api/v1/words/import` | Import CSV file |
| GET | `/api/v1/words/export` | Export to CSV |

### Query Parameters for GET /api/v1/words

- `search` - partial match on word
- `source` - filter by source
- `tag` - filter by tag
- `from_date` / `to_date` - date range filter (YYYY-MM-DD)
- `limit` / `offset` - pagination

## Examples

### Create a word

```bash
curl -X POST http://localhost:8080/api/v1/words \
  -H "Content-Type: application/json" \
  -d '{
    "word": "ephemeral",
    "source": "Book: The Road",
    "date_learned": "2024-01-15",
    "part_of_speech": "adjective",
    "example_sentence": "The ephemeral beauty of cherry blossoms",
    "tags": ["literature", "nature"]
  }'
```

### Get a random word

```bash
curl http://localhost:8080/api/v1/words/random
```

### Get definition

```bash
curl http://localhost:8080/api/v1/words/1/definition
```

### List with filters

```bash
# Filter by tag
curl "http://localhost:8080/api/v1/words?tag=literature"

# Search
curl "http://localhost:8080/api/v1/words?search=eph"

# Date range
curl "http://localhost:8080/api/v1/words?from_date=2024-01-01&to_date=2024-12-31"
```

### Import CSV

```bash
curl -X POST http://localhost:8080/api/v1/words/import \
  -F "file=@words.csv"
```

### Export CSV

```bash
curl http://localhost:8080/api/v1/words/export -o words.csv
```

## CSV Format

```csv
word,source,date_learned,part_of_speech,example_sentence,tags
ephemeral,Book: The Road,2024-01-15,adjective,"The ephemeral beauty of cherry blossoms","literature,nature"
ubiquitous,Article: Tech Trends,2024-02-20,adjective,,"technology"
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| PORT | 8080 | Server port |
| DATABASE_PATH | ./vocabulator.db | SQLite database file path |
| MIGRATIONS_PATH | ./migrations | Path to migration files |

## Project Structure

```
vocabulator/
├── cmd/server/          # Application entry point
├── internal/
│   ├── api/            # HTTP handlers and routes
│   ├── models/         # Data structures
│   ├── repository/     # Database operations
│   └── services/       # Business logic
├── migrations/          # Database migrations
├── Dockerfile
├── docker-compose.yml
├── Makefile
└── README.md
```

## Development

### Prerequisites

- Go 1.21+
- Make
- Docker (optional)

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage
```

### Database Migrations

Migrations are run automatically on server start. To run manually:

```bash
# Apply migrations
make migrate-up

# Rollback migrations
make migrate-down
```
