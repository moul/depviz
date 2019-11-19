package main

import (
	"context"
	"flag"
	"log"
	"os"
	"regexp"

	"github.com/google/go-github/v28/github"
	"golang.org/x/oauth2"
)

var (
	owner   = flag.String("owner", "", "github user")
	repo    = flag.String("repo", "", "github repo")
	pattern = flag.String("pattern", "", "regex pattern")
	replace = flag.String("replace", "", "regex replace")
)

func main() {
	flag.Parse()

	re := regexp.MustCompile(*pattern)

	// github client
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	var allIssues []*github.Issue
	opts := &github.IssueListByRepoOptions{State: "all"}
	for {
		issues, resp, err := client.Issues.ListByRepo(ctx, *owner, *repo, opts)
		if err != nil {
			log.Fatal(err)
		}
		allIssues = append(allIssues, issues...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	for _, issue := range allIssues {
		after := re.ReplaceAllString(*issue.Body, *replace)
		if *issue.Body != after {
			if _, _, err := client.Issues.Edit(ctx, *owner, *repo, *issue.Number, &github.IssueRequest{Body: &after}); err != nil {
				log.Fatal(err)
			}
			log.Println(*issue.Number)
		}
	}
}
