package main

import (
	"fmt"
	"html"
	"net/url"
	"strings"

	"github.com/awalterschulze/gographviz"
	"github.com/google/go-github/github"
	"github.com/spf13/viper"
	gitlab "github.com/xanzy/go-gitlab"
)

type Issue struct {
	// proxy
	GitHub *github.Issue
	GitLab *gitlab.Issue

	// internal
	Provider         Provider
	DependsOn        IssueSlice
	Blocks           IssueSlice
	weightMultiplier int
	BaseWeight       int
	IsOrphan         bool
	Hidden           bool
	Duplicates       []string
	LinkedWithEpic   bool
	Errors           []error

	// mapping
	Number    int
	Title     string
	State     string
	Body      string
	RepoURL   string
	URL       string
	Labels    []*IssueLabel
	Assignees []*Profile
}

type IssueLabel struct {
	Name  string
	Color string
}

type Profile struct {
	Name     string
	Username string
}

func FromGitHubIssue(input *github.Issue) *Issue {
	body := ""
	if input.Body != nil {
		body = *input.Body
	}
	issue := &Issue{
		Provider:  GitHubProvider,
		GitHub:    input,
		Number:    *input.Number,
		Title:     *input.Title,
		State:     *input.State,
		Body:      body,
		URL:       *input.HTMLURL,
		RepoURL:   *input.RepositoryURL,
		Labels:    make([]*IssueLabel, 0),
		Assignees: make([]*Profile, 0),
	}
	for _, label := range input.Labels {
		issue.Labels = append(issue.Labels, &IssueLabel{
			Name:  *label.Name,
			Color: *label.Color,
		})
	}
	for _, assignee := range input.Assignees {
		name := *assignee.Login
		if assignee.Name != nil {
			name = *assignee.Name
		}
		issue.Assignees = append(issue.Assignees, &Profile{
			Name:     name,
			Username: *assignee.Login,
		})
	}
	return issue
}

func FromGitLabIssue(input *gitlab.Issue) *Issue {
	issue := &Issue{
		Provider:  GitLabProvider,
		GitLab:    input,
		Number:    input.IID,
		Title:     input.Title,
		State:     input.State,
		URL:       input.WebURL,
		Body:      input.Description,
		RepoURL:   input.Links.Project,
		Labels:    make([]*IssueLabel, 0),
		Assignees: make([]*Profile, 0),
	}
	for _, label := range input.Labels {
		issue.Labels = append(issue.Labels, &IssueLabel{
			Name:  label,
			Color: "cccccc",
		})
	}
	for _, assignee := range input.Assignees {
		issue.Assignees = append(issue.Assignees, &Profile{
			Name:     assignee.Name,
			Username: assignee.Username,
		})
	}
	return issue
}

func (i Issue) Path() string {
	u, err := url.Parse(i.URL)
	if err != nil {
		return ""
	}
	parts := strings.Split(u.Path, "/")
	return strings.Join(parts[:len(parts)-2], "/")
}

func (i Issue) ProviderURL() string {
	u, _ := url.Parse(i.URL)
	return fmt.Sprintf("%s://%s", u.Scheme, u.Host)
}

func (i Issue) IsEpic() bool {
	for _, label := range i.Labels {
		if label.Name == viper.GetString("epic-label") {
			return true
		}
	}
	return false
	//return !i.IsOrphan && len(i.Blocks) == 0
}

func (i Issue) Repo() string {
	return strings.Split(i.URL, "/")[5]
}

func (i Issue) RepoID() string {
	id := i.Path()[1:]
	id = strings.Replace(id, "/", "", -1)
	id = strings.Replace(id, "-", "", -1)
	return id
}

func (i Issue) Owner() string {
	return strings.Split(i.URL, "/")[4]
}

func (i Issue) IsClosed() bool {
	return i.State == "closed"
}

func (i Issue) IsReady() bool {
	return !i.IsOrphan && len(i.DependsOn) == 0
}

func (i Issue) NodeName() string {
	return fmt.Sprintf(`%s#%d`, i.Path()[1:], i.Number)
}

func (i Issue) NodeTitle() string {
	title := fmt.Sprintf("%s: %s", i.NodeName(), i.Title)
	title = strings.Replace(title, "|", "-", -1)
	title = strings.Replace(html.EscapeString(wrap(title, 20)), "\n", "<br/>", -1)
	labels := []string{}
	for _, label := range i.Labels {
		switch label.Name {
		case "t/step", "t/epic":
			continue
		}
		labels = append(labels, fmt.Sprintf(`<td bgcolor="#%s">%s</td>`, label.Color, label.Name))
	}
	labelsText := ""
	if len(labels) > 0 {
		labelsText = "<tr><td><table><tr>" + strings.Join(labels, "") + "</tr></table></td></tr>"
	}
	assigneeText := ""
	if len(i.Assignees) > 0 {
		assignees := []string{}
		for _, assignee := range i.Assignees {
			assignees = append(assignees, assignee.Username)
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

func (i Issue) GetRelativeIssueURL(target string) string {
	if strings.Contains(target, "://") {
		return target
	}

	u, err := url.Parse(target)
	if err != nil {
		return ""
	}
	path := u.Path
	if path == "" {
		path = i.Path()
	}

	return fmt.Sprintf("%s%s/issues/%s", i.ProviderURL(), path, u.Fragment)
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
	for _, dep := range i.Blocks.Unique() {
		weight += dep.Weight()
	}
	return weight * i.WeightMultiplier()
}

func (i Issue) WeightMultiplier() int {
	multiplier := i.weightMultiplier
	for _, dep := range i.Blocks.Unique() {
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
		//log.Print("edge", escape(i.URL), "->", escape(dependency.URL))
		if err := g.AddEdge(
			escape(i.URL),
			escape(dependency.URL),
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
	attrs["href"] = escape(i.URL)

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

	return g.AddNode(
		parent,
		escape(i.URL),
		attrs,
	)
}

func (i Issue) IsPR() bool {
	switch i.Provider {
	case GitHubProvider:
		return i.GitHub.PullRequestLinks != nil
	case GitLabProvider:
		return false // only fetching issues for now
	}
	panic("should not happen")
}
