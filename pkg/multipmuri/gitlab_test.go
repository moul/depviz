package multipmuri

import "fmt"

func ExampleNewGitLabIssue() {
	entity := NewGitLabIssue("", "moul", "depviz", "42")
	fmt.Println("entity")
	fmt.Println(" ", entity.String())
	fmt.Println(" ", entity.Kind())
	fmt.Println(" ", entity.Provider())

	relatives := []string{
		"@moul",
		"#4242",
		"moul2/depviz2#43",
		"gitlab.com/moul2/depviz2#42",
		"https://gitlab.com/moul2/depviz2#42",
		"https://example.com/a/b#42",
		"https://gitlab.com/moul/depviz/issues/42",
	}
	fmt.Println("relationships")
	for _, name := range relatives {
		rel, err := entity.RelDecodeString(name)
		if err != nil {
			fmt.Printf("  %-42s -> error: %v\n", name, err)
			continue
		}
		fmt.Printf("  %-42s -> %s\n", name, rel.String())
	}
	fmt.Println("repo:", entity.RepoEntity().String())
	// Output:
	// entity
	//   https://gitlab.com/moul/depviz/issues/42
	//   issue
	//   gitlab
	// relationships
	//   @moul                                      -> https://gitlab.com/moul
	//   #4242                                      -> https://gitlab.com/moul/depviz/issues/4242
	//   moul2/depviz2#43                           -> https://gitlab.com/moul2/depviz2/issues/43
	//   gitlab.com/moul2/depviz2#42                -> https://gitlab.com/moul2/depviz2/issues/42
	//   https://gitlab.com/moul2/depviz2#42        -> https://gitlab.com/moul2/depviz2/issues/42
	//   https://example.com/a/b#42                 -> error: ambiguous uri "https://example.com/a/b#42"
	//   https://gitlab.com/moul/depviz/issues/42   -> https://gitlab.com/moul/depviz/issues/42
	// repo: https://gitlab.com/moul/depviz
}

func ExampleNewGitLabService() {
	entity := NewGitLabService("gitlab.com")
	fmt.Println("entity")
	fmt.Println(" ", entity.String())
	fmt.Println(" ", entity.Kind())
	fmt.Println(" ", entity.Provider())

	relatives := []string{
		"https://gitlab.com",
		"gitlab.com",
		"gitlab.com/moul",
		"@moul",
		"gitlab.com/moul/depviz",
		"moul/depviz",
		"moul/depviz/-/milestones/1",
		"moul/depviz#1",
		"gitlab.com/moul/depviz/issues/2",
		"gitlab.com/moul/depviz/merge_requests/1",
		"https://gitlab.com/moul/depviz/issues/1",
		"https://gitlab.com/moul/depviz#1",
		"gitlab://moul/depviz#1",
		"gitlab://gitlab.com/moul/depviz#1",
		"gitlab://https://gitlab.com/moul/depviz#1",
		"gitlab.com/a/b/c/d/e/f",
		"gitlab.com/a/b/c/d/e/f#1",
		"gitlab.com/a/b/c/d/e/f!1",
		"a/b/c/d/e/f!1",
		"a/b/c/d/e/f#1",
		"a/b#1",
		"a/b!1",
	}
	fmt.Println("relationships")
	for _, name := range relatives {
		rel, err := entity.RelDecodeString(name)
		if err != nil {
			fmt.Printf("  %-42s -> error: %v\n", name, err)
			continue
		}
		fmt.Printf("  %-42s -> %-48s %s\n", name, rel.String(), rel.Kind())
	}
	// Output:
	// entity
	//   https://gitlab.com/
	//   service
	//   gitlab
	// relationships
	//   https://gitlab.com                         -> https://gitlab.com/                              service
	//   gitlab.com                                 -> https://gitlab.com/                              service
	//   gitlab.com/moul                            -> https://gitlab.com/moul                          user-or-organization
	//   @moul                                      -> https://gitlab.com/moul                          user-or-organization
	//   gitlab.com/moul/depviz                     -> https://gitlab.com/moul/depviz                   organization-or-project
	//   moul/depviz                                -> https://gitlab.com/moul/depviz                   organization-or-project
	//   moul/depviz/-/milestones/1                 -> https://gitlab.com/moul/depviz/-/milestones/1    milestone
	//   moul/depviz#1                              -> https://gitlab.com/moul/depviz/issues/1          issue
	//   gitlab.com/moul/depviz/issues/2            -> https://gitlab.com/moul/depviz/issues/2          issue
	//   gitlab.com/moul/depviz/merge_requests/1    -> https://gitlab.com/moul/depviz/merge_requests/1  merge-request
	//   https://gitlab.com/moul/depviz/issues/1    -> https://gitlab.com/moul/depviz/issues/1          issue
	//   https://gitlab.com/moul/depviz#1           -> https://gitlab.com/moul/depviz/issues/1          issue
	//   gitlab://moul/depviz#1                     -> https://gitlab.com/moul/depviz/issues/1          issue
	//   gitlab://gitlab.com/moul/depviz#1          -> https://gitlab.com/moul/depviz/issues/1          issue
	//   gitlab://https://gitlab.com/moul/depviz#1  -> https://gitlab.com/moul/depviz/issues/1          issue
	//   gitlab.com/a/b/c/d/e/f                     -> https://gitlab.com/a/b/c/d/e/f                   project
	//   gitlab.com/a/b/c/d/e/f#1                   -> https://gitlab.com/a/b/c/d/e/f/issues/1          issue
	//   gitlab.com/a/b/c/d/e/f!1                   -> https://gitlab.com/a/b/c/d/e/f/merge_requests/1  merge-request
	//   a/b/c/d/e/f!1                              -> https://gitlab.com/a/b/c/d/e/f/merge_requests/1  merge-request
	//   a/b/c/d/e/f#1                              -> https://gitlab.com/a/b/c/d/e/f/issues/1          issue
	//   a/b#1                                      -> https://gitlab.com/a/b/issues/1                  issue
	//   a/b!1                                      -> https://gitlab.com/a/b/merge_requests/1          merge-request

}

func ExampleNewGitLabService_Enterprise() {
	entity := NewGitLabService("ge.company.com")
	fmt.Println("entity")
	fmt.Println(" ", entity.String())
	fmt.Println(" ", entity.Kind())
	fmt.Println(" ", entity.Provider())

	relatives := []string{
		"https://gitlab.com",
		"gitlab.com",
		"gitlab.com/moul",
		"@moul",
		"gitlab.com/moul/depviz",
		"moul/depviz",
		"moul/depviz/-/milestones/1",
		"moul/depviz#1",
		"gitlab.com/moul/depviz/issues/2",
		"gitlab.com/moul/depviz/merge_requests/1",
		"https://gitlab.com/moul/depviz/issues/1",
		"https://gitlab.com/moul/depviz#1",
		"gitlab://moul/depviz#1",
		"gitlab://gitlab.com/moul/depviz#1",
		"gitlab://https://gitlab.com/moul/depviz#1",
	}
	fmt.Println("relationships")
	for _, name := range relatives {
		rel, err := entity.RelDecodeString(name)
		if err != nil {
			fmt.Printf("  %-42s -> error: %v\n", name, err)
			continue
		}
		fmt.Printf("  %-42s -> %-43s %s\n", name, rel.String(), rel.Kind())
	}
	// Output:
	// entity
	//   https://ge.company.com/
	//   service
	//   gitlab
	// relationships
	//   https://gitlab.com                         -> https://gitlab.com/                         service
	//   gitlab.com                                 -> https://gitlab.com/                         service
	//   gitlab.com/moul                            -> https://gitlab.com/moul                     user-or-organization
	//   @moul                                      -> https://ge.company.com/moul                 user-or-organization
	//   gitlab.com/moul/depviz                     -> https://gitlab.com/moul/depviz              organization-or-project
	//   moul/depviz                                -> https://ge.company.com/moul/depviz          organization-or-project
	//   moul/depviz/-/milestones/1                 -> https://ge.company.com/moul/depviz/-/milestones/1 milestone
	//   moul/depviz#1                              -> https://ge.company.com/moul/depviz/issues/1 issue
	//   gitlab.com/moul/depviz/issues/2            -> https://gitlab.com/moul/depviz/issues/2     issue
	//   gitlab.com/moul/depviz/merge_requests/1    -> https://gitlab.com/moul/depviz/merge_requests/1 merge-request
	//   https://gitlab.com/moul/depviz/issues/1    -> https://gitlab.com/moul/depviz/issues/1     issue
	//   https://gitlab.com/moul/depviz#1           -> https://gitlab.com/moul/depviz/issues/1     issue
	//   gitlab://moul/depviz#1                     -> https://gitlab.com/moul/depviz/issues/1     issue
	//   gitlab://gitlab.com/moul/depviz#1          -> https://gitlab.com/moul/depviz/issues/1     issue
	//   gitlab://https://gitlab.com/moul/depviz#1  -> https://gitlab.com/moul/depviz/issues/1     issue
}
