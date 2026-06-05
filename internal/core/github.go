package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

type GitHubSyncOptions struct {
	Repo  string
	Limit int
}

func SyncGitHub(ctx context.Context, s *Store, opts GitHubSyncOptions) (int, error) {
	if opts.Repo == "" {
		return 0, fmt.Errorf("repo is required")
	}
	if opts.Limit <= 0 {
		opts.Limit = 200
	}
	sourceID := "github:" + opts.Repo
	if err := s.UpsertSource(ctx, Source{
		ID:           sourceID,
		Kind:         "github",
		Name:         opts.Repo,
		URL:          "https://github.com/" + opts.Repo,
		Capabilities: `{"read":true,"write":"via-gh-later"}`,
		Sync:         `{"tool":"gh"}`,
		UpdatedAt:    nowUTC(),
	}); err != nil {
		return 0, err
	}
	issues, err := ghList[ghIssue](ctx, "issue", opts.Repo, opts.Limit)
	if err != nil {
		return 0, err
	}
	prs, err := ghList[ghPR](ctx, "pr", opts.Repo, opts.Limit)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, issue := range issues {
		id := fmt.Sprintf("gh:%s#%d", opts.Repo, issue.Number)
		body := issue.Body
		payload := githubPayload("issue", opts.Repo, issue.Number, issue.LabelNames(), issue.AssigneeNames(), body)
		n := Node{
			ID:        id,
			Kind:      "issue",
			Title:     issue.Title,
			State:     strings.ToLower(issue.State),
			Owner:     first(issue.AssigneeNames()),
			DataJSON:  payload,
			UpdatedAt: parseGitHubTime(issue.UpdatedAt),
		}
		if err := s.UpsertNode(ctx, n); err != nil {
			return count, err
		}
		if err := s.UpsertSourceRef(ctx, id, sourceID, fmt.Sprintf("#%d", issue.Number), issue.URL); err != nil {
			return count, err
		}
		if err := s.AddNodeToBoard(ctx, DefaultBoardID, id, "issue", ""); err != nil {
			return count, err
		}
		count++
		for _, edge := range extractDependencyEdges(opts.Repo, id, body) {
			if _, err := s.AddEdge(ctx, DefaultBoardID, edge.From, edge.To, edge.Kind, "github-text", edge); err != nil {
				return count, err
			}
		}
	}
	for _, pr := range prs {
		id := fmt.Sprintf("gh:%s!%d", opts.Repo, pr.Number)
		payload := githubPayload("pr", opts.Repo, pr.Number, pr.LabelNames(), pr.AssigneeNames(), pr.Body)
		state := strings.ToLower(pr.State)
		if pr.MergedAt != "" {
			state = "merged"
		}
		n := Node{
			ID:        id,
			Kind:      "pr",
			Title:     pr.Title,
			State:     state,
			Owner:     first(pr.AssigneeNames()),
			DataJSON:  payload,
			UpdatedAt: parseGitHubTime(pr.UpdatedAt),
			URL:       pr.URL,
		}
		if err := s.UpsertNode(ctx, n); err != nil {
			return count, err
		}
		if err := s.UpsertSourceRef(ctx, id, sourceID, fmt.Sprintf("!%d", pr.Number), pr.URL); err != nil {
			return count, err
		}
		if err := s.AddNodeToBoard(ctx, DefaultBoardID, id, "pr", ""); err != nil {
			return count, err
		}
		count++
		for _, edge := range extractDependencyEdges(opts.Repo, id, pr.Body) {
			if _, err := s.AddEdge(ctx, DefaultBoardID, edge.From, edge.To, edge.Kind, "github-text", edge); err != nil {
				return count, err
			}
		}
	}
	return count, nil
}

func ghList[T any](ctx context.Context, kind, repo string, limit int) ([]T, error) {
	fields := "number,title,state,url,labels,assignees,updatedAt,createdAt,body"
	if kind == "pr" {
		fields = "number,title,state,url,labels,assignees,updatedAt,createdAt,body,mergedAt"
	}
	cmd := exec.CommandContext(ctx, "gh", kind, "list", "--repo", repo, "--state", "all", "--limit", fmt.Sprint(limit), "--json", fields)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("gh %s list failed: %w: %s", kind, err, strings.TrimSpace(stderr.String()))
	}
	var items []T
	if err := json.Unmarshal(out, &items); err != nil {
		return nil, err
	}
	return items, nil
}

type ghIssue struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	State     string    `json:"state"`
	URL       string    `json:"url"`
	Body      string    `json:"body"`
	Labels    []ghLabel `json:"labels"`
	Assignees []ghUser  `json:"assignees"`
	UpdatedAt string    `json:"updatedAt"`
}

type ghPR struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	State     string    `json:"state"`
	URL       string    `json:"url"`
	Body      string    `json:"body"`
	MergedAt  string    `json:"mergedAt"`
	Labels    []ghLabel `json:"labels"`
	Assignees []ghUser  `json:"assignees"`
	UpdatedAt string    `json:"updatedAt"`
}

type ghLabel struct {
	Name string `json:"name"`
}

type ghUser struct {
	Login string `json:"login"`
	Name  string `json:"name"`
}

func (g ghIssue) LabelNames() []string {
	return labelNames(g.Labels)
}

func (g ghIssue) AssigneeNames() []string {
	return assigneeNames(g.Assignees)
}

func (g ghPR) LabelNames() []string {
	return labelNames(g.Labels)
}

func (g ghPR) AssigneeNames() []string {
	return assigneeNames(g.Assignees)
}

func labelNames(labels []ghLabel) []string {
	out := make([]string, 0, len(labels))
	for _, l := range labels {
		if l.Name != "" {
			out = append(out, l.Name)
		}
	}
	return out
}

func assigneeNames(users []ghUser) []string {
	out := make([]string, 0, len(users))
	for _, u := range users {
		if u.Login != "" {
			out = append(out, u.Login)
		} else if u.Name != "" {
			out = append(out, u.Name)
		}
	}
	return out
}

func githubPayload(kind, repo string, number int, labels, assignees []string, body string) string {
	payload, _ := json.Marshal(map[string]any{
		"source":    "github",
		"kind":      kind,
		"repo":      repo,
		"number":    number,
		"labels":    labels,
		"assignees": assignees,
		"body":      body,
	})
	return string(payload)
}

type extractedEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Kind string `json:"kind"`
	Line string `json:"line"`
}

var refRE = regexp.MustCompile(`(?i)([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+)?([#!])([0-9]+)`)

func extractDependencyEdges(repo, currentID, body string) []extractedEdge {
	var edges []extractedEdge
	for _, line := range strings.Split(body, "\n") {
		lower := strings.ToLower(line)
		if !strings.Contains(lower, "block") && !strings.Contains(lower, "depend") && !strings.Contains(lower, "after") {
			continue
		}
		matches := refRE.FindAllStringSubmatchIndex(line, -1)
		for _, match := range matches {
			if match[0] > 0 && line[match[0]-1] == '&' {
				continue
			}
			refRepo := repo
			if match[2] >= 0 {
				refRepo = line[match[2]:match[3]]
			}
			target := "gh:" + refRepo + line[match[4]:match[5]] + line[match[6]:match[7]]
			if target == currentID {
				continue
			}
			switch {
			case strings.Contains(lower, "blocked by"), strings.Contains(lower, "depends on"), strings.Contains(lower, "depend on"), strings.Contains(lower, "after"):
				edges = append(edges, extractedEdge{From: currentID, To: target, Kind: "blocked_by", Line: strings.TrimSpace(line)})
			case strings.Contains(lower, "blocks"), strings.Contains(lower, "unblocks"):
				edges = append(edges, extractedEdge{From: currentID, To: target, Kind: "blocks", Line: strings.TrimSpace(line)})
			}
		}
	}
	return edges
}

func parseGitHubTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nowUTC()
	}
	return t.UTC()
}

func first(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
