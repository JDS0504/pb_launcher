# AI Handoff Template

Use this template when pausing work so the next model can continue with minimal rediscovery.

## Goal

- Describe the user-facing goal in one or two sentences.

## Current State

- Summarize what is already implemented.
- Mention files changed.
- Mention any schema/migration changes.

## Important Decisions

- Capture constraints and product decisions that should not be revisited accidentally.
- Include stopped/running safety requirements when relevant.
- Include compatibility constraints such as repository/version matching.

## Remaining Work

- List concrete next steps.
- Prefer file paths and exact commands.

## Verification

- Record commands already run and results.
- If verification was skipped, explain why.

Expected full verification order:

1. `cd ui && npm run lint`
2. `cd ui && npm run build`
3. `go test ./...`

## Known Risks

- Mention warnings, edge cases, or untested paths.
- Mention any existing dirty worktree changes that were not yours.

## Useful Context

- Link to `docs/AI_GUIDE.md`.
- Mention relevant routes, collection names, or runtime paths.
