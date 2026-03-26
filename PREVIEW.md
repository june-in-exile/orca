# Preview Duration: User-Configurable Plan

## Goals
- Allow users to choose preview length per upload.
- Keep strong server-side enforcement with clear maximums.
- Preserve current defaults (10s) when user does not choose.

## Current Behavior Summary
- Default preview length is hard-coded to 10s in `internal/config/config.go` (`PreviewDuration`).
- Frontend fetches `/api/config` and uses `preview_duration` as a single fixed value for client-side preview generation.
- Paid uploads: client generates a preview clip; backend validates duration with `FFprobe` against `MaxPreviewDuration`.
- Free uploads: backend uses FFmpeg to extract a preview of `PreviewDuration` seconds.

## Proposed Product Behavior
- User selects a preview duration (seconds) at upload time.
- If not specified, the default remains 10s.
- System enforces a max duration (existing `PAYLOCK_MAX_PREVIEW_DURATION`, default 30s).
- UI displays the allowed range and default value.

## API/Config Changes
1. **Expose defaults + limits via `/api/config`**
   - Add fields:
     - `preview_duration_default` (default 10)
     - `preview_duration_max` (from `PAYLOCK_MAX_PREVIEW_DURATION`)
     - Optional `preview_duration_min` (e.g., 1 or 2 seconds)
2. **Upload request accepts `preview_duration`**
   - For both free and paid uploads, accept a form field `preview_duration` in seconds.
   - If missing, server falls back to `preview_duration_default`.

## Backend Changes
1. **Config**
   - Add `PreviewDurationDefault` to `internal/config/config.go` (default 10).
   - Keep `MaxPreviewDuration` as the cap (already exists).
   - Optional: add `MinPreviewDuration` constant if you want a floor (e.g., 1s or 2s).
2. **Upload handling**
   - Parse `preview_duration` in `internal/handler/upload.go`.
   - Validate integer bounds: `min <= requested <= max`.
   - Free uploads: pass the validated duration into `processor.ExtractPreview` instead of `cfg.PreviewDuration`.
   - Paid uploads:
     - Keep `ValidatePreviewDuration` to enforce max (already covers actual preview length).
     - If FFmpeg/FFprobe disabled, decide policy:
       - **Safer**: reject paid uploads because duration cannot be verified (aligns with README note), or
       - **Less strict**: allow but rely on `MaxPreviewSize` + client UI, and log a warning.
3. **App config handler**
   - Update `internal/handler/app_config.go` to return the new fields so the UI can render correct controls.

## Frontend Changes
1. **Add UI control**
   - Slider/input for preview seconds on upload screen.
   - Show default and max (from `/api/config`).
2. **Client preview generation**
   - Replace fixed `previewDurationSec` with the user-selected value.
   - Clamp to video duration: `Math.min(video.duration, selectedDuration)` (already done).
3. **Submit to backend**
   - Include `preview_duration` in `FormData` for both free and paid flows.

## Validation & Edge Cases
- If `preview_duration` is invalid (non-numeric, <=0, >max), return `400` with clear error.
- If `preview_duration` > actual video duration, the actual generated clip will be shorter; server-side ffprobe validation should accept it.
- Consider locale and input type for mobile UX (numeric keypad).

## Tests to Add/Update
1. `internal/handler/upload_test.go`
   - Free upload uses requested duration within bounds (mock or assert derived behavior).
   - Paid upload rejects invalid `preview_duration` (if enforcing in handler).
   - Paid upload still validates actual preview duration with ffprobe when enabled.
2. `internal/handler/app_config_test.go` (if present or add new)
   - Ensure config returns default/max values.

## Rollout Plan
1. Backend: config + handler validation + API fields.
2. Frontend: UI control + pass `preview_duration` + client preview clip length.
3. Docs: update `API.md` with new parameter.
4. Optional: add a feature flag to force server default only until UI is ready.

## Open Questions
- Minimum allowed preview length: 1s or 2s?
- Policy when FFprobe/FFmpeg is disabled for paid uploads.
