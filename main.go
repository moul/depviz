package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/awalterschulze/gographviz"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

var (
	onlyOrphans  = os.Getenv("ONLY_ORPHANS") == "1"
	showOrphans  = onlyOrphans || os.Getenv("SHOW_ORPHANS") == "1"
	showClosed   = os.Getenv("SHOW_CLOSED") == "1"
	onlyFetch    = os.Getenv("ONLY_FETCH") == "1"
	roadmapFetch = onlyFetch || os.Getenv("ROADMAP_FETCH") == "1"
	organization = os.Getenv("DEPVIZ_ORGANIZATION")
	repos        = strings.Split(os.Getenv("DEPVIS_REPOS"), ",")
)

type Issue struct {
	github.Issue
	DependsOn        []*Issue
	Blocks           []*Issue
	weightMultiplier int
	BaseWeight       int
	IsOrphan         bool
	Hidden           bool
	IsDuplicate      int
	LinkedWithEpic   bool
	Errors           []error
}

type Issues map[string]*Issue

func (i Issue) IsEpic() bool {
	for _, label := range i.Labels {
		if *label.Name == "t/epic" {
			return true
		}
	}
	return false
	//return !i.IsOrphan && len(i.Blocks) == 0
}

func (i Issue) Repo() string {
	return strings.Split(*i.RepositoryURL, "/")[5]
}

func (i Issue) FullRepo() string {
	return fmt.Sprintf("%s/%s", i.Owner(), i.Repo())
}

func (i Issue) RepoID() string {
	return strings.Replace(i.FullRepo(), "/", "", -1)
}

func (i Issue) Owner() string {
	return strings.Split(*i.RepositoryURL, "/")[4]
}

func (i Issue) IsClosed() bool {
	return *i.State == "closed"
}

func (i Issue) IsReady() bool {
	return !i.IsOrphan && len(i.DependsOn) == 0
}

func (i Issue) NodeName() string {
	return fmt.Sprintf(`"%s#%d"`, i.FullRepo(), *i.Number)
}

func (i Issue) NodeTitle() string {
	title := fmt.Sprintf("%s: %s", i.NodeName(), *i.Title)
	title = strings.Replace(html.EscapeString(wrap(title, 20)), "\n", "<br/>", -1)
	labels := []string{}
	for _, label := range i.Labels {
		switch *label.Name {
		case "t/step", "t/epic":
			continue
		}
		labels = append(labels, fmt.Sprintf(`<td bgcolor="#%s">%s</td>`, *label.Color, *label.Name))
	}
	labelsText := ""
	if len(labels) > 0 {
		labelsText = "<tr><td><table><tr>" + strings.Join(labels, "") + "</tr></table></td></tr>"
	}
	assigneeText := ""
	if len(i.Assignees) > 0 {
		assignees := []string{}
		for _, assignee := range i.Assignees {
			assignees = append(assignees, *assignee.Login)
		}
		assigneeText = fmt.Sprintf(`<tr><td><font color="purple"><i>@%s</i></font></td></tr>`, strings.Join(assignees, ", @"))
	}
	errorsText := ""
	if len(i.Errors) > 0 {
		errors := []string{}
		for _, err := range i.Errors {
			errors = append(errors, err.Error())
		}
		errorsText = fmt.Sprintf(`<tr><td bgcolor="red">ERR: %s</td></tr>`, strings.Join(errors, "; "))
	}
	return fmt.Sprintf(`<<table><tr><td>%s</td></tr>%s%s%s</table>>`, title, labelsText, assigneeText, errorsText)
}

func (i Issue) BlocksAnEpic() bool {
	for _, dep := range i.Blocks {
		if dep.IsEpic() || dep.BlocksAnEpic() {
			return true
		}
	}
	return false
}

func (i Issue) DependsOnAnEpic() bool {
	for _, dep := range i.DependsOn {
		if dep.IsEpic() || dep.DependsOnAnEpic() {
			return true
		}
	}
	return false
}

func (i Issue) Weight() int {
	weight := i.BaseWeight
	for _, dep := range i.Blocks {
		weight += dep.Weight()
	}
	return weight * i.WeightMultiplier()
}

func (i Issue) WeightMultiplier() int {
	multiplier := i.weightMultiplier
	for _, dep := range i.Blocks {
		multiplier *= dep.WeightMultiplier()
	}
	return multiplier
}

func (i Issue) AddEdgesToGraph(g *gographviz.Graph) error {
	if i.Hidden {
		return nil
	}
	for _, dependency := range i.DependsOn {
		if dependency.Hidden {
			continue
		}
		attrs := map[string]string{}
		attrs["color"] = "lightblue"
		//attrs["label"] = "depends on"
		//attrs["style"] = "dotted"
		attrs["dir"] = "none"
		if i.IsClosed() || dependency.IsClosed() {
			attrs["color"] = "grey"
			attrs["style"] = "dotted"
		}
		if dependency.IsReady() {
			attrs["color"] = "pink"
		}
		if i.IsEpic() {
			attrs["color"] = "orange"
			attrs["style"] = "dashed"
		}
		log.Print("edge", i.NodeName(), "->", dependency.NodeName())
		if err := g.AddEdge(
			i.NodeName(),
			dependency.NodeName(),
			true,
			attrs,
		); err != nil {
			return err
		}
	}
	return nil
}

func (i Issue) AddNodeToGraph(g *gographviz.Graph, parent string) error {
	attrs := map[string]string{}
	attrs["label"] = i.NodeTitle()
	//attrs["xlabel"] = ""
	attrs["shape"] = "record"
	attrs["style"] = `"rounded,filled"`
	attrs["color"] = "lightblue"

	if i.IsEpic() {
		attrs["shape"] = "oval"
	}

	switch {

	case i.IsClosed():
		attrs["color"] = `"#cccccc33"`

	case i.IsReady():
		attrs["color"] = "pink"

	case i.IsEpic():
		attrs["color"] = "orange"
		attrs["style"] = `"rounded,filled,bold"`

	case i.IsOrphan || !i.LinkedWithEpic:
		attrs["color"] = "gray"
	}

	log.Print("node", i.NodeName(), parent)
	return g.AddNode(
		parent,
		i.NodeName(),
		attrs,
	)
}

func escape(input string) string {
	return fmt.Sprintf("%q", input)
}

func enrich(issues Issues) error {
	var (
		dependsOnRegex, _        = regexp.Compile(`(?i)(require|requires|blocked by|block by|depend on|depends on|parent of) ([a-z/]*#[0-9]+)`)
		blocksRegex, _           = regexp.Compile(`(?i)(blocks|block|address|addresses|part of|child of) ([a-z/]*#[0-9]+)`)
		weightMultiplierRegex, _ = regexp.Compile(`(?i)(weight_multiplier=)([0-9]+)`)
		baseWeightRegex, _       = regexp.Compile(`(?i)(base_weight=)([0-9]+)`)
		hideFromRoadmapRegex, _  = regexp.Compile(`(?i)(!hide_from_roadmap)`) // FIXME: use label
		isDuplicateRegex, _      = regexp.Compile(`(?i)(duplicates|duplicate) #([0-9]+)`)
	)

	for _, issue := range issues {
		issue.DependsOn = make([]*Issue, 0)
		issue.Blocks = make([]*Issue, 0)
		issue.IsOrphan = true
		issue.weightMultiplier = 1
		issue.BaseWeight = 1
	}
	for _, issue := range issues {
		if issue.Body == nil {
			continue
		}

		if match := isDuplicateRegex.FindStringSubmatch(*issue.Body); match != nil {
			issue.IsDuplicate, _ = strconv.Atoi(match[len(match)-1])
		}

		if match := weightMultiplierRegex.FindStringSubmatch(*issue.Body); match != nil {
			issue.weightMultiplier, _ = strconv.Atoi(match[len(match)-1])
		}

		if match := hideFromRoadmapRegex.FindStringSubmatch(*issue.Body); match != nil {
			delete(issues, issue.NodeName())
			continue
		}

		if match := baseWeightRegex.FindStringSubmatch(*issue.Body); match != nil {
			issue.BaseWeight, _ = strconv.Atoi(match[len(match)-1])
		}

		for _, match := range dependsOnRegex.FindAllStringSubmatch(*issue.Body, -1) {
			num := match[len(match)-1]
			if num[0] == '#' {
				num = fmt.Sprintf(`"%s%s"`, issue.FullRepo(), num)
			}
			dep, found := issues[num]
			if !found {
				issue.Errors = append(issue.Errors, fmt.Errorf("dep %q not found", num))
				continue
			}
			issue.DependsOn = append(issue.DependsOn, dep)
			issues[num].Blocks = append(dep.Blocks, issue)
			issue.IsOrphan = false
			issues[num].IsOrphan = false
		}

		for _, match := range blocksRegex.FindAllStringSubmatch(*issue.Body, -1) {
			num := match[len(match)-1]
			if num[0] == '#' {
				num = fmt.Sprintf(`"%s%s"`, issue.FullRepo(), num)
			}
			dep, found := issues[num]
			if !found {
				issue.Errors = append(issue.Errors, fmt.Errorf("dep %q not found", num))
				continue
			}
			issues[num].DependsOn = append(dep.DependsOn, issue)
			issue.Blocks = append(issue.Blocks, dep)
			issue.IsOrphan = false
			issues[num].IsOrphan = false
		}
	}
	for _, issue := range issues {
		if issue.IsDuplicate != 0 {
			issue.Hidden = true
		}
		if issue.PullRequestLinks != nil {
			issue.Hidden = true
		}
		if issue.IsOrphan && !showOrphans {
			issue.Hidden = true
		}
		if issue.IsClosed() && !showClosed {
			issue.Hidden = true
		}
	}
	for _, issue := range issues {
		issue.LinkedWithEpic = !issue.Hidden && (issue.IsEpic() || issue.BlocksAnEpic() || issue.DependsOnAnEpic())
	}
	for _, issue := range issues {
		if !issue.LinkedWithEpic && !showOrphans {
			issue.Hidden = true
		}
	}

	return nil
}

func panicIfErr(err error) {
	if err != nil {
		panic(err)
	}
}

func orphanGraph(issues Issues) (string, error) {
	g := gographviz.NewGraph()
	panicIfErr(g.SetName("G"))
	attrs := map[string]string{}
	attrs["truecolor"] = "true"
	attrs["overlap"] = "false"
	attrs["splines"] = "true"
	attrs["rankdir"] = "RL"
	attrs["ranksep"] = "0.3"
	attrs["nodesep"] = "0.1"
	attrs["margin"] = "0.2"
	attrs["center"] = "true"
	attrs["constraint"] = "false"
	for k, v := range attrs {
		panicIfErr(g.AddAttr("G", k, v))
	}
	panicIfErr(g.SetDir(true))

	repos := map[string]string{}
	for _, issue := range issues {
		if !issue.IsOrphan && issue.LinkedWithEpic {
			issue.Hidden = true
		}
		if issue.Hidden {
			continue
		}
		panicIfErr(issue.AddNodeToGraph(g, fmt.Sprintf("cluster_%s", issue.RepoID())))
		repos[issue.RepoID()] = issue.FullRepo()
	}

	for id, repo := range repos {
		panicIfErr(g.AddSubGraph(
			"G",
			fmt.Sprintf("cluster_%s", id),
			map[string]string{"label": escape(repo), "style": "dashed"},
		))
	}

	// issue relationships
	for _, issue := range issues {
		panicIfErr(issue.AddEdgesToGraph(g))
	}

	// FIXME: sub-cluster by state

	return g.String(), nil
}

func roadmapGraph(issues Issues) (string, error) {
	var (
		invisStyle = map[string]string{"style": "invis"}
		weightMap  = map[int]bool{}
		weights    = []int{}
	)
	for _, issue := range issues {
		if issue.Hidden {
			continue
		}
		weightMap[issue.Weight()] = true
	}
	for weight := range weightMap {
		weights = append(weights, weight)
	}
	sort.Ints(weights)

	// initialize graph
	g := gographviz.NewGraph()
	panicIfErr(g.SetName("G"))
	attrs := map[string]string{}
	attrs["truecolor"] = "true"
	attrs["overlap"] = "false"
	attrs["splines"] = "true"
	attrs["rankdir"] = "RL"
	attrs["ranksep"] = "0.3"
	attrs["nodesep"] = "0.1"
	attrs["margin"] = "0.2"
	attrs["center"] = "true"
	attrs["constraint"] = "false"
	for k, v := range attrs {
		panicIfErr(g.AddAttr("G", k, v))
	}
	panicIfErr(g.SetDir(true))

	// issue nodes
	issueNumbers := []string{}
	for _, issue := range issues {
		issueNumbers = append(issueNumbers, issue.NodeName())
	}
	sort.Strings(issueNumbers)
	for _, id := range issueNumbers {
		issue := issues[id]
		if issue.Hidden {
			continue
		}
		parent := fmt.Sprintf("anon%d", issue.Weight())
		if issue.IsOrphan || !issue.LinkedWithEpic {
			parent = "cluster_orphans"
		}

		panicIfErr(issue.AddNodeToGraph(g, parent))
	}

	// issue relationships
	for _, issue := range issues {
		panicIfErr(issue.AddEdgesToGraph(g))
	}

	// orphans cluster and placeholder
	if showOrphans {
		panicIfErr(g.AddSubGraph("G", "cluster_orphans", map[string]string{"label": "orphans", "style": "dashed"}))
		panicIfErr(g.AddNode("cluster_orphans", "placeholder_orphans", invisStyle))
		panicIfErr(g.AddEdge(
			fmt.Sprintf("placeholder_%d", weights[len(weights)-1]),
			"placeholder_orphans",
			true,
			invisStyle,
		))
	}

	// set weights clusters and placeholders
	for _, weight := range weights {
		clusterName := fmt.Sprintf("anon%d", weight)
		panicIfErr(g.AddSubGraph("G", clusterName, map[string]string{"rank": "same"}))
		//clusterName := fmt.Sprintf("cluster_w%d", weight)
		//panicIfErr(g.AddSubGraph("G", clusterName, map[string]string{"label": fmt.Sprintf("w%d", weight)}))
		panicIfErr(g.AddNode(
			clusterName,
			fmt.Sprintf("placeholder_%d", weight),
			map[string]string{
				"shape": "none",
				"label": fmt.Sprintf(`"weight=%d"`, weight),
			},
		))
	}
	for i := 0; i < len(weights)-1; i++ {
		panicIfErr(g.AddEdge(
			fmt.Sprintf("placeholder_%d", weights[i]),
			fmt.Sprintf("placeholder_%d", weights[i+1]),
			true,
			invisStyle,
		))
	}

	return g.String(), nil
}

func load() (Issues, error) {
	var issues []*Issue
	content, err := ioutil.ReadFile("/tmp/issues.json")
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(content, &issues); err != nil {
		return nil, err
	}
	m := make(Issues)
	for _, issue := range issues {
		m[issue.NodeName()] = issue
	}
	return m, nil
}

func fetch() error {
	log.Print("fetching new issues")
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	var (
		wg        sync.WaitGroup
		allIssues []*github.Issue
		out       = make(chan []*github.Issue, 100)
	)
	wg.Add(len(repos))
	for _, repo := range repos {
		go func(repo string) {
			total := 0
			defer wg.Done()
			opts := &github.IssueListByRepoOptions{State: "all"}
			for {
				issues, resp, err := client.Issues.ListByRepo(ctx, organization, repo, opts)
				if err != nil {
					log.Fatal(err)
					return
				}
				total += len(issues)
				log.Printf("repo:%s new-issues:%d total:%d", repo, len(issues), total)
				out <- issues
				if resp.NextPage == 0 {
					break
				}
				opts.Page = resp.NextPage
			}
		}(repo)
	}
	wg.Wait()
	close(out)
	for issues := range out {
		allIssues = append(allIssues, issues...)
	}

	issuesJson, _ := json.MarshalIndent(allIssues, "", "  ")
	rateLimits, _, err := client.RateLimits(ctx)
	if err != nil {
		return err
	}
	log.Printf("GitHub API Rate limit: %s", rateLimits.GetCore().String())
	return ioutil.WriteFile("/tmp/issues.json", issuesJson, 0644)
}

func main() {
	if roadmapFetch {
		if err := fetch(); err != nil {
			log.Fatalf("failed to fetch issues: %v", err)
		}
		if onlyFetch {
			os.Exit(0)
		}
	}

	issues, err := load()
	if err != nil {
		log.Fatalf("failed to load issues: %v", err)
	}

	if err = enrich(issues); err != nil {
		log.Fatalf("failed to enrich issues: %v", err)
	}

	// rendering
	var out string
	if onlyOrphans {
		out, err = orphanGraph(issues)
	} else {
		out, err = roadmapGraph(issues)
	}
	if err != nil {
		log.Fatalf("failed to render graph: %v", err)
	}

	fmt.Println(out)

	log.Printf("done (%d issues)", len(issues))
}

func wrap(text string, lineWidth int) string {
	words := strings.Fields(strings.TrimSpace(text))
	if len(words) == 0 {
		return text
	}
	wrapped := words[0]
	spaceLeft := lineWidth - len(wrapped)
	for _, word := range words[1:] {
		if len(word)+1 > spaceLeft {
			wrapped += "\n" + word
			spaceLeft = lineWidth - len(word)
		} else {
			wrapped += " " + word
			spaceLeft -= 1 + len(word)
		}
	}

	return wrapped
}
