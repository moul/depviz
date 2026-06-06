# DepViz Flow

DepViz Flow is the human input format for `depviz live`.

JSON/JSONL remains the machine format. Flow is optimized for editing by hand,
copying into Markdown, and growing from one repo to many repos without changing
the first lines you wrote.

The Live editor accepts the fenced Markdown form directly, so a block from an
issue, PR, README, or HackMD note can be pasted without removing the fences.

## Example

```depviz
depviz LR
board "DepViz v4 POC"

repo moul/depviz
repo moul/depviz2 as d2

#679 "Bootstrap v4 root" [open] @v4
d2#10 "Old POC PR" [closed] @poc
note flow "Design DepViz Flow"

flow -> #679
d2#10 -> #679
gh:openai/codex#123 ~> #679
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

flow -> release
```

## Edges

Edges read left-to-right as prerequisite to result:

```depviz
A -> B
```

means `A` blocks or unlocks `B`.

Use reverse form when it reads better:

```depviz
B <- A
```

Use soft edges for suspected dependencies:

```depviz
A ~> B
```

## Nodes

```depviz
#123 "Issue title" [open] @label +owner
!45 "PR title" [merged] @release
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
