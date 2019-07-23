package issues

import "fmt"

func ExampleParseTargets() {
	inputs := []string{}
	for _, input := range []string{
		"moul/project1",
		"moul/project2#42",
		"github.com/moul/project3",
		"https://github.com/moul/project4/issues/42",
		"https://gitlab.com/moul/project5#42",
		"gitlab.com/moul/project6",
		"forge.company.com/a/b",
		"gitlab.company.com/c/d",
		"github.company.com/d/e",
		"jira.company.com/browse/CP-42",
		"https://jira.company.com/browse/CP",
		"https://jira.company.com/browse/CP-42",
	} {
		for _, prefix := range []string{"", "github://", "gitlab://", "jira://"} { // FIXME: support github-enterprise://
			inputs = append(inputs, prefix+input)
		}
	}

	targets, _ := ParseTargets(inputs)
	for _, target := range targets {
		fmt.Printf(
			"target=%q\n  canonical=%q\n  project=%q\n  namespace=%q\n  providerurl=%q\n  driver=%q\n  path=%q\n  projecturl=%q\n  issue=%q\n\n",
			target,
			target.Canonical(),
			target.Project(),
			target.Namespace(),
			target.ProviderURL(),
			target.Driver(),
			target.Path(),
			target.ProjectURL(),
			target.Issue(),
		)
	}
	// Output:
	// FIXME
}

func ExampleTargets_UniqueProjects() {
	targets, _ := ParseTargets([]string{
		"moul/project1",
		"moul/project2#42",
		"moul/project2",
		"github.com/moul/project1",
		"https://github.com/moul/project2/issues/42",
		"https://gitlab.com/moul/project1#42",
		"https://gitlab.com/moul/project1",
		"https://gitlab.com/moul/project2",
		"gitlab.com/moul/project1",
	})
	for _, target := range targets.UniqueProjects() {
		fmt.Println(target.Canonical())
	}
	// Output:
	// https://github.com/moul/project1
	// https://github.com/moul/project2
	// https://gitlab.com/moul/project1
	// https://gitlab.com/moul/project2
}
