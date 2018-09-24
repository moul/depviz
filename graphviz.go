package main

import (
	"fmt"
	"math"
	"sort"

	"github.com/awalterschulze/gographviz"
	"go.uber.org/zap"
)

func graphviz(issues Issues, opts *runOptions) (string, error) {
	var (
		stats = map[string]int{
			"nodes":     0,
			"edges":     0,
			"hidden":    0,
			"subgraphs": 0,
		}
		invisStyle = map[string]string{"style": "invis", "label": escape("")}
		weightMap  = map[int]bool{}
		weights    = []int{}
	)
	if opts.DebugGraph {
		invisStyle = map[string]string{}
	}
	for _, issue := range issues {
		if issue.Hidden {
			stats["hidden"]++
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
	attrs["rankdir"] = "RL"
	attrs["constraint"] = "true"
	attrs["compound"] = "true"
	if !opts.NoCompress {
		attrs["center"] = "true"
		attrs["ranksep"] = "0.3"
		attrs["nodesep"] = "0.1"
		attrs["margin"] = "0.2"
		attrs["sep"] = "-0.7"
		attrs["constraint"] = "false"
		attrs["splines"] = "true"
		attrs["overlap"] = "compress"
	}
	if opts.DarkTheme {
		attrs["bgcolor"] = "black"
	}

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
		parent := fmt.Sprintf("cluster_weight_%d", issue.Weight())
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
		stats["nodes"]++
	}

	// issue relationships
	for _, issue := range issues {
		panicIfErr(issue.AddEdgesToGraph(g))
		stats["edges"]++
	}

	// orphans cluster and placeholder
	if orphansWithoutLinks > 0 {
		panicIfErr(g.AddSubGraph(
			"G",
			"cluster_orphans_without_links",
			map[string]string{"label": escape("orphans without links"), "style": "dashed"},
		))
		stats["subgraphs"]++

		panicIfErr(g.AddSubGraph(
			"cluster_orphans_without_links",
			"cluster_orphans_without_links_0",
			invisStyle,
		))
		stats["subgraphs"]++
		for i := 0; i < orphansCols; i++ {
			panicIfErr(g.AddNode(
				fmt.Sprintf("cluster_orphans_without_links_%d", i),
				fmt.Sprintf("placeholder_orphans_without_links_%d", i),
				invisStyle,
			))
			stats["nodes"]++
		}

		panicIfErr(g.AddEdge(
			fmt.Sprintf("placeholder_%d", weights[len(weights)-1]),
			"placeholder_orphans_without_links_0",
			true,
			invisStyle,
		))
		stats["edges"]++

		for i := 1; i < orphansCols; i++ {
			panicIfErr(g.AddSubGraph(
				"cluster_orphans_without_links",
				fmt.Sprintf("cluster_orphans_without_links_%d", i),
				invisStyle,
			))
			stats["subgraphs"]++
			panicIfErr(g.AddEdge(
				fmt.Sprintf("placeholder_orphans_without_links_%d", i-1),
				fmt.Sprintf("placeholder_orphans_without_links_%d", i),
				true,
				invisStyle,
			))
			stats["edges"]++
		}
	}
	if hasOrphansWithLinks {
		attrs := map[string]string{}
		attrs["label"] = escape("orphans with links")
		attrs["style"] = "dashed"
		panicIfErr(g.AddSubGraph("G", "cluster_orphans_with_links", attrs))
		stats["subgraphs"]++

		panicIfErr(g.AddNode("cluster_orphans_with_links", "placeholder_orphans_with_links", invisStyle))
		stats["nodes"]++

		panicIfErr(g.AddEdge(
			"placeholder_orphans_with_links",
			fmt.Sprintf("placeholder_%d", weights[0]),
			true,
			invisStyle,
		))
		stats["edges"]++
	}

	// set weights clusters and placeholders
	for _, weight := range weights {
		clusterName := fmt.Sprintf("cluster_weight_%d", weight)
		attrs := invisStyle
		attrs["rank"] = "same"
		panicIfErr(g.AddSubGraph("G", clusterName, attrs))
		stats["subgraphs"]++

		attrs = invisStyle
		attrs["shape"] = "none"
		attrs["label"] = fmt.Sprintf(`"weight=%d"`, weight)
		panicIfErr(g.AddNode(
			clusterName,
			fmt.Sprintf("placeholder_%d", weight),
			attrs,
		))
		stats["nodes"]++
	}
	for i := 0; i < len(weights)-1; i++ {
		panicIfErr(g.AddEdge(
			fmt.Sprintf("placeholder_%d", weights[i]),
			fmt.Sprintf("placeholder_%d", weights[i+1]),
			true,
			invisStyle,
		))
		stats["edges"]++
	}

	logger().Debug("graph stats", zap.Any("stats", stats))
	return g.String(), nil
}
