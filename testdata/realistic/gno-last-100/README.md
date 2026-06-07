# Gno Last 100 Snapshot

This fixture is a realistic GitHub sync snapshot for `gnolang/gno`.

Captured from:

```text
depviz sync github gnolang/gno --limit 50
```

`--limit 50` imports up to 50 issues and 50 pull requests, so this represents
100 GitHub cards plus placeholder nodes created from referenced external refs.

Captured at `2026-06-07T22:10:55Z`.

Files:

- `export.json`: full DepViz export accepted by Live mode
- `brief.txt`: rendered brief from the same snapshot
- `summary.json`: small review-friendly summary of counts and inferred edges

The fixture is intentionally frozen. Do not refresh it automatically in tests;
create a new dated fixture when GitHub behavior or parser behavior needs a new
real-world sample.
