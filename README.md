# pgs3backup

[![Go Report Card](https://goreportcard.com/badge/github.com/vladkanatov/pgs3backup)](https://goreportcard.com/report/github.com/vladkanatov/pgs3backup)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/github/go-mod/go-version/vladkanatov/pgs3backup)](https://github.com/vladkanatov/pgs3backup)
[![Release](https://img.shields.io/github/v/release/vladkanatov/pgs3backup)](https://github.com/vladkanatov/pgs3backup/releases)
[![Docker](https://img.shields.io/badge/docker-ghcr.io-blue)](https://github.com/vladkanatov/pgs3backup/pkgs/container/pgs3backup)
[![CI](https://github.com/vladkanatov/pgs3backup/workflows/Test/badge.svg)](https://github.com/vladkanatov/pgs3backup/actions)

PostgreSQL backup utility with S3 storage.

## Features

- Pure Go implementation using `database/sql` (no external dependencies)
- Exports schema (CREATE TABLE statements) and data as CSV files in tar archive
- Optional gzip compression
- S3/MinIO compatible storage
- Configuration via environment variables
- Docker ready

## Installation

```bash
go mod download
go build -o pgs3backup ./cmd/pgs3backup
```

## Usage

1. Copy `.env.example` to `.env`:
```bash
cp .env.example .env
```

2. Edit `.env` with your settings:
```env
DB_HOST=localhost
DB_PORT=5432
DB_NAME=mydb
DB_USER=postgres
DB_PASSWORD=password

S3_BUCKET=my-backups
S3_REGION=us-east-1
S3_ACCESS_KEY=your-access-key
S3_SECRET_KEY=your-secret-key
```

3. Run backup:
```bash
./pgs3backup
```

## Environment Variables

### PostgreSQL
- `DB_HOST` - database host (default: `localhost`)
- `DB_PORT` - database port (default: `5432`)
- `DB_NAME` - database name (required)
- `DB_USER` - database user (default: `postgres`)
- `DB_PASSWORD` - user password

### S3
- `S3_BUCKET` - S3 bucket name (required)
- `S3_REGION` - S3 region (default: `us-east-1`)
- `S3_ACCESS_KEY` - access key (required)
- `S3_SECRET_KEY` - secret key (required)
- `S3_ENDPOINT` - custom endpoint for MinIO, etc. (optional)

### Backup
- `BACKUP_PREFIX` - prefix for files in S3 (default: `backups`)
- `COMPRESS` - enable gzip compression (default: `true`)

## Backup Format

Files are saved as:
```
{BACKUP_PREFIX}/{DB_NAME}_{YYYY-MM-DD_HH-MM-SS}.tar[.gz]
```

Example: `backups/mydb_2025-11-11_15-30-45.tar.gz`

Archive contents:
- `schema.sql` - CREATE TABLE statements for all tables
- `data/public.table_name.csv` - CSV files with table data

## Docker

Pull from GitHub Container Registry:
```bash
docker pull ghcr.io/vladkanatov/pgs3backup:latest
```

Run with environment variables:
```bash
docker run --rm \
  -e DB_HOST=your-db-host \
  -e DB_NAME=your-db-name \
  -e DB_USER=postgres \
  -e DB_PASSWORD=your-password \
  -e S3_BUCKET=your-bucket \
  -e S3_ACCESS_KEY=your-key \
  -e S3_SECRET_KEY=your-secret \
  ghcr.io/vladkanatov/pgs3backup:latest
```

Or use with `.env` file:
```bash
docker run --rm --env-file .env ghcr.io/vladkanatov/pgs3backup:latest
```

Build locally:
```bash
docker build -t pgs3backup .
```
              valueFrom:
```

## Cron Usage

```bash
# Daily at 2:00 AM
0 2 * * * /path/to/pgs3backup >> /var/log/pgs3backup.log 2>&1
```

## Requirements

- Go 1.21+
- PostgreSQL database access
- S3 or S3-compatible storage access

Note: `pg_dump` is not required - pure Go implementation.

## License

MIT
