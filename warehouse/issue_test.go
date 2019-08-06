package warehouse

import "fmt"

func ExampleIssue_GetRelativeURL() {
	issue := Issue{
		URL: "https://github.com/moul/depviz/issues/42",
		Repository: &Repository{
			URL: "https://github.com/moul/depviz",
			Provider: &Provider{
				URL: "https://github.com/",
			},
		},
	}
	for _, target := range []string{
		"#43",
		"moul/depviz#44",
		"github.com/moul/depviz/issues/45",
		"https://github.com/moul/depviz/issues/46",
		"test/test#47",
		"github.com/test/test/issues/48",
		"https://github.com/test/test/issues/49",
		"gitlab.com/moul/depviz/issues/50",
		"https://gitlab.com/moul/depviz/issues/51",
		"gitlab.com/test2/issues/52",
		"gitlab.com/test2/test3/test4/issues/53",
	} {
		fmt.Printf("%-42s -> %s\n", target, issue.GetRelativeURL(target))
	}

	issue = Issue{
		URL: "https://gitlab.com/moul/depviz/issues/42",
		Repository: &Repository{
			URL: "https://gitlab.com/moul/depviz",
			Provider: &Provider{
				URL: "https://gitlab.com/",
			},
		},
	}
	for _, target := range []string{
		"#43",
		"moul/depviz#44",
		"gitlab.com/moul/depviz/issues/45",
		"https://gitlab.com/moul/depviz/issues/46",
		"test/test#47",
		"gitlab.com/test/test/issues/48",
		"https://gitlab.com/test/test/issues/49",
		"github.com/moul/depviz/issues/50",
		"https://github.com/moul/depviz/issues/51",
		"github.com/test2/issues/52",
		"github.com/test2/test3/test4/issues/53",
	} {
		fmt.Printf("%-42s -> %s\n", target, issue.GetRelativeURL(target))
	}

	// Output:
	// #43                                        -> https://github.com/moul/depviz/issues/43
	// moul/depviz#44                             -> https://github.com/moul/depviz/issues/44
	// github.com/moul/depviz/issues/45           -> https://github.com/moul/depviz/issues/45
	// https://github.com/moul/depviz/issues/46   -> https://github.com/moul/depviz/issues/46
	// test/test#47                               -> https://github.com/test/test/issues/47
	// github.com/test/test/issues/48             -> https://github.com/test/test/issues/48
	// https://github.com/test/test/issues/49     -> https://github.com/test/test/issues/49
	// gitlab.com/moul/depviz/issues/50           -> https://gitlab.com/moul/depviz/issues/50
	// https://gitlab.com/moul/depviz/issues/51   -> https://gitlab.com/moul/depviz/issues/51
	// gitlab.com/test2/issues/52                 -> https://gitlab.com/test2/issues/52
	// gitlab.com/test2/test3/test4/issues/53     -> https://gitlab.com/test2/test3/test4/issues/53
	// #43                                        -> https://gitlab.com/moul/depviz/issues/43
	// moul/depviz#44                             -> https://gitlab.com/moul/depviz/issues/44
	// gitlab.com/moul/depviz/issues/45           -> https://gitlab.com/moul/depviz/issues/45
	// https://gitlab.com/moul/depviz/issues/46   -> https://gitlab.com/moul/depviz/issues/46
	// test/test#47                               -> https://gitlab.com/test/test/issues/47
	// gitlab.com/test/test/issues/48             -> https://gitlab.com/test/test/issues/48
	// https://gitlab.com/test/test/issues/49     -> https://gitlab.com/test/test/issues/49
	// github.com/moul/depviz/issues/50           -> https://github.com/moul/depviz/issues/50
	// https://github.com/moul/depviz/issues/51   -> https://github.com/moul/depviz/issues/51
	// github.com/test2/issues/52                 -> https://github.com/test2/issues/52
	// github.com/test2/test3/test4/issues/53     -> https://github.com/test2/test3/test4/issues/53
}
