package core

import "testing"

func TestExtractDependencyEdgesParsesRelationChunks(t *testing.T) {
	body := `
depends on #2, moul/depviz2#3 and blocks !4
addresses https://github.com/moul/depviz/issues/5
relates to gh:moul/depviz#6
closes #7
`
	edges := extractDependencyEdges("moul/depviz", "gh:moul/depviz#1", body)
	want := []extractedEdge{
		{From: "gh:moul/depviz#1", To: "gh:moul/depviz#2", Kind: "blocked_by", Confidence: 0.75},
		{From: "gh:moul/depviz#1", To: "gh:moul/depviz2#3", Kind: "blocked_by", Confidence: 0.75},
		{From: "gh:moul/depviz#1", To: "gh:moul/depviz!4", Kind: "blocks", Confidence: 0.75},
		{From: "gh:moul/depviz#1", To: "gh:moul/depviz#5", Kind: "addresses", Confidence: 0.55},
		{From: "gh:moul/depviz#1", To: "gh:moul/depviz#6", Kind: "relates_to", Confidence: 0.55},
		{From: "gh:moul/depviz#1", To: "gh:moul/depviz#7", Kind: "closes", Confidence: 0.7},
	}
	if len(edges) != len(want) {
		t.Fatalf("edges = %+v, want %+v", edges, want)
	}
	for i := range want {
		if edges[i].From != want[i].From || edges[i].To != want[i].To || edges[i].Kind != want[i].Kind || edges[i].Confidence != want[i].Confidence {
			t.Fatalf("edge %d = %+v, want %+v", i, edges[i], want[i])
		}
	}
}

func TestExtractDependencyEdgesSkipsHTMLEntities(t *testing.T) {
	edges := extractDependencyEdges("moul/depviz", "gh:moul/depviz#1", "blocks &#8203;\nblocks #2")
	if len(edges) != 1 {
		t.Fatalf("edges = %+v, want one real GitHub ref", edges)
	}
	if edges[0].To != "gh:moul/depviz#2" {
		t.Fatalf("edge target = %s, want gh:moul/depviz#2", edges[0].To)
	}
}
