package githubprovider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/go-github/v30/github"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"moul.io/depviz/v3/internal/dvmodel"
	"moul.io/multipmuri"
)

type Opts struct {
	Since  *time.Time  `json:"since"`
	Logger *zap.Logger `json:"-"`
}

func FetchRepo(ctx context.Context, entity multipmuri.Entity, token string, out chan<- dvmodel.Batch, opts Opts) { // nolint:interfacer
	if opts.Logger == nil {
		opts.Logger = zap.NewNop()
	}

	type multipmuriMinimalInterface interface {
		Repo() *multipmuri.GitHubRepo
	}
	target, ok := entity.(multipmuriMinimalInterface)
	if !ok {
		opts.Logger.Warn("invalid entity", zap.String("entity", fmt.Sprintf("%v", entity.String())))
		return
	}
	repo := target.Repo()

	// create client
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	// queries
	totalIssues := 0
	callOpts := &github.IssueListByRepoOptions{State: "all"}
	if opts.Since != nil {
		callOpts.Since = *opts.Since
	}
	for {
		issues, resp, err := client.Issues.ListByRepo(ctx, repo.OwnerID(), repo.RepoID(), callOpts)
		if err != nil {
			opts.Logger.Error("fetch GitHub issues", zap.Error(err))
			return
		}
		totalIssues += len(issues)
		opts.Logger.Debug("paginate",
			zap.Any("opts", opts),
			zap.String("provider", "github"),
			zap.String("repo", repo.String()),
			zap.Int("new-issues", len(issues)),
			zap.Int("total-issues", totalIssues),
		)

		if len(issues) > 0 {
			batch := fromIssues(issues, opts.Logger)
			out <- batch
		}

		// handle pagination
		if resp.NextPage == 0 {
			break
		}
		callOpts.Page = resp.NextPage
	}

	if rateLimits, _, err := client.RateLimits(ctx); err == nil {
		opts.Logger.Debug("github API rate limiting", zap.Stringer("limit", rateLimits.GetCore()))
	}

	// FIXME: fetch incomplete/old users, orgs, teams & repos
}

func AddAssignee(ctx context.Context, assignee string, id int, owner string, repo string, gitHubToken string, Logger *zap.Logger) bool {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: gitHubToken})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	_, resp, err := client.Issues.AddAssignees(ctx, owner, repo, id, []string{assignee})
	if err != nil {
		Logger.Error("add assignee", zap.Error(err))
		return false
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		Logger.Info("add assignee", zap.Int("status code", resp.StatusCode))
		return true
	}
	Logger.Warn("add assignee", zap.String("assignee", assignee), zap.Int("id", id), zap.String("owner", owner), zap.String("repo", repo), zap.Int("status", resp.StatusCode))
	return false
}

func IssueAddMetadata(ctx context.Context, id int, owner string, repo string, gitHubToken string, metadata string, Logger *zap.Logger) bool {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: gitHubToken})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	issue, resp, err := client.Issues.Get(ctx, owner, repo, id)
	if err != nil {
		Logger.Error("get issue", zap.Error(err))
		return false
	}

	// add metadata at the end of the body in the "-- depviz auto --" section
	newBody := *issue.Body
	var hasSection bool
	for _, s := range strings.Split(*issue.Body, "\n") {
		// check if the section exist(mark to change)
		if s == "-- depviz auto --\r" {
			hasSection = true
		}
		// return true if duplicate
		if s == metadata {
			return true
		}
	}
	if !hasSection {
		newBody += "\n\n-- depviz auto --"
	}
	newBody += "\n" + metadata

	_, resp, err = client.Issues.Edit(ctx, owner, repo, id, &github.IssueRequest{Body: &newBody})
	if err != nil {
		Logger.Error("add metadata", zap.Error(err))
		return false
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		Logger.Info("add metadata", zap.Int("status code", resp.StatusCode))
		return true
	}
	Logger.Warn("add metadata", zap.String("metadata", metadata), zap.Int("id", id), zap.String("owner", owner), zap.String("repo", repo), zap.Int("status", resp.StatusCode))
	return false
}
