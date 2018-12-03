package repo

import "fmt"

func ExampleParseTargets() {
	targets, _ := ParseTargets([]string{
		"moul/project1",
		"moul/project2#42",
		"github.com/moul/project3",
		"https://github.com/moul/project4/issues/42",
		"https://gitlab.com/moul/project5#42",
		"gitlab.com/moul/project6",
	})
	for _, target := range targets {
		fmt.Printf(
			"target=%q canonical=%q project=%q namespace=%q providerurl=%q driver=%q path=%q projecturl=%q issue=%q\n",
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
	// target="https://github.com/moul/project1" canonical="https://github.com/moul/project1" project="project1" namespace="moul" providerurl="https://github.com" driver="github" path="moul/project1" projecturl="https://github.com/moul/project1" issue=""
	// target="https://github.com/moul/project2/issues/42" canonical="https://github.com/moul/project2/issues/42" project="project2" namespace="moul" providerurl="https://github.com" driver="github" path="moul/project2" projecturl="https://github.com/moul/project2" issue="42"
	// target="https://github.com/moul/project3" canonical="https://github.com/moul/project3" project="project3" namespace="moul" providerurl="https://github.com" driver="github" path="moul/project3" projecturl="https://github.com/moul/project3" issue=""
	// target="https://github.com/moul/project4/issues/42" canonical="https://github.com/moul/project4/issues/42" project="project4" namespace="moul" providerurl="https://github.com" driver="github" path="moul/project4" projecturl="https://github.com/moul/project4" issue="42"
	// target="https://gitlab.com/moul/project5/issues/42" canonical="https://gitlab.com/moul/project5/issues/42" project="project5" namespace="moul" providerurl="https://gitlab.com" driver="gitlab" path="moul/project5" projecturl="https://gitlab.com/moul/project5" issue="42"
	// target="https://gitlab.com/moul/project6" canonical="https://gitlab.com/moul/project6" project="project6" namespace="moul" providerurl="https://gitlab.com" driver="gitlab" path="moul/project6" projecturl="https://gitlab.com/moul/project6" issue=""
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
