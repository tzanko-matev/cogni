# Plan: Avoid tracked report asset build output

Date: 2026-01-16
Owner: Codex
Status: DONE

## Goal

Prevent routine frontend builds from modifying tracked files under
`internal/reportserver/assets` while keeping embedded assets available for Go
builds and tests.

## Constraints

- Preserve embedded assets for `internal/reportserver` tests.
- Default build should not touch tracked assets.
- Provide an explicit sync step for updating embedded assets when needed.

## Steps

### Step 1: Build output isolation

- Update Vite build output to `web/dist` (untracked).
- Add `web/dist/` to `.gitignore`.
- Tests: `npm test` still runs with sources unchanged (no build needed).

### Step 2: Explicit asset sync

- Add a `just web-sync-assets` recipe that copies `web/dist/*` into
  `internal/reportserver/assets/`.
- Update `just web-build` to only build into `web/dist`.
- Tests: `just build` still succeeds and uses embedded assets.

### Step 3: Refresh embedded assets once

- Run `just web-sync-assets` to update embedded assets with the latest UI.
- Commit updated `internal/reportserver/assets/*` and `manifest.json`.

## Completion Criteria

- Running `just web-build` does not modify tracked assets.
- Running `just web-sync-assets` updates embedded assets.
- Documentation/notes in status file updated.
