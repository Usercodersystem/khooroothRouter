# Khoorooth Router — Go edition

A lightweight, single-binary AI gateway with an embedded Persian dashboard. This Go implementation is a clean-room port inspired by the feature set and data layout of [9Router](https://github.com/decolua/9router), with a compatibility importer for 9Router backups.

**Current milestone:** dashboard login, provider management, API key management, OpenAI-compatible `/v1/chat/completions` and `/v1/models`, priority fallback, usage logging, and 9Router backup migration. This is not yet feature-complete parity with all 9Router executors: provider-specific OAuth refresh, non-OpenAI protocol translators, and several specialized provider flows are not implemented. Imported OAuth data and unknown tables are retained; common bearer access tokens may be carried forward but are not refreshed.

## Start with Docker

```sh
cp .env.example .env
# Edit .env: set a strong INITIAL_PASSWORD and SESSION_SECRET
docker compose up -d --build
```

Open `http://SERVER:20128`. Create an API key in the dashboard and use `http://SERVER:20128/v1` as the OpenAI-compatible base URL.

## Build without Docker

The Go source and embedded dashboard are shipped in `source/khr-go-source.tar.gz` to keep the initial migration commit compact. Expand it once before building:

```sh
make unpack
go mod download
go build -trimpath -ldflags='-s -w' -o khoorooth ./cmd/khoorooth
INITIAL_PASSWORD='strong-password' SESSION_SECRET="$(openssl rand -hex 32)" DATA_DIR=./data ./khoorooth
```

## Import a 9Router backup

In the dashboard, upload a ZIP/TAR.GZ of the `.9router` data directory containing `db/data.sqlite`, the standalone `db/data.sqlite`, or a legacy `db.json` export. The importer validates SQLite with `PRAGMA quick_check`, maps recognizable provider connections/API keys/combos/aliases, and preserves all imported rows in `legacy_data` for future migrations. Common API-key provider base URLs are inferred if absent from the backup. Verify every provider before relying on it and keep the original backup.

Upload limits: 64 MiB archive; 192 MiB expanded archive. The current importer reads the uploaded archive into memory, so large backups can exceed the 128 MiB Compose memory envelope. Increase memory for larger imports; the 128 MiB limit is a target to measure, not a guarantee.

## API

- `GET /healthz`
- `GET /v1/models`
- `POST /v1/chat/completions` (streaming response passthrough)

Public API routes require `Authorization: Bearer <key>` by default. Providers are tried in ascending priority order; network errors, HTTP 429, and upstream 5xx responses trigger fallback.

## Acknowledgement

9Router by decolua and contributors is a separate MIT-licensed project. This repository uses its documented data layout as an interoperability target; the upstream Next.js runtime is not included. See `UPSTREAM-9ROUTER-LICENSE.txt` and https://github.com/decolua/9router.
