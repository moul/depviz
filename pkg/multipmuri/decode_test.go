package multipmuri

import "fmt"

func ExampleDecodeString() {
	for _, uri := range []string{
		"https://github.com",
		"github.com",
		"github.com/moul",
		"@moul",
		"github.com/moul/depviz",
		"moul/depviz",
		"moul/depviz/milestone/1",
		"moul/depviz#1",
		"github.com/moul/depviz/issues/2",
		"github.com/moul/depviz/pull/1",
		"https://github.com/moul/depviz/issues/1",
		"https://github.com/moul/depviz#1",
		"github://moul/depviz#1",
		"github://github.com/moul/depviz#1",
		"github://https://github.com/moul/depviz#1",
		"github://ghenterprise.company.com/a/b#42",
		"github://https://ghenterprise.company.com",
		"git@github.com:moul/depviz",
	} {
		decoded, err := DecodeString(uri)
		if err != nil {
			fmt.Printf("%-42s error: %v\n", uri, err)
			continue
		}
		fmt.Printf("%-42s %-48s %-8s %s\n", uri, decoded.String(), decoded.Provider(), decoded.Kind())
	}
	// Output:
	// https://github.com                         https://github.com/                              github   service
	// github.com                                 https://github.com/                              github   service
	// github.com/moul                            https://github.com/moul                          github   user-or-organization
	// @moul                                      error: ambiguous uri "@moul"
	// github.com/moul/depviz                     https://github.com/moul/depviz                   github   project
	// moul/depviz                                error: ambiguous uri "moul/depviz"
	// moul/depviz/milestone/1                    error: ambiguous uri "moul/depviz/milestone/1"
	// moul/depviz#1                              error: ambiguous uri "moul/depviz#1"
	// github.com/moul/depviz/issues/2            https://github.com/moul/depviz/issues/2          github   issue
	// github.com/moul/depviz/pull/1              https://github.com/moul/depviz/issues/1          github   merge-request
	// https://github.com/moul/depviz/issues/1    https://github.com/moul/depviz/issues/1          github   issue
	// https://github.com/moul/depviz#1           https://github.com/moul/depviz/issues/1          github   issue-or-merge-request
	// github://moul/depviz#1                     https://github.com/moul/depviz/issues/1          github   issue-or-merge-request
	// github://github.com/moul/depviz#1          https://github.com/moul/depviz/issues/1          github   issue-or-merge-request
	// github://https://github.com/moul/depviz#1  https://github.com/moul/depviz/issues/1          github   issue-or-merge-request
	// github://ghenterprise.company.com/a/b#42   https://ghenterprise.company.com/a/b/issues/42   github   issue-or-merge-request
	// github://https://ghenterprise.company.com  https://ghenterprise.company.com/                github   service
	// git@github.com:moul/depviz                 https://github.com/moul/depviz                   github   project
}
