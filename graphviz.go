package main

import (
	"fmt"
	"math"
	"sort"

	"github.com/awalterschulze/gographviz"
)

func graphviz(issues Issues, opts *runOptions) (string, error) {
	var (
		invisStyle = map[string]string{"style": "invis", "label": escape("")}
		weightMap  = map[int]bool{}
		weights    = []int{}
	)
	if opts.DebugGraph {
		invisStyle = map[string]string{}
	}
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

	orphansWithoutLinks := 0
	for _, id := range issueNumbers {
		issue := issues[id]
		if issue.Hidden {
			continue
		}
		if len(issue.DependsOn) == 0 && len(issue.Blocks) == 0 {
			orphansWithoutLinks++
		}
	}
	orphansCols := int(math.Ceil(math.Sqrt(float64(orphansWithoutLinks)) / 2))
	colIndex := 0
	hasOrphansWithLinks := false
	for _, id := range issueNumbers {
		issue := issues[id]
		if issue.Hidden {
			continue
		}
		parent := fmt.Sprintf("anon%d", issue.Weight())
		if issue.IsOrphan || !issue.LinkedWithEpic {
			if len(issue.DependsOn) > 0 || len(issue.Blocks) > 0 {
				parent = "cluster_orphans_with_links"
				hasOrphansWithLinks = true
			} else {
				parent = fmt.Sprintf("cluster_orphans_without_links_%d", colIndex%orphansCols)
				colIndex++
			}
		}

		panicIfErr(issue.AddNodeToGraph(g, parent))
	}

	// issue relationships
	for _, issue := range issues {
		panicIfErr(issue.AddEdgesToGraph(g))
	}

	// orphans cluster and placeholder
	if orphansWithoutLinks > 0 {
		panicIfErr(g.AddSubGraph(
			"G",
			"cluster_orphans_without_links",
			map[string]string{"label": escape("orphans without links"), "style": "dashed"},
		))

		panicIfErr(g.AddSubGraph(
			"cluster_orphans_without_links",
			"cluster_orphans_without_links_0",
			invisStyle,
		))
		for i := 0; i < orphansCols; i++ {
			panicIfErr(g.AddNode(
				fmt.Sprintf("cluster_orphans_without_links_%d", i),
				fmt.Sprintf("placeholder_orphans_without_links_%d", i),
				invisStyle,
			))
		}

		panicIfErr(g.AddEdge(
			fmt.Sprintf("placeholder_%d", weights[len(weights)-1]),
			"placeholder_orphans_without_links_0",
			true,
			invisStyle,
		))

		for i := 1; i < orphansCols; i++ {
			panicIfErr(g.AddSubGraph(
				"cluster_orphans_without_links",
				fmt.Sprintf("cluster_orphans_without_links_%d", i),
				invisStyle,
			))
			panicIfErr(g.AddEdge(
				fmt.Sprintf("placeholder_orphans_without_links_%d", i-1),
				fmt.Sprintf("placeholder_orphans_without_links_%d", i),
				true,
				invisStyle,
			))
		}
	}
	if hasOrphansWithLinks {
		panicIfErr(g.AddSubGraph("G", "cluster_orphans_with_links", map[string]string{"label": escape("orphans with links"), "style": "dashed"}))
		panicIfErr(g.AddNode("cluster_orphans_with_links", "placeholder_orphans_with_links", invisStyle))
		panicIfErr(g.AddEdge(
			"placeholder_orphans_with_links",
			fmt.Sprintf("placeholder_%d", weights[0]),
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
