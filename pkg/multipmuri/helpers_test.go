package multipmuri

import "fmt"

func ExampleRepoEntity() {
	entities := []Entity{
		NewGitHubIssue("", "moul", "depviz", "42"),
		NewGitHubMilestone("", "moul", "depviz", "42"),
		NewGitHubRepo("", "moul", "depviz"),
		NewGitHubOwner("", "moul"),
		NewGitHubService(""),
	}
	for _, entity := range entities {
		fmt.Printf("%-50s -> %v\n", entity, RepoEntity(entity))
	}
	// Output:
	// https://github.com/moul/depviz/issues/42           -> https://github.com/moul/depviz
	// https://github.com/moul/depviz/milestone/42        -> https://github.com/moul/depviz
	// https://github.com/moul/depviz                     -> https://github.com/moul/depviz
	// https://github.com/moul                            -> <nil>
	// https://github.com/                                -> <nil>
}

func ExampleOwnerEntity() {
	entities := []Entity{
		NewGitHubIssue("", "moul", "depviz", "42"),
		NewGitHubMilestone("", "moul", "depviz", "42"),
		NewGitHubRepo("", "moul", "depviz"),
		NewGitHubOwner("", "moul"),
		NewGitHubService(""),
	}
	for _, entity := range entities {
		fmt.Printf("%-50s -> %v\n", entity, OwnerEntity(entity))
	}
	// Output:
	// https://github.com/moul/depviz/issues/42           -> https://github.com/moul
	// https://github.com/moul/depviz/milestone/42        -> https://github.com/moul
	// https://github.com/moul/depviz                     -> https://github.com/moul
	// https://github.com/moul                            -> https://github.com/moul
	// https://github.com/                                -> <nil>
}

func ExampleServiceEntity() {
	entities := []Entity{
		NewGitHubIssue("", "moul", "depviz", "42"),
		NewGitHubMilestone("", "moul", "depviz", "42"),
		NewGitHubRepo("", "moul", "depviz"),
		NewGitHubOwner("", "moul"),
		NewGitHubService(""),
	}
	for _, entity := range entities {
		fmt.Printf("%-50s -> %v\n", entity, ServiceEntity(entity))
	}
	// Output:
	// https://github.com/moul/depviz/issues/42           -> https://github.com/
	// https://github.com/moul/depviz/milestone/42        -> https://github.com/
	// https://github.com/moul/depviz                     -> https://github.com/
	// https://github.com/moul                            -> https://github.com/
	// https://github.com/                                -> https://github.com/
}
