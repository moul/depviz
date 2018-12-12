package repo

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/jinzhu/gorm"
)

var (
	rxDNSName           = regexp.MustCompile(`^([a-zA-Z0-9_]{1}[a-zA-Z0-9_-]{0,62}){1}(\.[a-zA-Z0-9_]{1}[a-zA-Z0-9_-]{0,62})*[\._]?$`)
	childrenRegex, _    = regexp.Compile(`(?i)(require|requires|blocked by|block by|depend on|depends on|parent of) ([a-z0-9:/_.-]+issues/[0-9]+|[a-z0-9:/_.-]+#[0-9]+|[a-z0-9/_-]*#[0-9]+)`)
	parentsRegex, _     = regexp.Compile(`(?i)(blocks|block|address|addresses|part of|child of|fix|fixes) ([a-z0-9:/_.-]+issues/[0-9]+|[a-z0-9:/_.-]+#[0-9]+|[a-z0-9/_-]*#[0-9]+)`)
	isDuplicateRegex, _ = regexp.Compile(`(?i)(duplicates|duplicate|dup of|dup|duplicate of) ([a-z0-9:/_.-]+issues/[0-9]+|[a-z0-9:/_.-]+#[0-9]+|[a-z0-9/_-]*#[0-9]+)`)
	//weightMultiplierRegex, _ = regexp.Compile(`(?i)(depviz.weight_multiplier[:= ]+)([0-9]+)`)
	weightRegex, _ = regexp.Compile(`(?i)(depviz.base_weight|depviz.weight)[:= ]+([0-9]+)`)
	hideRegex, _   = regexp.Compile(`(?i)(depviz.hide)`) // FIXME: use label
)

// PullAndCompute pulls issues from the given targets, computes their fields, and stores the issues in the database.
func PullAndCompute(githubToken, gitlabToken string, db *gorm.DB, t Targets) error {
	// FIXME: handle the special '@me' target

	var (
		wg        sync.WaitGroup
		allIssues []*Issue
		out       = make(chan []*Issue, 100)
	)

	targets := t.UniqueProjects()

	// parallel fetches
	wg.Add(len(targets))
	for _, target := range targets {
		switch target.Driver() {
		case GithubDriver:
			go githubPull(target, &wg, githubToken, db, out)
		case GitlabDriver:
			go gitlabPull(target, &wg, gitlabToken, db, out)
		default:
			panic("should not happen")
		}
	}
	wg.Wait()
	close(out)
	for issues := range out {
		allIssues = append(allIssues, issues...)
	}

	// save
	for _, issue := range allIssues {
		if err := db.Save(issue).Error; err != nil {
			return err
		}
	}

	return Compute(db)
}

// Compute loads issues from the given database, computes their fields, and stores the issues back into the database.
func Compute(db *gorm.DB) error {
	issues, err := LoadIssues(db, nil)
	if err != nil {
		return err
	}

	for _, issue := range issues {
		// reset default values
		issue.Errors = []string{}
		issue.Parents = []*Issue{}
		issue.Children = []*Issue{}
		issue.Duplicates = []*Issue{}
		issue.Weight = 0
		issue.IsHidden = false
		issue.IsEpic = false
		issue.HasEpic = false
		issue.IsOrphan = true
	}

	for _, issue := range issues {
		if issue.Body == "" {
			continue
		}

		// is epic
		for _, label := range issue.Labels {
			// FIXME: get epic labels dynamically based on a configuration filein the repo
			if label.Name == "epic" || label.Name == "t/epic" {
				issue.IsEpic = true
			}
		}

		// hidden
		if match := hideRegex.FindStringSubmatch(issue.Body); match != nil {
			issue.IsHidden = true
			continue
		}

		// duplicates
		if match := isDuplicateRegex.FindStringSubmatch(issue.Body); match != nil {
			canonical := issue.GetRelativeURL(match[len(match)-1])
			rel := issues.Get(canonical)
			if rel == nil {
				issue.Errors = append(issue.Errors, fmt.Errorf("duplicate %q not found", canonical).Error())
				continue
			}
			issue.Duplicates = append(issue.Duplicates, rel)
			issue.IsHidden = true
			continue
		}

		// weight
		if match := weightRegex.FindStringSubmatch(issue.Body); match != nil {
			issue.Weight, _ = strconv.Atoi(match[len(match)-1])
		}

		// children
		for _, match := range childrenRegex.FindAllStringSubmatch(issue.Body, -1) {
			canonical := issue.GetRelativeURL(match[len(match)-1])
			child := issues.Get(canonical)
			if child == nil {
				issue.Errors = append(issue.Errors, fmt.Errorf("children %q not found", canonical).Error())
				continue
			}
			issue.Children = append(issue.Children, child)
			issue.IsOrphan = false
			child.Parents = append(child.Parents, issue)
			child.IsOrphan = false
		}

		// parents
		for _, match := range parentsRegex.FindAllStringSubmatch(issue.Body, -1) {
			canonical := issue.GetRelativeURL(match[len(match)-1])
			parent := issues.Get(canonical)
			if parent == nil {
				issue.Errors = append(issue.Errors, fmt.Errorf("parent %q not found", canonical).Error())
				continue
			}
			issue.Parents = append(issue.Parents, parent)
			issue.IsOrphan = false
			parent.Children = append(parent.Children, issue)
			parent.IsOrphan = false
		}
	}

	for _, issue := range issues {
		if issue.IsEpic {
			issue.HasEpic = true
			continue
		}
		// has epic
		issue.HasEpic, err = computeHasEpic(issue, 0)
		if err != nil {
			issue.Errors = append(issue.Errors, err.Error())
		}
	}

	for _, issue := range issues {
		issue.PostLoad()

		issue.ParentIDs = uniqueStrings(issue.ParentIDs)
		sort.Strings(issue.ParentIDs)
		issue.ChildIDs = uniqueStrings(issue.ChildIDs)
		sort.Strings(issue.ChildIDs)
		issue.DuplicateIDs = uniqueStrings(issue.DuplicateIDs)
		sort.Strings(issue.DuplicateIDs)
	}

	for _, issue := range issues {
		// TODO: add a "if changed" to preserve some CPU and time
		if err := db.Set("gorm:association_autoupdate", false).Save(issue).Error; err != nil {
			return err
		}
	}

	return nil
}

// LoadIssues returns the issues stored in the database.
func LoadIssues(db *gorm.DB, targets []Target) (Issues, error) {
	query := db.Model(Issue{}).Order("created_at")
	if len(targets) > 0 {
		return nil, fmt.Errorf("not implemented")
		// query = query.Where("repo_url IN (?)", canonicalTargets(targets))
		// OR WHERE parents IN ....
		// etc
	}

	perPage := 100
	var issues []*Issue
	for page := 0; ; page++ {
		var newIssues []*Issue
		if err := query.Limit(perPage).Offset(perPage * page).Find(&newIssues).Error; err != nil {
			return nil, err
		}
		issues = append(issues, newIssues...)
		if len(newIssues) < perPage {
			break
		}
	}

	for _, issue := range issues {
		issue.PostLoad()
	}

	return Issues(issues), nil
}

// FIXME: try to use gorm hooks to auto preload/postload items

func (i *Issue) Number() string {
	u, err := url.Parse(i.URL)
	if err != nil {
		return ""
	}
	parts := strings.Split(u.Path, "/")
	return parts[len(parts)-1]
}

func (i *Issue) Path() string {
	u, err := url.Parse(i.URL)
	if err != nil {
		return ""
	}
	parts := strings.Split(u.Path, "/")
	return strings.Join(parts[:len(parts)-2], "/")
}

func (i *Issue) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func (i Issue) GetRelativeURL(target string) string {
	if strings.Contains(target, "://") {
		return normalizeURL(target)
	}

	if target[0] == '#' {
		return fmt.Sprintf("%s/issues/%s", i.Repository.URL, target[1:])
	}

	target = strings.Replace(target, "#", "/issues/", -1)

	parts := strings.Split(target, "/")
	if strings.Contains(parts[0], ".") && isDNSName(parts[0]) {
		return fmt.Sprintf("https://%s", target)
	}

	return fmt.Sprintf("%s/%s", strings.TrimRight(i.Repository.Provider.URL, "/"), target)
}

func (i *Issue) PostLoad() {
	i.ParentIDs = []string{}
	i.ChildIDs = []string{}
	i.DuplicateIDs = []string{}
	for _, rel := range i.Parents {
		i.ParentIDs = append(i.ParentIDs, rel.ID)
	}
	for _, rel := range i.Children {
		i.ChildIDs = append(i.ChildIDs, rel.ID)
	}
	for _, rel := range i.Duplicates {
		i.DuplicateIDs = append(i.DuplicateIDs, rel.ID)
	}
}

func (i Issue) IsClosed() bool {
	return i.State == "closed"
}

func (i Issue) IsReady() bool {
	return !i.IsOrphan && len(i.Parents) == 0 // FIXME: switch parents with children?
}

func (i Issue) MatchesWithATarget(targets Targets) bool {
	return i.matchesWithATarget(targets, 0)
}

type Issues []*Issue

func (issues Issues) Get(id string) *Issue {
	for _, issue := range issues {
		if issue.ID == id {
			return issue
		}
	}
	return nil
}

func (issues Issues) FilterByTargets(targets []Target) Issues {
	filtered := Issues{}

	for _, issue := range issues {
		if issue.MatchesWithATarget(targets) {
			filtered = append(filtered, issue)
		}
	}

	return filtered
}

func normalizeURL(input string) string {
	parts := strings.Split(input, "://")
	output := fmt.Sprintf("%s://%s", parts[0], strings.Replace(parts[1], "//", "/", -1))
	output = strings.TrimRight(output, "#")
	output = strings.TrimRight(output, "/")
	return output
}

func isDNSName(input string) bool {
	return rxDNSName.MatchString(input)
}

func computeHasEpic(i *Issue, depth int) (bool, error) {
	if depth > 100 {
		return false, fmt.Errorf("very high blocking depth (>100), do not continue. (issue=%s)", i.URL)
	}
	if i.IsHidden {
		return false, nil
	}
	for _, parent := range i.Parents {
		if parent.IsEpic {
			return true, nil
		}
		parentHasEpic, err := computeHasEpic(parent, depth+1)
		if err != nil {
			return false, nil
		}
		if parentHasEpic {
			return true, nil
		}
	}
	return false, nil
}

func uniqueStrings(input []string) []string {
	u := make([]string, 0, len(input))
	m := make(map[string]bool)

	for _, val := range input {
		if _, ok := m[val]; !ok {
			m[val] = true
			u = append(u, val)
		}
	}

	return u
}

func (i Issue) matchesWithATarget(targets Targets, depth int) bool {
	if depth > 100 {
		log.Printf("circular dependency or too deep graph (>100), skipping this node. (issue=%s)", i)
		return false
	}

	for _, target := range targets {
		if target.Issue() != "" { // issue-mode
			if target.Canonical() == i.URL {
				return true
			}
		} else { // project-mode
			if i.RepositoryID == target.ProjectURL() {
				return true
			}
		}
	}

	for _, parent := range i.Parents {
		if parent.matchesWithATarget(targets, depth+1) {
			return true
		}
	}

	for _, child := range i.Children {
		if child.matchesWithATarget(targets, depth+1) {
			return true
		}
	}

	return false
}
