package pmbodyparser

import (
	"fmt"
	"testing"

	"moul.io/depviz/v3/pkg/multipmuri"
)

func ExampleRelParseString() {
	body := `
This PR fixes a lot of things and implement plenty new features.

Addresses #42
Depends on: github.com/moul/depviz#42
Blocks #45
Block: #46
fixes: #58
FIX github.com/moul/depviz#1337

Signed-off-by: Super Developer <super.dev@gmail.com>
`
	relationships, errs := RelParseString(
		multipmuri.NewGitHubIssue("github.com", "moul", "depviz", "1"),
		body,
	)
	if len(errs) > 0 {
		panic(errs)
	}
	for _, relationship := range relationships {
		fmt.Println(relationship)
	}
	// Output:
	// addresses https://github.com/moul/depviz/issues/42
	// blocks https://github.com/moul/depviz/issues/45
	// blocks https://github.com/moul/depviz/issues/46
	// depends-on https://github.com/moul/depviz/issues/42
	// fixes https://github.com/moul/depviz/issues/1337
	// fixes https://github.com/moul/depviz/issues/58
}

func ExampleParseString() {
	rels, errs := ParseString("Depends on github.com/moul/depviz#1")
	if len(errs) > 0 {
		panic(errs)
	}
	for _, rel := range rels {
		fmt.Println(rel)
	}
	// Output:
	// depends-on https://github.com/moul/depviz/issues/1
}

func TestRelParseString(t *testing.T) {
	var tests = []struct {
		name              string
		base              multipmuri.Entity
		body              string
		expectedErrsCount int
		expectedRels      Relationships
	}{
		{
			"simple",
			multipmuri.NewGitHubIssue("", "moul", "depviz", "1"),
			"Depends on #2",
			0,
			Relationships{
				{Kind: DependsOn, Target: multipmuri.NewGitHubIssue("", "moul", "depviz", "2")},
			},
		}, {
			"multiple",
			multipmuri.NewGitHubIssue("", "moul", "depviz", "1"),
			"Depends on #2\nDepends on #3",
			0,
			Relationships{
				{Kind: DependsOn, Target: multipmuri.NewGitHubIssue("", "moul", "depviz", "2")},
				{Kind: DependsOn, Target: multipmuri.NewGitHubIssue("", "moul", "depviz", "3")},
			},
		}, {
			"with-spaces",
			multipmuri.NewGitHubIssue("", "moul", "depviz", "1"),
			" Depends on #2 \n Depends on #3 \n\n ",
			0,
			Relationships{
				{Kind: DependsOn, Target: multipmuri.NewGitHubIssue("", "moul", "depviz", "2")},
				{Kind: DependsOn, Target: multipmuri.NewGitHubIssue("", "moul", "depviz", "3")},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rels, errs := RelParseString(test.base, test.body)
			if test.expectedErrsCount != len(errs) {
				t.Errorf("Expected %d errs, got %d.", test.expectedErrsCount, len(errs))
			}

			expectedStr := fmt.Sprintf("%v", test.expectedRels)
			gotStr := fmt.Sprintf("%v", rels)
			if expectedStr != gotStr {
				t.Errorf("Expected %s, got %s.", expectedStr, gotStr)
			}
		})
	}
}
