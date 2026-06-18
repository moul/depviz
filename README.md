# DepViz

DepViz v4 is a local-first work graph engine for humans, agents, and existing
issue trackers.

It turns scattered issues, PRs, plans, docs, local notes, and dependency facts
into a board-scoped execution graph that answers:

```text
What can move now?
What is blocked?
What should happen next?
```

## v4 status

This repository is now the v4 line.

The previous v3 codebase is preserved by:

- release tags such as `v3.20.0`
- the `v3` branch, created from the old `master`

Curious users can continue exploring or maintaining the old implementation from
that branch. New work should happen on v4.

## POC

The first v4 POC is **DepViz Board Brief**.

It currently supports:

- SQLite local state in `.depviz/state.db`
- JSONL/DepCrumb-style event ingest
- a default board
- local-only note cards
- manual dependency edges
- GitHub sync through `gh`
- ready/blocker queries
- morning `depviz brief`
- JSON export for tools and live mode
- single-file static HTML export
- stateless `depviz live` web app

The point of the POC is not a pretty graph. The point is a useful daily answer.

## Install

```text
make install
```

The binary is installed from:

```text
moul.io/depviz/v4/cmd/depviz
```

## Quickstart

```text
depviz init
depviz ingest events testdata/simple/events.jsonl
depviz board note default "Define POC scope"
depviz edge add note:define-poc-scope gh:moul/depviz2#47 --kind blocks
depviz brief
depviz gen json --out dist/depviz.json
depviz gen html --out dist/depviz.html
depviz live
```

For real GitHub data:

```text
depviz sync github owner/repo
depviz brief
```

GitHub auth uses the `gh` CLI. Run `gh auth login` first, or provide a
`GITHUB_TOKEN` through `gh`.

## Commands

```text
depviz init
depviz ingest events <path>
depviz sync github owner/repo [--limit 200]
depviz board list
depviz board note <board> <text>
depviz edge add <from> <to> --kind blocked_by
depviz query ready
depviz query blockers
depviz brief
depviz gen html --board default --view graph --out dist/depviz.html
depviz gen json --board default --out dist/depviz.json
depviz live --addr 127.0.0.1:8686
```

## Live Mode

`depviz live` serves a stateless browser app from the Go binary:

```text
make live
```

It accepts either:

- DepViz Flow, the concise Markdown-friendly human format
- the JSONL event format used by `depviz ingest events`
- the JSON export produced by `depviz gen json`

Example:

```depviz
repo moul/depviz

#679 "Bootstrap depviz v4" [open] @v4
#80 "Choose Flow syntax" [open] @live
#81 "Hydrate refs from GitHub" [open] @github
note flow "Design DepViz Flow"

#679 depends on #80, #81 and blocks #85
#156 depends on moul/depviz2#5252
#679 addresses flow
```

See [docs/DEPVIZ-FLOW.md](docs/DEPVIZ-FLOW.md).

In standalone Live, node definitions make the graph readable without fetching
GitHub. With GitHub sync/export, the same board can usually shrink to relation
lines like `#679 blocks #85`; GitHub owns titles, labels, and state.

The editor includes syntax highlighting for DepViz Flow, JSON, and JSONL.
Plain Flow and fenced Markdown blocks like ```` ```depviz ```` can both be
pasted directly.

Live can also refresh GitHub refs directly from the browser. This is the
backendless mode: it calls `api.github.com`, optionally with a token kept only
in `sessionStorage`, and updates titles, states, labels, owners, and URLs in
the current graph. PRs also pull review and CI/check signals when GitHub exposes
them, so Live can show compact badges for type, lifecycle, review, and CI state.
`Connect GitHub` opens GitHub's fine-grained token form with the read-only repo
permissions DepViz needs prefilled: metadata, issues, pull requests, checks, and
commit statuses. `Use copied token` then loads the generated token from the
clipboard. It is deliberately not a cache, sync backend, or full OAuth flow yet.
Refreshed refs are shown in the Brief summary even when closed cards are hidden,
and public refs fall back to unauthenticated GitHub reads if the current token
lacks scope.

Low-confidence or source-inferred edges appear as Suggested relations. `Focus`
highlights the edge and its endpoints in the graph; `Promote` writes an official
local relation back into the current input, so the decision survives share links
and exports.

Graph edges are selectable. Selecting one opens an edge inspector with endpoints,
authority, confidence, soft/official status, and captured evidence. Suggested
edges can be promoted or hidden from the inspector as well as from the suggestions
panel.

The graph view uses relation-aware placement: connected cards are arranged by
dependency direction, while unrelated visible cards are kept in a compact pool.
This keeps realistic imports, such as a hundred recent GitHub issues and PRs,
scan-friendly without changing the ready/blocker semantics.

The static files live under `live/app/` and are deployable as-is through
GitHub Pages. No Node.js build is required.

The Pages workflow publishes:

- `master` to `/live/`
- each open PR to `/previews/pr-N/live/`

PR previews are removed when the PR closes.

## Local State

Runtime state is ignored:

```text
.depviz/state.db
.depviz/cache/
.depviz/sync/
```

Reviewable project facts should be git-versioned when they become useful:

```text
.depviz/events.jsonl
.depviz/boards/*.toml
.depviz/views/*.toml
.depviz/notes/*.md
```

## Development

```text
make test
go vet ./...
```

Agents continuing project work should start with
[docs/AGENT-HANDOFF.md](docs/AGENT-HANDOFF.md).

Fixture dogfood:

```text
tmpdir=$(mktemp -d)
DEPVIZ_DB="$tmpdir/state.db" depviz init
DEPVIZ_DB="$tmpdir/state.db" depviz ingest events testdata/simple/events.jsonl
DEPVIZ_DB="$tmpdir/state.db" depviz brief
DEPVIZ_DB="$tmpdir/state.db" depviz gen json --out "$tmpdir/depviz.json"
DEPVIZ_DB="$tmpdir/state.db" depviz gen html --out "$tmpdir/depviz.html"
```

## License

Licensed under the [Apache License, Version 2.0](LICENSE-APACHE) or the
[MIT license](LICENSE-MIT), at your option. See [COPYRIGHT](COPYRIGHT) for
details.
