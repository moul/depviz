package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"sort"
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
			if _, err := s.AddEdgeWithConfidence(ctx, DefaultBoardID, edge.From, edge.To, edge.Kind, "github-inferred", edge.Confidence, edge); err != nil {
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
			if _, err := s.AddEdgeWithConfidence(ctx, DefaultBoardID, edge.From, edge.To, edge.Kind, "github-inferred", edge.Confidence, edge); err != nil {
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
	From       string  `json:"from"`
	To         string  `json:"to"`
	Kind       string  `json:"kind"`
	Line       string  `json:"line"`
	Confidence float64 `json:"confidence"`
}

var (
	githubRefRE          = regexp.MustCompile(`(?i)(?:gh:)?([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+)?([#!])([0-9]+)`)
	githubURLRefRE       = regexp.MustCompile(`(?i)https://(?:www\.)?(?:github|redirect\.github)\.com/([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+)/(issues|pull)/([0-9]+)`)
	htmlAnchorRefStartRE = regexp.MustCompile(`(?i)^\s*<a\s+href=["']https://(?:www\.)?(?:github|redirect\.github)\.com/[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+/(?:issues|pull)/[0-9]+["']>`)
	relationRefStartRE   = regexp.MustCompile(`(?i)^\s*[:\-]?\s*(?:https://(?:www\.)?(?:github|redirect\.github)\.com/[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+/(?:issues|pull)/[0-9]+|(?:gh:)?(?:[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+)?[#!][0-9]+)`)
	relationVerbRE       = regexp.MustCompile(`(?i)\b(blocked by|depends on|depend on|depends|after|blocks|unblocks|addresses|mentions|relates to|relates|closes|closed|close|fixes|fixed|fix|resolves|resolved|resolve)\b`)
)

func extractDependencyEdges(repo, currentID, body string) []extractedEdge {
	var edges []extractedEdge
	seen := map[string]bool{}
	for _, line := range strings.Split(body, "\n") {
		lower := strings.ToLower(line)
		if !strings.Contains(lower, "block") && !strings.Contains(lower, "depend") && !strings.Contains(lower, "after") && !strings.Contains(lower, "address") && !strings.Contains(lower, "mention") && !strings.Contains(lower, "relate") && !strings.Contains(lower, "close") && !strings.Contains(lower, "fix") && !strings.Contains(lower, "resolve") {
			continue
		}
		for _, chunk := range relationChunks(line) {
			kind := relationEdgeKind(chunk.verb)
			if kind == "" {
				continue
			}
			if !relationChunkTargetsCurrent(repo, chunk) {
				continue
			}
			confidence := relationConfidence(kind)
			for _, target := range githubRefs(repo, chunk.text) {
				if target == currentID {
					continue
				}
				key := currentID + "\x00" + target + "\x00" + kind
				if seen[key] {
					continue
				}
				seen[key] = true
				edges = append(edges, extractedEdge{From: currentID, To: target, Kind: kind, Line: strings.TrimSpace(line), Confidence: confidence})
			}
		}
	}
	return edges
}

func relationChunkTargetsCurrent(repo string, chunk relationChunk) bool {
	if !relationRefStartRE.MatchString(chunk.text) && !htmlAnchorRefStartRE.MatchString(chunk.text) {
		return false
	}
	return len(githubRefs(repo, chunk.prev)) == 0
}

type relationChunk struct {
	verb string
	text string
	prev string
}

func relationChunks(line string) []relationChunk {
	matches := relationVerbRE.FindAllStringSubmatchIndex(line, -1)
	if len(matches) == 0 {
		return nil
	}
	chunks := make([]relationChunk, 0, len(matches))
	for i, match := range matches {
		start := match[1]
		end := len(line)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		chunks = append(chunks, relationChunk{
			verb: line[match[2]:match[3]],
			text: line[start:end],
			prev: relationPrefix(line, match[0], i),
		})
	}
	return chunks
}

func relationPrefix(line string, verbStart, index int) string {
	if index > 0 {
		return ""
	}
	return line[:verbStart]
}

func relationEdgeKind(verb string) string {
	switch strings.ToLower(strings.TrimSpace(verb)) {
	case "blocked by", "depends on", "depend on", "depends", "after":
		return "blocked_by"
	case "blocks", "unblocks":
		return "blocks"
	case "addresses":
		return "addresses"
	case "mentions":
		return "mentions"
	case "relates", "relates to":
		return "relates_to"
	case "close", "closes", "closed", "fix", "fixes", "fixed", "resolve", "resolves", "resolved":
		return "closes"
	default:
		return ""
	}
}

func relationConfidence(kind string) float64 {
	switch kind {
	case "blocked_by", "blocks":
		return 0.75
	case "closes":
		return 0.7
	case "addresses", "mentions", "relates_to":
		return 0.55
	default:
		return 0.5
	}
}

func githubRefs(defaultRepo, text string) []string {
	type refMatch struct {
		start int
		end   int
		id    string
	}
	var matches []refMatch
	for _, match := range githubURLRefRE.FindAllStringSubmatchIndex(text, -1) {
		marker := "#"
		if strings.EqualFold(text[match[4]:match[5]], "pull") {
			marker = "!"
		}
		matches = append(matches, refMatch{
			start: match[0],
			end:   match[1],
			id:    "gh:" + text[match[2]:match[3]] + marker + text[match[6]:match[7]],
		})
	}
	for _, match := range githubRefRE.FindAllStringSubmatchIndex(text, -1) {
		if !validRefBoundary(text, match[0], match[1]) {
			continue
		}
		refRepo := defaultRepo
		if match[2] >= 0 {
			refRepo = strings.TrimPrefix(text[match[2]:match[3]], "gh:")
		}
		matches = append(matches, refMatch{
			start: match[0],
			end:   match[1],
			id:    "gh:" + refRepo + text[match[4]:match[5]] + text[match[6]:match[7]],
		})
	}
	if len(matches) == 0 {
		return nil
	}
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].start == matches[j].start {
			return matches[i].end > matches[j].end
		}
		return matches[i].start < matches[j].start
	})
	var refs []string
	seen := map[string]bool{}
	coveredUntil := -1
	for _, match := range matches {
		if match.start < coveredUntil {
			continue
		}
		coveredUntil = match.end
		if seen[match.id] {
			continue
		}
		seen[match.id] = true
		refs = append(refs, match.id)
	}
	return refs
}

func validRefBoundary(text string, start, end int) bool {
	if start > 0 {
		prev := text[start-1]
		if prev == '&' {
			return false
		}
		if prev == '>' && end < len(text) && text[end] == '<' {
			return false
		}
		if isRefWordByte(prev) {
			return false
		}
	}
	if end < len(text) && isRefWordByte(text[end]) {
		return false
	}
	return true
}

func isRefWordByte(b byte) bool {
	return b == '_' || b == '-' || b == '.' || (b >= '0' && b <= '9') || (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
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
