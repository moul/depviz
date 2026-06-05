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

- the JSONL event format used by `depviz ingest events`
- the JSON export produced by `depviz gen json`

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
