# DepViz Agent Handoff

This is the public handoff for agents continuing DepViz v4 work.

Last updated: 2026-06-18.

## Current State

DepViz v4 is now the main line of this repository. The old v3 code remains
available from release tags such as `v3.20.0` and the `v3` branch.

The current product direction is:

- local-first work graph engine
- board-scoped dependency graphs over real external work items
- GitHub issues and PRs as first-class external refs
- local-only notes/tasks for context that does not belong upstream
- multiple views over the same graph, starting with Brief, Graph, and Table
- stateless Live mode that can run from GitHub Pages before a backend exists
- inferred/source relations that stay soft until a human promotes them

The useful mental model is "a GitHub Project board, but graph-native, local
first, multi-source, and view-specialized." The same GitHub issue or PR can
appear in multiple DepViz boards with different local context and dependency
meaning.

## Main Capabilities

Merged v4 work currently supports:

- Go CLI and local SQLite state in `.depviz/state.db`
- GitHub sync through `gh`
- local-only notes
- dependency edges and ready/blocker summaries
- JSON export for tools and Live mode
- single-file HTML export
- `depviz live` static browser app
- DepViz Flow, JSONL, and JSON export input in Live
- syntax highlighting for Flow/JSON/JSONL
- browser-side GitHub hydration with optional token kept in `sessionStorage`
- compact badges for GitHub type, lifecycle, review, and CI state
- source-inferred relations from GitHub issue/PR bodies
- soft/dashed inferred relations that do not affect ready/blocker semantics
- Suggested relations panel with Focus/Promote/Hide
- relation-aware graph layout for realistic imports
- Pages previews for PRs

Important recent merged PRs:

- #679 `feat: bootstrap depviz v4`
- #682 `feat: hydrate live GitHub refs`
- #684 `feat: add click GitHub auth flow`
- #685 `fix: keep soft github relations nonblocking`
- #686 `feat: add suggested relation review`
- #687 `feat: improve live graph layout`

## Open PR Stack

As of this handoff, the open edge-inspector loop is:

- #688 `feat: make graph edges selectable`
- #689 `feat: show selected edge inspector`
- #690 `feat: add live edge inspector workflow`

Review #690 first. It targets `master` and includes the full result. #688 and
#689 are smaller step PRs for precise review comments.

The intended loop for future UI work is:

1. open PR A for a small isolated step
2. open PR B stacked on PR A for the next step
3. open PR C against `master` with the full combined result

The user can review PR C directly or comment on PR A/B if a specific step needs
changes. Every UI PR must include a Manual QA section with either a Pages preview
link or exact local `make live` steps.

## Source Layout

Key files:

- `cmd/depviz/main.go`: CLI entrypoint
- `internal/core/`: local model, store, brief/export/render logic, GitHub sync
- `live/static.go`: embedded Live assets
- `live/app/index.html`: Live shell
- `live/app/app.js`: Live parser, renderer, GitHub hydration, graph UI
- `live/app/style.css`: Live styling
- `docs/DEPVIZ-FLOW.md`: human input format
- `docs/POC.md`: POC success criteria and next slices
- `testdata/simple/`: small fixture
- `testdata/realistic/gno-last-100/`: realistic GitHub import fixture

## Data Model Notes

Nodes represent work items or local-only notes/tasks.

GitHub refs use canonical ids:

```text
gh:owner/repo#123
gh:owner/repo!456
```

Edges have:

- `from_id`
- `to_id`
- `kind`
- `authority`
- `confidence`
- `evidence_json`

Important edge semantics:

- `depends on` in Flow is stored as a blocking relationship where each target
  blocks the subject.
- `blocks` means the subject blocks each target.
- `addresses`, `mentions`, `relates_to`, and `closes` are non-blocking.
- `authority` values containing `inferred` or `soft`, or confidence below 1,
  should render soft and should not drive ready/blocker calculations.
- Promoting a soft edge should create or rewrite a local/official edge with
  confidence 1 while preserving provenance in evidence.

## DepViz Flow

Flow is the human-readable input format for Live and documentation snippets.
JSON/JSONL remain the machine formats.

Canonical example:

```depviz
repo moul/depviz

#679 depends on #80, #81 and blocks #85
#156 depends on moul/depviz2#5252
```

Standalone Live examples can define nodes by hand:

```depviz
repo moul/depviz

#679 "Bootstrap depviz v4" [open] @v4
note flow "Design DepViz Flow"

#679 addresses flow
```

Keep Flow Markdown-friendly and verb-first. Prefer `#1 depends on #2` over
arrow syntax in new examples.

## Testing And Verification

Baseline:

```text
make test
node --check live/app/app.js
```

Local Live:

```text
make live
# opens http://127.0.0.1:8686/
```

Pages routes:

- `master`: `https://moul.github.io/depviz/live/`
- PR preview: `https://moul.github.io/depviz/previews/pr-N/live/`

If a freshly published preview returns 404 on the route without query params,
try a cache-buster such as `?v=prN`. The `gh-pages` branch may have the files
before GitHub Pages cache has refreshed.

For realistic UI checks, use:

```text
testdata/realistic/gno-last-100/export.json
```

That fixture has 103 nodes and 11 inferred edges. It is useful for checking that
the graph, Suggested relations, and filtering are still usable on non-toy data.

## Product Decisions To Preserve

- DepViz should not force one canonical board for everything. Independent boards
  with specialized views are useful, even if a future "single big graph with
  filters/views" becomes possible.
- External GitHub truth and local DepViz-only context must coexist.
- Local-only notes, labels, comments, and view metadata are not second-class;
  they are part of making external work manageable.
- When possible, UI actions should eventually affect the real upstream thing.
  For now, Live is stateless and writes decisions back into the current input.
- Source-inferred relations are useful suggestions, not official dependency
  truth. They must be visibly soft and easy to promote.
- Live must remain usable without a backend. Backend/cache/MCP work can come
  later without invalidating the static mode.
- Avoid Node.js build requirements for Live unless the value is overwhelming.

## Good Next Slices

Good next work after the edge-inspector loop:

1. Better graph ergonomics: pan/zoom, fit-to-selection, selected edge scroll.
2. Saved board/view config files under `.depviz/views/*.toml`.
3. Better GitHub stale/placeholder hydration diagnostics.
4. Shared parser/golden fixtures for CLI and Live Flow parsing.
5. Local `depviz mcp` for agent use.
6. `depviz.io/live` routing to the Pages Live app.
7. Gantt view over the same snapshot.
8. Dark mode.

Prefer small PR loops with visible Manual QA over large unreviewable drops.
