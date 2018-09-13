package main

import (
	"fmt"
	"sort"

	"github.com/awalterschulze/gographviz"
)

func orphansGraph(issues Issues, opts *runOptions) (string, error) {
	if !opts.ShowClosed {
		issues.HideClosed()
	}
	issues.processEpicLinks()

	g := gographviz.NewGraph()
	panicIfErr(g.SetName("G"))
	attrs := map[string]string{}
	attrs["truecolor"] = "true"
	attrs["overlap"] = "false"
	attrs["splines"] = "true"
	attrs["rankdir"] = "RL"
	attrs["ranksep"] = "0.3"
	attrs["nodesep"] = "0.1"
	attrs["margin"] = "0.2"
	attrs["center"] = "true"
	attrs["constraint"] = "false"
	for k, v := range attrs {
		panicIfErr(g.AddAttr("G", k, v))
	}
	panicIfErr(g.SetDir(true))

	repos := map[string]string{}
	for _, issue := range issues {
		if !issue.IsOrphan && issue.LinkedWithEpic {
			issue.Hidden = true
		}
		if issue.Hidden {
			continue
		}
		panicIfErr(issue.AddNodeToGraph(g, fmt.Sprintf("cluster_%s", issue.RepoID())))
		repos[issue.RepoID()] = issue.Path()
	}

	for id, repo := range repos {
		panicIfErr(g.AddSubGraph(
			"G",
			fmt.Sprintf("cluster_%s", id),
			map[string]string{"label": escape(repo), "style": "dashed"},
		))
	}

	// issue relationships
	for _, issue := range issues {
		panicIfErr(issue.AddEdgesToGraph(g))
	}

	// FIXME: sub-cluster by state

	return g.String(), nil
}

func graphviz(issues Issues, opts *runOptions) (string, error) {
	var (
		invisStyle = map[string]string{"style": "invis"}
		weightMap  = map[int]bool{}
		weights    = []int{}
	)
	for _, issue := range issues {
		if issue.Hidden {
			continue
		}
		weightMap[issue.Weight()] = true
	}
	for weight := range weightMap {
		weights = append(weights, weight)
	}
	sort.Ints(weights)

	// initialize graph
	g := gographviz.NewGraph()
	panicIfErr(g.SetName("G"))
	attrs := map[string]string{}
	attrs["truecolor"] = "true"
	attrs["overlap"] = "false"
	attrs["splines"] = "true"
	attrs["rankdir"] = "RL"
	attrs["ranksep"] = "0.3"
	attrs["nodesep"] = "0.1"
	attrs["margin"] = "0.2"
	attrs["center"] = "true"
	attrs["constraint"] = "false"
	for k, v := range attrs {
		panicIfErr(g.AddAttr("G", k, v))
	}
	panicIfErr(g.SetDir(true))

	// issue nodes
	issueNumbers := []string{}
	for _, issue := range issues {
		issueNumbers = append(issueNumbers, issue.URL)
	}
	sort.Strings(issueNumbers)
	for _, id := range issueNumbers {
		issue := issues[id]
		if issue.Hidden {
			continue
		}
		parent := fmt.Sprintf("anon%d", issue.Weight())
		if issue.IsOrphan || !issue.LinkedWithEpic {
			parent = "cluster_orphans"
		}

		panicIfErr(issue.AddNodeToGraph(g, parent))
	}

	// issue relationships
	for _, issue := range issues {
		panicIfErr(issue.AddEdgesToGraph(g))
	}

	// orphans cluster and placeholder
	if issues.HasOrphans() {
		panicIfErr(g.AddSubGraph("G", "cluster_orphans", map[string]string{"label": "orphans", "style": "dashed"}))
		panicIfErr(g.AddNode("cluster_orphans", "placeholder_orphans", invisStyle))
		panicIfErr(g.AddEdge(
			fmt.Sprintf("placeholder_%d", weights[len(weights)-1]),
			"placeholder_orphans",
			true,
			invisStyle,
		))
	}

	// set weights clusters and placeholders
	for _, weight := range weights {
		clusterName := fmt.Sprintf("anon%d", weight)
		panicIfErr(g.AddSubGraph("G", clusterName, map[string]string{"rank": "same"}))
		//clusterName := fmt.Sprintf("cluster_w%d", weight)
		//panicIfErr(g.AddSubGraph("G", clusterName, map[string]string{"label": fmt.Sprintf("w%d", weight)}))
		panicIfErr(g.AddNode(
			clusterName,
			fmt.Sprintf("placeholder_%d", weight),
			map[string]string{
				"shape": "none",
				"label": fmt.Sprintf(`"weight=%d"`, weight),
			},
		))
	}
	for i := 0; i < len(weights)-1; i++ {
		panicIfErr(g.AddEdge(
			fmt.Sprintf("placeholder_%d", weights[i]),
			fmt.Sprintf("placeholder_%d", weights[i+1]),
			true,
			invisStyle,
		))
	}

	return g.String(), nil
}
