# depviz
dependency visualizer (auto roadmap)

`depviz` aggregates issues from multiple repositories and outputs a visual representation of the dependencies.

## Example

![](https://raw.githubusercontent.com/moul/depviz/master/examples/depviz/depviz.svg?sanitize=true)

## Install (with Golang)

`go get moul.io/depviz`

## Usage

```console
$ export GITHUB_TOKEN=xxxx
$ depviz render --repos=moul/depviz | dot -Tpng > depviz-roadmap.png
$ open depviz-roadmap.png
$ depviz render --repos=moul/depviz -t orphans | dot -Tpng > depviz-orphans.png
$ open depviz-orphans.png
```

## License

Apache
