# DepViz v4

DepViz v4 is the local-first work graph rewrite.

The v3 code still lives at the repository root. v4 starts here as a separate
Go module so it can become useful without breaking the existing release line.
Once v4 is mature, it can replace the old implementation as the main `depviz`.

## What v4 proves first

The first POC is **DepViz Board Brief**:

- SQLite local state in `.depviz/state.db`
- JSONL/DepCrumb-style event ingest
- default board
- local-only note cards
- manual dependency edges
- GitHub sync through `gh`
- ready/blocker queries
- morning `depviz brief`
- single-file static HTML export

The point is not to show a pretty graph. The point is to answer:

```text
What can move now?
What is blocked?
What should happen next?
```

## Quickstart

```text
make test
make install
depviz init
depviz ingest events testdata/simple/events.jsonl
depviz board note default "Define POC scope"
depviz edge add note:define-poc-scope gh:moul/depviz2#47 --kind blocks
depviz brief
depviz gen html --out dist/depviz.html
```

For real GitHub data:

```text
depviz sync github owner/repo
depviz brief
```

GitHub auth uses `gh auth token` or `GITHUB_TOKEN` through the `gh` CLI.

## Versioning intent

- v3 remains the existing tagged product line.
- v4 lives under `v4/` with module path `moul.io/depviz/v4`.
- When v4 is ready, tag a final v3 release, make v4 the default story, and
  retire or archive the old implementation.
