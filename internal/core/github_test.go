package core

import "testing"

func TestExtractDependencyEdgesSkipsHTMLEntities(t *testing.T) {
	edges := extractDependencyEdges("moul/depviz", "gh:moul/depviz#1", "blocks &#8203;\nblocks #2")
	if len(edges) != 1 {
		t.Fatalf("edges = %+v, want one real GitHub ref", edges)
	}
	if edges[0].To != "gh:moul/depviz#2" {
		t.Fatalf("edge target = %s, want gh:moul/depviz#2", edges[0].To)
	}
}
