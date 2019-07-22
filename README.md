<h1 align="center">
  <br>
  <img src="https://raw.githubusercontent.com/moul/depviz/master/assets/depviz.svg?sanitize=true" alt="depviz" height="140px">
  <br>
  <br>
  DepViz
  <br>
</h1>

<h3 align="center">👓 Issue dependency visualizer, a.k.a. "auto-roadmap".</h3>

<p align="center"><b>
    <a href="https://manfred.life/depviz">Manfred.life</a> •
    <a href="https://twitter.com/moul">Twitter</a>
</b></p>

[![CircleCI](https://circleci.com/gh/moul/depviz.svg?style=shield)](https://circleci.com/gh/moul/depviz)
[![GoDoc](https://godoc.org/moul.io/depviz?status.svg)](https://godoc.org/moul.io/depviz)
[![License](https://img.shields.io/github/license/moul/depviz.svg)](https://github.com/moul/depviz/blob/master/LICENSE)
[![GitHub release](https://img.shields.io/github/release/moul/depviz.svg)](https://github.com/moul/depviz/releases)
[![Go Report Card](https://goreportcard.com/badge/moul.io/depviz)](https://goreportcard.com/report/moul.io/depviz)
[![Docker Metrics](https://images.microbadger.com/badges/image/moul/depviz.svg)](https://microbadger.com/images/moul/depviz)
[![Made by Manfred Touron](https://img.shields.io/badge/made%20by-Manfred%20Touron-blue.svg?style=flat)](https://manfred.life/)

## Introduction
dependency visualizer (auto roadmap)

**work in progress**: I'm already using this tool on a daily basis, but I know it lacks a lot of work to make it cool for other people too

`depviz` aggregates issues from multiple repositories and outputs a visual representation of the dependencies.

_inspired by this discussion: [jbenet/random-ideas#37](https://github.com/jbenet/random-ideas/issues/37)_

## Example

![](https://raw.githubusercontent.com/moul/depviz/master/examples/depviz/depviz.svg?sanitize=true)

## Install (with Golang)

```
go get moul.io/depviz
```

## Usage

```console
$ export GITHUB_TOKEN=xxxx

# render and display the roadmap
$ depviz run moul/depviz | dot -Tpng > depviz-roadmap.png
$ open depviz-roadmap.png

# render and display the orphans
$ depviz run moul/depviz --show-orphans | dot -Tpng > depviz-orphans.png
$ open depviz-orphans.png
```

### Preview image withing iterm2

```console
# install imgcat
$ go get github.com/olivere/iterm2-imagetools/cmd/imgcat
$ depviz run https://github.com/moul/depviz/issues/42 | dot -Tpng | imgcat
```

![](https://raw.githubusercontent.com/moul/depviz/master/examples/imgcat.png)

## License

Apache
