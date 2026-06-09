# DepViz Flow

DepViz Flow is the human input format for `depviz live`.

JSON/JSONL remains the machine format. Flow is optimized for writing dependency
intent by hand, copying into Markdown, and growing from one repo to many repos
without changing the first lines you wrote.

The Live editor guesses the input format. Plain Flow, JSONL, exported JSON, and
Markdown code fences are all accepted without a mode switch.

## Example

GitHub-connected boards can stay relation-only:

```depviz
repo moul/depviz

#679 depends on #80, #81 and blocks #85
#156 depends on moul/depviz2#5252
```

GitHub titles, labels, owners, and states come from sync/exported JSON in that
mode. Flow is mostly the relationship layer.

Standalone Live examples can define nodes by hand so the graph has readable
cards before any sync exists:

```depviz
repo moul/depviz

#679 "Bootstrap depviz v4" [open] @v4
#80 "Choose Flow syntax" [open] @live

#679 depends on #80
```

## References

When a default repo is set:

```depviz
repo moul/depviz

#679
!42
```

resolves to:

```text
gh:moul/depviz#679
gh:moul/depviz!42
```

External refs can be canonical or aliased:

```depviz
gh:moul/depviz2#10
moul/depviz2#10

repo moul/depviz2 as d2
d2#10
```

Local-only refs are explicit:

```depviz
note flow "Design DepViz Flow"
task release "Prepare v4 release"

flow blocks release
```

## Relations

Relations use verbs because they read well in issue comments and PR
descriptions:

```depviz
#679 depends on #80, #81
#679 blocks #85
#679 addresses flow
#679 closes #85
```

`depends on` means each target blocks the subject. `blocks` means the subject
blocks each target. `addresses`, `mentions`, `relates to`, and `closes` record
non-blocking relationships.

Lists can use commas and `and`:

```depviz
#679 depends on #80, #81 and blocks #85
```

The parser still accepts arrow aliases for imported snippets, but the human
format should prefer verbs.

## Node Definitions

GitHub refs can be declared manually when the input is the whole demo or note:

```depviz
#123 "Issue title" [open] @label +owner
!45 "PR title" [merged] @release
```

When GitHub is connected, prefer omitting those definitions unless you are
adding local-only context. The synced source owns external state.

Local-only refs are explicit:

```depviz
note slug "Local note"
task slug "Local task"
```

## Markdown Shape

Flow should stay pleasant when it is rendered by plain Markdown:

- every useful example fits in a `depviz` fenced code block
- one line should usually mean one node, one edge, or one setting
- comments use `# comment` or `// comment`
- short refs are for the current repo; canonical refs are stored internally
- aliases are local to the block, so examples can stay compact
- GitHub state is written by hand only in standalone Live snippets
- connected data owns external truth when a GitHub sync/export is present
- source-inferred relations should stay visibly softer than curated DepViz
  relations and should not drive ready/blocker decisions until a human
  promotes them

That makes the same snippet usable in a GitHub issue, a PR description, a
HackMD note, a README, or the Live editor.

## Parser Roadmap

The v1 parser is intentionally small. Future parser work should optimize for
clarity before cleverness:

1. grow the language through golden examples, not hidden heuristics
2. keep the Markdown rendering readable without custom CSS
3. add a shared grammar or shared fixtures before the CLI and Live parser diverge
4. make ref resolution two-pass so local aliases and repo aliases can be used
   before they are declared
5. preserve concise syntax for the 80% case, then add explicit canonical syntax
   for multi-repo and automation-heavy boards
6. return line/column diagnostics that are good enough to edit live
