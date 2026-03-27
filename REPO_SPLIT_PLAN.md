# Repo Split Plan (Frontend + PayLock Infra)

## Goal

Split the current codebase into two repositories:

- **Infra repo**: `paylock` (backend API + on-chain infra + Sui Move contracts)
- **Frontend repo**: application layer UI (vanilla JS SPA)

This document captures the planned updates for the current repo **before** implementation.

## Key Decisions

- Infra will be public and usable by any frontend.
- `/api/*` endpoints in infra must support CORS for cross-origin frontends.
- Final state is **two separate repos**, not a monorepo with two folders.
- Sui Move contracts (`contracts/`) stay in the infra repo — they define the on-chain protocol.

## Current Architecture

```
cmd/paylock/
├── main.go            — wires all packages; embeds SPA via go:embed
└── web/               — embedded SPA (11 vanilla JS files)

internal/
├── config/            — env-based configuration
├── handler/           — HTTP handlers (upload, status, videos, delete, config, stream)
├── indexer/           — Sui chain reindexer (FetchAll on startup)
├── middleware/         — CORS middleware (currently /stream/* only)
├── model/             — VideoStore (sync.RWMutex + JSON file persistence)
├── processor/         — FFmpeg validators, magic-byte checks, preview extraction
├── suiauth/           — Sui wallet signature verification
├── testutil/          — test helpers
├── walrus/            — Walrus HTTP client (Store, BlobURL)
└── watcher/           — chain event watcher (polls VideoCreated events)

contracts/             — Sui Move contract (gating.move)
```

### Current Routes

| Method | Path | CORS | Description |
|--------|------|------|-------------|
| `POST` | `/api/upload` | No | Upload video (202 async) |
| `GET` | `/api/status/{id}` | No | Get video status |
| `GET` | `/api/status/{id}/events` | No | SSE stream for status updates |
| `GET` | `/api/videos` | No | List videos (paginated) |
| `DELETE` | `/api/videos/{id}` | No | Delete video record |
| `GET` | `/api/config` | No | Client configuration |
| `GET` | `/stream/{id}/preview` | Yes | 307 redirect to Walrus preview blob |
| `GET` | `/stream/{id}/full` | Yes | 307 redirect to Walrus full blob |
| `GET` | `/` | — | Serve embedded SPA (fallback routing) |

## Scope of Changes (Infra Repo)

### Code Changes

1. **Remove embedded frontend SPA**
   - SPA assets live in `cmd/paylock/web/` (11 files: app.js, wallet.js, player-view.js, upload-section.js, etc.)
   - Server embeds SPA via `//go:embed web` in `cmd/paylock/main.go`
   - Plan: delete `cmd/paylock/web/`, remove `go:embed` directive, remove the `GET /` SPA handler (lines 106-124)
   - Replace `GET /` with a simple JSON health/info response (e.g., `{"service": "paylock", "version": "..."}`)

2. **CORS for `/api/*`**
   - Current state: only `/stream/*` has CORS (allows `Origin: *`, methods `GET, OPTIONS`, header `Range`)
   - Plan: enable CORS for all `/api/*` routes
   - Required methods: `GET, POST, DELETE, OPTIONS`
   - Required headers: `Content-Type, Range, X-Wallet-Address, X-Wallet-Sig, X-Wallet-Timestamp, X-Creator`
   - Expose headers: `Content-Range, Content-Length`
   - Allowed origins: configurable via `PAYLOCK_CORS_ORIGINS` env var (default: `*` for dev, restrict in prod)

3. **SSE CORS consideration**
   - `GET /api/status/{id}/events` uses Server-Sent Events — needs CORS to work cross-origin
   - EventSource API only sends simple requests, so standard CORS headers suffice

### Documentation Changes

1. **README.md**
   - Remove "embedded web UI" from features list
   - Remove SPA-related setup instructions
   - Clarify infra as standalone backend service
   - Add section on integrating with an external frontend

2. **API.md**
   - Add CORS documentation (allowed origins, preflight behavior)
   - Reframe examples as "external frontend integration"

3. **Agent docs (CLAUDE.md, GEMINI.md, AGENTS.md)**
   - Remove `cmd/paylock/web/` from architecture descriptions
   - Update route table (remove `GET /`, add health endpoint)

## Scope of Changes (Frontend Repo)

- Create new repo with its own README
- Migrate `cmd/paylock/web/` files as starting point
- Document setup, env vars (e.g., `PAYLOCK_BASE_URL` for API base)
- Include wallet integration flows (Sui wallet sig auth)
- Document Seal encryption flow (client-side encrypt → Walrus upload)
- Document purchase flow (AccessPass → Seal decrypt → playback)
- Add build/dev tooling as needed (currently vanilla JS, no bundler)

## Non-Goals

- No refactors to business logic at this stage
- No new features beyond CORS and repo split cleanup
- No changes to Sui Move contracts
- No changes to chain indexer/watcher logic

## Resolved Decisions

- **`GET /` behavior**: return a JSON health/info response (not 404 or redirect)
- **CORS policy**: configurable via env var; default `*` for development convenience
- **Contracts location**: stays in infra repo (defines the on-chain protocol)

## Open Questions

- Should the frontend repo use a bundler/framework, or stay vanilla JS?
- Do we need a shared types/constants package between repos (e.g., auth header names)?

## Proposed Order of Work

1. Remove embedded SPA and `GET /` route; add health endpoint
2. Add `/api/*` CORS support (with configurable origins)
3. Update infra docs (README, API.md, agent docs)
4. Create frontend repo and migrate UI assets
5. Verify cross-origin integration works end-to-end
