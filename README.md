<h1 align="center">
  <br>
  <img src="https://raw.githubusercontent.com/moul/depviz/master/assets/depviz.svg?sanitize=true" alt="depviz" height="60px">
  <br>
  <br>
  DepViz
  <br>
</h1>

<h3 align="center">👓 Issue dependency visualizer, a.k.a. "auto-roadmap".</h3>

<p align="center">
  <a href="https://circleci.com/gh/moul/depviz">
    <img src="https://circleci.com/gh/moul/depviz.svg?style=shield"
         alt="Build Status">
  </a>
  <a href="https://goreportcard.com/report/moul.io/depviz">
    <img src="https://goreportcard.com/badge/moul.io/depviz"
         alt="Go Report Card">
  </a>
  <a href="https://github.com/moul/depviz/releases">
    <img src="https://badge.fury.io/gh/moul%2Fdepviz.svg"
         alt="GitHub version">
  </a>
  <a href="https://godoc.org/moul.io/depviz">
    <img src="https://godoc.org/moul.io/depviz?status.svg"
         alt="GoDoc">
  </a>
</p>

<p align="center"><b>
    <a href="https://moul.io/depviz">Website</a> •
    <a href="https://twitter.com/moul">Twitter</a>
</b></p>

## Introduction
dependency visualizer (auto roadmap)

**work in progress**: I'm already using this tool on a daily basis, but I know it lacks a lot of work to make it cool for other people too

`depviz` aggregates issues from multiple repositories and outputs a visual representation of the dependencies.

_inspired by this discussion: [jbenet/random-ideas#37](https://github.com/jbenet/random-ideas/issues/37)_

## Example

![](https://raw.githubusercontent.com/moul/depviz/master/examples/depviz/depviz.svg?sanitize=true)

## Install (with Golang)

`go get moul.io/depviz`

## Usage

```console
$ export GITHUB_TOKEN=xxxx

# render and display the roadmap
$ depviz render --repos=moul/depviz | dot -Tpng > depviz-roadmap.png
$ open depviz-roadmap.png

# render and display the orphans
$ depviz render --repos=moul/depviz -t orphans | dot -Tpng > depviz-orphans.png
$ open depviz-orphans.png
```

### Preview image withing iterm2

```console
# install imgcat
$ go get github.com/olivere/iterm2-imagetools/cmd/imgcat
$ depviz render | dot -Tpng | imgcat
```

![](https://raw.githubusercontent.com/moul/depviz/master/examples/imgcat.png)

## License

Apache
