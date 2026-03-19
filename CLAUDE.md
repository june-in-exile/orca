# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
make run          # Run dev server (go run ./cmd/orca)
make build        # Compile to bin/orca
make test         # Run all tests with race detector and coverage
make lint         # go vet ./...
make clean        # Remove bin/ and storage/

# Run a single test
go test ./internal/processor/... -run TestValidateMagicBytes -v
```

**Prerequisites:** `ffmpeg` and `ffprobe` must be installed and on `PATH`. The server will fail to start if either is missing.

## Environment Variables

| Var | Default | Description |
|-----|---------|-------------|
| `ORCA_PORT` | `8080` | HTTP listen port |
| `ORCA_STORAGE_DIR` | `./storage` | Local storage root |
| `ORCA_API_KEY` | _(none)_ | Required for `/api/*` endpoints |
| `ORCA_FFMPEG_PATH` | `ffmpeg` | Path to ffmpeg binary |
| `ORCA_FFPROBE_PATH` | `ffprobe` | Path to ffprobe binary |
| `ORCA_MAX_FILE_SIZE_MB` | `500` | Upload size limit in MB |

## Architecture

Orca is a video-aware middleware layer that sits above generic blob storage (currently local disk, eventually Walrus on Sui). It accepts MP4 uploads, segments them into HLS via FFmpeg, and serves them with native byte-range and CORS support.

```
cmd/orca/main.go          — wires all packages; two route groups:
                            POST /api/upload, GET /api/status/{id}  → API key required
                            GET /stream/{id}/{file...}              → CORS open

internal/config/          — env-based config; validates ffmpeg/ffprobe at startup
internal/model/           — in-memory VideoStore (sync.RWMutex); NOT persisted across restarts
internal/storage/         — Backend interface + LocalStorage; future: WalrusStorage
internal/processor/       — FFmpeg wrapper (Segment/Probe) + magic-byte MP4 validator
internal/handler/         — HTTP handlers; upload streams to disk, then goroutine processes async
internal/middleware/      — APIKey auth (X-API-Key header) + CORS (open on /stream/*)
```

### Key design decisions

- **Upload flow is async**: `POST /api/upload` returns `202 processing` immediately; a goroutine runs ffprobe validation then ffmpeg segmentation; poll `GET /api/status/{id}` for `ready`/`failed`.
- **VideoStore is in-memory only**: videos are lost on restart. Adding persistence (DB or file-based index) is the next natural step.
- **Storage.Backend interface** (`internal/storage/storage.go`) is the seam for swapping local disk → Walrus. New backends implement `SaveUpload`, `SegmentPath`, `ManifestPath`, `OutputDir`.
- **Stream handler uses `http.ServeFile`** which handles HTTP Range requests (206 Partial Content) automatically — required for HLS seek to work.
- **Only MP4 input is accepted** (magic bytes check: `ftyp` at offset 4); FFmpeg outputs HLS with `-codec copy` (no re-encode) into 6-second segments named `seg%03d.ts`.
