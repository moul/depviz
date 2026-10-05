# DepViz Agent Handoff

This is the public handoff for agents continuing DepViz v4 work.

Last updated: 2026-10-05.

## Current Checkpoint

The working repository is `/Users/moul/p/gh/moul/depviz` and the canonical
remote is `https://github.com/moul/depviz`. Some environments mention a
parallel `depviz2` path, but the active v4 line and current `master` are in
this `depviz` repository.

At this checkpoint:

- `master` is clean and matches `origin/master`; this handoff refresh is the
  latest documentation checkpoint
- the latest commit says `chore: disable dependency automation (repo is unmaintained)`
- open PRs are #724 (setup-go 7), #726 (README capitalization), and #730
  (modernc.org/sqlite 1.58.0); all currently report clean/successful checks
- open issue #703 is the stateful UX roadmap and the best product backlog
- the old edge-inspector PR stack (#688/#689/#690) is historical, not open
- release tags include `v4.0.0`, `v3-final`, and `v3.20.0`; the `v3` branch
  still preserves the former implementation

Treat the repository's “unmaintained” commit as an explicit resumption
checkpoint: first decide whether to revive product work or only do maintenance,
then update this handoff when that decision is made.

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

The implementation has moved beyond the original stateless POC. It now has a
stateful backend foundation alongside Live:

- `depviz server` serves the embedded app and `/api/*` endpoints
- GitHub OAuth/App account plumbing creates local HTTP-only sessions
- Basic Auth can gate public deployments without an OAuth app
- stateful boards, repo/org presets, sync metadata, board-status JSON, and
  private demo-board snapshots support a dogfood deployment
- the root route is a public landing page; the application lives under `/app/`
- the static Pages workflow still publishes `/live/` and per-PR previews

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

Important merged milestones:

- #679 `feat: bootstrap depviz v4`
- #682 `feat: hydrate live GitHub refs`
- #684 `feat: add click GitHub auth flow`
- #685 `fix: keep soft github relations nonblocking`
- #686 `feat: add suggested relation review`
- #687 `feat: improve live graph layout`
- #699 `feat: add backend account foundation`
- #700 `feat: add stateful live mode`
- #702 `support GitHub App auth`
- #705 `Relations UX + clickable filters, confidence tiers, sync quality panel`
- #706 `interactive strategy cockpit`
- #710 `DepViz UX backlog — all 20 stateful cockpit improvements`
- #712 `persistence, GitHub writes, sync reliability, and production-grade cockpit`
- #714 `correctness, workspaces, webhooks, and daily-use hardening`
- #715 `real-time activity progress bar`
- #716 `board-status brief workflow`
- #717 `SQLite WAL + busy_timeout`
- #718 `deployment gate + board-status JSON + post-deploy contract`
- #719 `landing page at /, app at /app/`
- #721 `cold-open sample-board fallback`
- #722 `private demo board behind auth`
- #723 `count demo-board done rows as closed`

## Open PRs And Backlog

The current open PRs are maintenance-only:

- #724 `chore(deps): bump actions/setup-go from 6 to 7`
- #726 `docs: capitalize bullet list items in README.md`
- #730 `chore(deps): bump modernc.org/sqlite from 1.53.0 to 1.58.0`

Their checks were green at the time of this handoff. They are independent of
the product roadmap; review and merge them only after deciding whether the
repository is being resumed. Issue #703 is the active product roadmap. Its
recommended sequence is Relations-first UX and filters, sync quality and
per-view settings, GitHub write actions, webhook/background sync, then shared
cache and personal overrides.

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
- `internal/backend/`: HTTP server, GitHub auth, sessions, Basic Auth, and
  authenticated demo-board snapshot APIs
- `live/static.go`: embedded Live assets
- `live/app/index.html`: Live shell
- `live/app/app.js`: Live parser, renderer, GitHub hydration, graph UI
- `live/app/style.css`: Live styling
- `docs/DEPVIZ-FLOW.md`: human input format
- `docs/POC.md`: POC success criteria and next slices
- `docs/meta-repo-strategy.md`: multi-repo/meta-repo direction
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
go vet ./...
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

For backend/deployment checks, use:

```text
go test ./...
scripts/check-deploy.sh https://depviz.example
```

The CI fixture smoke also exercises `init`, event ingest, local notes, edges,
brief output, JSON export, and HTML export in a temporary database.

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

Good next work if the project is explicitly resumed:

1. Re-open the product loop with issue #703: make Relations the primary
   stateful surface and add label/assignee/status/link filters.
2. Add explicit sync-quality diagnostics and per-view refresh settings.
3. Finish the GitHub write-action surface: comment, labels, assignment,
   milestone, and create/link workflows.
4. Reconcile webhook ingestion and background jobs with the existing auth,
   workspace, and cache model.
5. Add saved `.depviz/views/*.toml` configs and preserve board/view/selection
   context across reloads.
6. Keep the parser and Live/CLI golden fixtures in lockstep.
7. Start local `depviz mcp` once the stateful model is stable enough for agents.
8. Add the shared-graph zoom/filter model, then Gantt and dark mode.

Prefer small PR loops with visible Manual QA over large unreviewable drops.

## Handoff Procedure

Before changing code, inspect `git status`, `git log -10`, open PRs, and issue
#703. Confirm whether the goal is maintenance or product revival. For frontend
changes, run the local Live app and the stateful server when relevant, include
exact manual QA steps and a preview URL in the PR, and use a cache-buster if
Pages appears stale. Keep commits conventional and single-purpose; never add
an AI co-author line.
