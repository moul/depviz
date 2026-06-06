# DepViz v4 POC

The first v4 POC is **DepViz Board Brief**.

It proves that DepViz can be useful locally before becoming a hosted product:

- import real GitHub issues and PRs
- mix them with local-only planning notes
- keep dependency edges board-scoped
- compute ready work and blockers
- print a morning brief
- export JSON for tools and live mode
- export one static HTML file
- serve a stateless Live v1 app without a Node.js build

## Success Criteria

The POC succeeds if:

- `make install` works from a clean checkout
- `depviz init` creates ignored local SQLite state
- `depviz sync github owner/repo` imports real cards through `gh`
- `depviz board note default "..."` creates a local-only card
- `depviz brief` is worth reading on a real repo
- `depviz gen json` creates a stable machine-readable export
- `depviz gen html` creates an inspectable static file
- `depviz live` serves a browser app that accepts JSONL or exported JSON
- Live input has syntax highlighting for JSON/JSONL
- every same-repo PR can publish a `/previews/pr-N/live/` Pages preview
- fixture output is covered by golden tests

The POC fails if the first impressive artifact is only a graph screenshot.

## Next Slices

1. Improve GitHub dependency extraction and stale detection.
2. Add saved board/view config files under `.depviz/`.
3. Point `depviz.io/live` at the GitHub Pages live route.
4. Add local `depviz mcp`.
