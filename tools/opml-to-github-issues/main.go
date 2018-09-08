package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/gilliek/go-opml/opml"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

var (
	opmlpath = flag.String("opml-file", "", "path to opml file")
	owner    = flag.String("owner", "", "github user")
	repo     = flag.String("repo", "", "github repo")
)

func main() {
	flag.Parse()
	doc, err := opml.NewOPMLFromFile(*opmlpath)
	if err != nil {
		panic(err)
	}

	// github client
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	for _, outline := range doc.Outlines() {
		if err := handleOutline(ctx, client, &outline, 0, ""); err != nil {
			panic(err)
		}
	}
	log.Println("done")
}

func handleOutline(ctx context.Context, client *github.Client, outline *opml.Outline, parentID int, prefix string) error {
	body := ""
	if parentID > 0 {
		body = fmt.Sprintf("blocks #%d", parentID)
	}
	req := github.IssueRequest{
		Title: &outline.Text,
		Body:  &body,
	}
	issue, resp, err := client.Issues.Create(ctx, *owner, *repo, &req)
	if err != nil {
		return err
	}
	log.Printf("%s issue:%d parent:%d quota:%d text:%q\n", prefix, *issue.Number, parentID, resp.Rate.Remaining, outline.Text)

	for _, child := range outline.Outlines {
		if err := handleOutline(ctx, client, &child, *issue.Number, prefix+"  "); err != nil {
			return err
		}
	}
	return nil
}
