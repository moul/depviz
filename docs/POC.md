# DepViz v4 POC

The first v4 POC is **DepViz Board Brief**.

It proves that DepViz can be useful locally before becoming a hosted product:

- import real GitHub issues and PRs
- mix them with local-only planning notes
- keep dependency edges board-scoped
- compute ready work and blockers
- print a morning brief
- export one static HTML file

## Success Criteria

The POC succeeds if:

- `make install` works from a clean checkout
- `depviz init` creates ignored local SQLite state
- `depviz sync github owner/repo` imports real cards through `gh`
- `depviz board note default "..."` creates a local-only card
- `depviz brief` is worth reading on a real repo
- `depviz gen html` creates an inspectable static file

The POC fails if the first impressive artifact is only a graph screenshot.

## Next Slices

1. Improve GitHub dependency extraction and stale detection.
2. Add saved board/view config files under `.depviz/`.
3. Add `depviz.io/live` as a GitHub Pages static app.
4. Add PR previews for `depviz.io/live` using GitHub Actions, with cleanup.
5. Add local `depviz mcp`.
