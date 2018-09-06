package main

import (
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"

	"github.com/awalterschulze/gographviz"
	"github.com/google/go-github/github"
	"github.com/spf13/viper"
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
		if *label.Name == viper.GetString("epic-label") {
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
	return fmt.Sprintf(`%s#%d`, i.FullRepo(), *i.Number)
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
		//log.Print("edge", i.NodeName(), "->", dependency.NodeName())
		if err := g.AddEdge(
			escape(i.NodeName()),
			escape(dependency.NodeName()),
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
	attrs["href"] = escape(*i.HTMLURL)

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

	//log.Print("node", i.NodeName(), parent)
	return g.AddNode(
		parent,
		escape(i.NodeName()),
		attrs,
	)
}

func (issues Issues) prepare() error {
	var (
		dependsOnRegex, _        = regexp.Compile(`(?i)(require|requires|blocked by|block by|depend on|depends on|parent of) ([a-z/]*#[0-9]+)`)
		blocksRegex, _           = regexp.Compile(`(?i)(blocks|block|address|addresses|part of|child of|fix|fixes) ([a-z/]*#[0-9]+)`)
		isDuplicateRegex, _      = regexp.Compile(`(?i)(duplicates|duplicate|dup of|dup|duplicate of) #([0-9]+)`)
		weightMultiplierRegex, _ = regexp.Compile(`(?i)(depviz.weight_multiplier[:= ]+)([0-9]+)`)
		baseWeightRegex, _       = regexp.Compile(`(?i)(depviz.base_weight[:= ]+)([0-9]+)`)
		hideFromRoadmapRegex, _  = regexp.Compile(`(?i)(depviz.hide_from_roadmap)`) // FIXME: use label
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
				num = fmt.Sprintf(`%s%s`, issue.FullRepo(), num)
			}
			dep, found := issues[num]
			if !found {
				issue.Errors = append(issue.Errors, fmt.Errorf("parent %q not found", num))
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
				num = fmt.Sprintf(`%s%s`, issue.FullRepo(), num)
			}
			dep, found := issues[num]
			if !found {
				issue.Errors = append(issue.Errors, fmt.Errorf("child %q not found", num))
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
	}
	issues.processEpicLinks()
	return nil
}

func (issues Issues) processEpicLinks() {
	for _, issue := range issues {
		issue.LinkedWithEpic = !issue.Hidden && (issue.IsEpic() || issue.BlocksAnEpic() || issue.DependsOnAnEpic())

	}
}

func (issues Issues) HideClosed() {
	for _, issue := range issues {
		if issue.IsClosed() {
			issue.Hidden = true
		}
	}
}

func (issues Issues) HideOrphans() {
	for _, issue := range issues {
		if issue.IsOrphan {
			issue.Hidden = true
		}
	}
}
