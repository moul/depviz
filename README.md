# Depviz

<h1 align="center">
  <img src="https://raw.githubusercontent.com/moul/depviz/master/assets/depviz.svg?sanitize=true" alt="Depviz" title="Depviz" height="200px">
  <br>
</h1>

<h3 align="center">👓 Issue dependency visualizer, a.k.a. "auto-roadmap".</h3>

[![GoDoc](https://img.shields.io/static/v1?label=godoc&message=reference&color=blue)](https://pkg.go.dev/moul.io/depviz/v3)
[![License](https://img.shields.io/badge/license-Apache--2.0%20%2F%20MIT-%2397ca00.svg)](https://github.com/moul/depviz/blob/master/COPYRIGHT)
[![GitHub release](https://img.shields.io/github/release/moul/depviz.svg)](https://github.com/moul/depviz/releases)
[![Go Report Card](https://goreportcard.com/badge/moul.io/depviz)](https://goreportcard.com/report/moul.io/depviz)
[![CodeFactor](https://www.codefactor.io/repository/github/moul/depviz/badge)](https://www.codefactor.io/repository/github/moul/depviz)
[![Docker Metrics](https://images.microbadger.com/badges/image/moul/depviz.svg)](https://microbadger.com/images/moul/depviz)
[![GolangCI](https://golangci.com/badges/github.com/moul/depviz.svg)](https://golangci.com/r/github.com/moul/depviz)
[![Made by Manfred Touron](https://img.shields.io/badge/made%20by-Manfred%20Touron-blue.svg?style=flat)](https://manfred.life/)

<!-- [![codecov](https://codecov.io/gh/moul/depviz/branch/master/graph/badge.svg)](https://codecov.io/gh/moul/depviz) -->

## Introduction

dependency visualizer (auto roadmap)

`depviz` aggregates **tasks** from multiple projects and generates visual representations (graphs) of the dependencies.

_inspired by this discussion: [jbenet/random-ideas#37](https://github.com/jbenet/random-ideas/issues/37)_

## Philosophy

The ultimate goal of this tool is to allow the tech and the non-tech to collaborate seamlessly.

Oftentimes, there are “non-technical project managers” that love tools like Jira and try to define everything, including the delay required.
Developers, however, mostly hate Jira-like tools and prefer to focus on small tasks with an easy-to-use interface, like Trello, GitHub issues, GitLab issues.

The idea of depviz is to:

* link those different tools (aggregate the different sources and find the relationships: find that this exact “Jira user story” belongs to those 5 technical issues on github
* create various visual ways of displaying this information. Then, we can have a company that has some non-technical project manager only focusing on user stories and their priorities, and devs that focus on tasks and estimate the tasks by themselves (everyone doing what they are good at)
* in general, help everyone have the overall vision more clear

## Target

* Graphs are “fun” but not very useful yet, a good dependency tool would be like graphviz. The current depviz version makes the graph in something that is more “weight-based”, because nodes will be grouped to make the graph fit the screen. Graphviz is not focused on making things beautiful, but focused on being 100% clear on the dependency. We need a good graph driver that supports this kind of graph.
* Having options for multiple layouts/graphs.
* Implementing the [PERT method](https://en.wikipedia.org/wiki/Program_evaluation_and_review_technique) and adding more fields in depviz: due date, difficulty, etc, in order to create graphs for “finding the shortest path”, for example.
* Improving the UI to improve collaboration (sharing a URL, etc).

## Demo

https://depviz-demo.moul.io/

_Limited to the following repos: [moul/depviz](https://github.com/moul/depviz), [moul/depviz-test](https://github.com/moul/depviz-test), [moul-bot/depviz-test](https://github.com/moul-bot/depviz-test)._

## Supported providers

_Depviz_ aggregates the entities of multiple providers into 3 generic ones.

---

Supported providers:

* GitHub
  * Task: Issue, Pull Request, Milestone
  * Owner: TODO
  * Topic: TODO
* GitLab: _(planned)_
* Jira _(planned)_
* Trello _(planned)_

TODO: detailed mapping table

## Under the hood

### Depviz entities

There are 3 entities:

* A `Task` that have a real life cycle: opened->closed
* An `Owner` which only contains things
* A `Topic` which allows categorizing/tagging other things

**Examples**:

* a `Milestone` is a `Depviz Task`, because even if it contains other tasks, it also has a well defined lifecycle: to be closed when every children tasks are finished.
* a `Repository` is a `Depviz Owner` because even if you can archive a repository, it's not the normal lifecycle, and will most of the time be unrelated with the amount of tasks done

A `Task` can be considered as something directly actionable, or indirectly/automatically closable based on a business rule.

**More info here: [./api/dvmodel.proto](./api/dvmodel.proto)**

#### Task

should have:

* a unique `ID`: canonical URL
* a `LocalID`: human-readable identifier
* a `Title`: _not necessarily unique_
* a `Kind`: `Issue`, `Pull Request`, `Milestone`, `Epic`, `Story`, `Card`
* a `State`: `opened`, `in progress`, or `closed`
* an `Owner`: _see below_
* a `Driver`: `GitHub`, `GitLab`, `Jira`, `Trello`

may have:

* other relationships: `Author`, `Milestone`, `Assignees`, `Reviewers`, `Label`, `Dependencies`, `Dependents`, `Related`, `Parts`, `Parents`
* other metadata: `Description`
* other states: `Locked`
* timestamps: `Created`, `Updated`, `Due`, `Completed`
* metrics: `NumDownvotes`, `NumUpvotes`, `NumComments`

#### Owner

should have:

* a unique `ID`: canonical URL
* a `LocalID`: human-readable identifier
* a `Title`: _not necessarily unique_
* a `Kind`: `User`, `Organization`, `Team`, `Repo`, `Provider`
* a `Driver`: `GitHub`, `GitLab`, `Jira`, `Trello`

may have:

* an `Owner`
* other states: `Fork`
* other metadata: `Homepage`, `Description`, `Avatar`, `Fullname`, `Shortname`
* timestamps: `Created`, `Updated`

#### Topic

should have:

* a unique `ID`: canonical URL
* a `LocalID`: human-readable identifier
* a `Title`: _not necessarily unique_
* a `Kind`: `Label`
* a `Driver`: `GitHub`, `GitLab`, `Jira`, `Trello`

may have:

* an `Owner`: _see above_
* other metadata: `Color`, `Description`

## Install

### Download a release

https://github.com/moul/depviz/releases

### Install With Golang

```bash
go get moul.io/depviz/cmd/depviz/v3
```

### Using brew

```bash
brew install moul/moul/depviz
```

## Usage

TODO

## License

© 2018-2021 [Manfred Touron](https://manfred.life)

Licensed under the [Apache License, Version 2.0](https://www.apache.org/licenses/LICENSE-2.0) ([`LICENSE-APACHE`](LICENSE-APACHE)) or the [MIT license](https://opensource.org/licenses/MIT) ([`LICENSE-MIT`](LICENSE-MIT)), at your option. See the [`COPYRIGHT`](COPYRIGHT) file for more details.

`SPDX-License-Identifier: (Apache-2.0 OR MIT)`
