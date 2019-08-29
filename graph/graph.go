package graph // import "moul.io/depviz/graph"

import (
	"encoding/json"
	"fmt"

	"github.com/jinzhu/gorm"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
	"moul.io/depviz/compute"
	"moul.io/depviz/sql"
	"moul.io/graphman"
	"moul.io/graphman/viz"
	"moul.io/multipmuri"
)

type Options struct {
	SQL             sql.Options         `mapstructure:"sql"`     // inherited with sql.GetOptions()
	Targets         []multipmuri.Entity `mapstructure:"targets"` // parsed from Args
	ShowClosed      bool                `mapstructure:"show-closed"`
	ShowOrphans     bool                `mapstructure:"show-orphans"`
	ShowPRs         bool                `mapstructure:"show-prs"`
	ShowAllRelated  bool                `mapstructure:"show-all-related"`
	NoPertEstimates bool                `mapstructure:"no-pert-estimates"`
	Vertical        bool                `mapstructure:"vertical"`
	Format          string              `mapstructure:"format"`
}

func (opts Options) String() string {
	out, _ := json.Marshal(opts)
	return string(out)
}

func Graph(opts *Options) error {
	zap.L().Debug("graph", zap.Stringer("opts", *opts))

	db, err := sql.FromOpts(&opts.SQL)
	if err != nil {
		return err
	}

	if err := graph(opts, db); err != nil {
		return err
	}

	return nil
}

func graph(opts *Options, db *gorm.DB) error {
	issues, err := sql.LoadAllIssues(db)
	if err != nil {
		return err
	}

	// compute and filter issues
	computed := compute.Compute(issues)
	computed.FilterByTargets(opts.Targets)
	// FIXME: if !opts.ShowOrphans { computed.FilterOrphans() }
	// FIXME: if !opts.ShowAllRelated { computed.FilterAllRelated()
	// FIXME: if !opts.ShowPRs { computed.FilterPRs()
	// FIXME: if !opts.ShowClosed { computed.FilterClosed()

	// initialize graph config
	config := graphman.PertConfig{
		Actions: []graphman.PertAction{},
		States:  []graphman.PertState{},
	}
	config.Opts.NoSimplify = false

	// process computed issues
	for _, issue := range computed.Issues {
		// fmt.Println(issue.Hidden, issue.URL)
		if issue.Hidden {
			continue
		}
		config.Actions = append(
			config.Actions,
			graphman.PertAction{
				ID:        issue.URL,
				Title:     issue.Title,
				DependsOn: issue.DependsOn,
				// Estimate
				// FIXME: set style based on type, active, etc
			},
		)
	}
	for _, milestone := range computed.Milestones {
		config.States = append(
			config.States,
			graphman.PertState{
				ID:        milestone.URL,
				Title:     milestone.Title,
				DependsOn: milestone.DependsOn,
			},
		)
	}
	if len(computed.Repos) > 1 {
		// FIXME: alternative layout with repo a bordered subgraph
		for _, repo := range computed.Repos {
			config.States = append(
				config.States,
				graphman.PertState{
					ID:        repo.URL,
					Title:     fmt.Sprintf("Repo %q", repo.URL),
					DependsOn: repo.DependsOn,
				},
			)
		}
	}

	if opts.Format == "graphman-pert" {
		out, err := yaml.Marshal(config)
		if err != nil {
			return err
		}
		fmt.Println(string(out))
		return nil
	}

	// initialize graph from config
	graph := graphman.FromPertConfig(config)
	if !opts.NoPertEstimates {
		_ = graphman.ComputePert(graph)
		//for _, e := range graph.Edges() {log.Println("*", e)}
	}

	// graph fine tuning
	graph.GetVertex("Start").SetColor("blue")
	graph.GetVertex("Finish").SetColor("blue")
	if opts.Vertical {
		graph.Attrs["rankdir"] = "TB"
	}
	//graph.Attrs["size"] = "\"11,11\""
	graph.Attrs["overlap"] = "false"
	graph.Attrs["pack"] = "true"
	graph.Attrs["splines"] = "true"
	// graph.Attrs["layout"] = "neato"
	graph.Attrs["sep"] = "0.1"
	// graph.Attrs["start"] = "random"
	// FIXME: hightlight critical paths
	// FIXME: highlight other infos
	// FIXME: highlight target

	// graphviz
	s, err := viz.ToGraphviz(graph, &viz.Opts{
		CommentsInLabel: true,
	})
	if err != nil {
		return err
	}
	fmt.Println(s)

	return nil
}

/*

func graph(opts *graphOptions) error {
	zap.L().Debug("graph", zap.Stringer("opts", *opts))
	issues, err := model.Load(db, nil)
	if err != nil {
		return errors.Wrap(err, "failed to load issues")
	}
	filtered := issues.FilterByTargets(opts.Targets)

	out, err := graphviz(filtered, opts)
	if err != nil {
		return errors.Wrap(err, "failed to render graph")
	}

	switch opts.Format {
	case "png", "svg":
		return fmt.Errorf("only supporting .dot format for now")

	//case "dot":
	default:
	}

	var dest io.WriteCloser
	switch opts.Output {
	case "-", "":
		dest = os.Stdout
	default:
		var err error
		dest, err = os.Create(opts.Output)
		if err != nil {
			return err
		}
		defer dest.Close()
	}
	fmt.Fprintln(dest, out)
	return nil
}

func isIssueHidden(issue *model.Issue, opts *graphOptions) bool {
	if issue.IsHidden {
		return true
	}
	if !opts.ShowClosed && issue.IsClosed() {
		return true
	}
	if !opts.ShowOrphans && issue.IsOrphan {
		return true
	}
	if !opts.ShowPRs && issue.IsPR {
		return true
	}
	return false
}

func graphviz(issues model.Issues, opts *graphOptions) (string, error) {
	for _, issue := range issues {
		if isIssueHidden(issue, opts) {
			continue
		}
	}

	var (
		stats = map[string]int{
			"nodes":     0,
			"edges":     0,
			"hidden":    0,
			"subgraphs": 0,
		}
		invisStyle    = map[string]string{"style": "invis", "label": escape("")}
		weightMap     = map[int]bool{}
		weights       = []int{}
		existingNodes = map[string]bool{}
	)
	if opts.DebugGraph {
		invisStyle = map[string]string{}
	}
	for _, issue := range issues {
		if isIssueHidden(issue, opts) {
			stats["hidden"]++
			continue
		}
		weightMap[issue.Weight] = true
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
	attrs["rankdir"] = "RL"
	attrs["constraint"] = "true"
	attrs["compound"] = "true"
	if !opts.NoCompress {
		attrs["center"] = "true"
		attrs["ranksep"] = "0.3"
		attrs["nodesep"] = "0.1"
		attrs["margin"] = "0.2"
		attrs["sep"] = "-0.7"
		attrs["constraint"] = "false"
		attrs["splines"] = "true"
		attrs["overlap"] = "compress"
	}
	if opts.DarkTheme {
		attrs["bgcolor"] = "black"
	}

	for k, v := range attrs {
		panicIfErr(g.AddAttr("G", k, v))
	}
	panicIfErr(g.SetDir(true))

	// issue nodes
	issueNumbers := []string{}
	for _, issue := range issues {
		issueNumbers = append(issueNumbers, issue.URL)
	}
	sort.Strings(issueNumbers)

	orphansWithoutLinks := 0
	for _, id := range issueNumbers {
		issue := issues.Get(id)
		if isIssueHidden(issue, opts) {
			continue
		}
		if len(issue.Parents) == 0 && len(issue.Children) == 0 {
			orphansWithoutLinks++
		}
	}
	orphansCols := int(math.Ceil(math.Sqrt(float64(orphansWithoutLinks)) / 2))
	colIndex := 0
	hasOrphansWithLinks := false
	for _, id := range issueNumbers {
		issue := issues.Get(id)
		if isIssueHidden(issue, opts) {
			continue
		}
		parent := fmt.Sprintf("cluster_weight_%d", issue.Weight)
		if issue.IsOrphan || !issue.HasEpic {
			if len(issue.Children) > 0 || len(issue.Parents) > 0 {
				parent = "cluster_orphans_with_links"
				hasOrphansWithLinks = true
			} else {
				parent = fmt.Sprintf("cluster_orphans_without_links_%d", colIndex%orphansCols)
				colIndex++
			}
		}

		existingNodes[issue.URL] = true
		panicIfErr(AddNodeToGraph(g, issue, parent))
		stats["nodes"]++
	}

	// issue relationships
	for _, issue := range issues {
		panicIfErr(AddEdgesToGraph(g, issue, opts, existingNodes))
		stats["edges"]++
	}

	// orphans cluster and placeholder
	if orphansWithoutLinks > 0 {
		panicIfErr(g.AddSubGraph(
			"G",
			"cluster_orphans_without_links",
			map[string]string{"label": escape("orphans without links"), "style": "dashed"},
		))
		stats["subgraphs"]++

		panicIfErr(g.AddSubGraph(
			"cluster_orphans_without_links",
			"cluster_orphans_without_links_0",
			invisStyle,
		))
		stats["subgraphs"]++
		for i := 0; i < orphansCols; i++ {
			panicIfErr(g.AddNode(
				fmt.Sprintf("cluster_orphans_without_links_%d", i),
				fmt.Sprintf("placeholder_orphans_without_links_%d", i),
				invisStyle,
			))
			stats["nodes"]++
		}

		panicIfErr(g.AddEdge(
			fmt.Sprintf("placeholder_%d", weights[len(weights)-1]),
			"placeholder_orphans_without_links_0",
			true,
			invisStyle,
		))
		stats["edges"]++

		for i := 1; i < orphansCols; i++ {
			panicIfErr(g.AddSubGraph(
				"cluster_orphans_without_links",
				fmt.Sprintf("cluster_orphans_without_links_%d", i),
				invisStyle,
			))
			stats["subgraphs"]++
			panicIfErr(g.AddEdge(
				fmt.Sprintf("placeholder_orphans_without_links_%d", i-1),
				fmt.Sprintf("placeholder_orphans_without_links_%d", i),
				true,
				invisStyle,
			))
			stats["edges"]++
		}
	}
	if hasOrphansWithLinks {
		attrs := map[string]string{}
		attrs["label"] = escape("orphans with links")
		attrs["style"] = "dashed"
		panicIfErr(g.AddSubGraph("G", "cluster_orphans_with_links", attrs))
		stats["subgraphs"]++

		panicIfErr(g.AddNode("cluster_orphans_with_links", "placeholder_orphans_with_links", invisStyle))
		stats["nodes"]++

		panicIfErr(g.AddEdge(
			"placeholder_orphans_with_links",
			fmt.Sprintf("placeholder_%d", weights[0]),
			true,
			invisStyle,
		))
		stats["edges"]++
	}

	// set weights clusters and placeholders
	for _, weight := range weights {
		clusterName := fmt.Sprintf("cluster_weight_%d", weight)
		attrs := invisStyle
		attrs["rank"] = "same"
		panicIfErr(g.AddSubGraph("G", clusterName, attrs))
		stats["subgraphs"]++

		attrs = invisStyle
		attrs["shape"] = "none"
		attrs["label"] = fmt.Sprintf(`"weight=%d"`, weight)
		panicIfErr(g.AddNode(
			clusterName,
			fmt.Sprintf("placeholder_%d", weight),
			attrs,
		))
		stats["nodes"]++
	}
	for i := 0; i < len(weights)-1; i++ {
		panicIfErr(g.AddEdge(
			fmt.Sprintf("placeholder_%d", weights[i]),
			fmt.Sprintf("placeholder_%d", weights[i+1]),
			true,
			invisStyle,
		))
		stats["edges"]++
	}

	zap.L().Debug("graph stats", zap.Any("stats", stats))
	return g.String(), nil
}

func AddNodeToGraph(g *gographviz.Graph, i *model.Issue, parent string) error {
	attrs := map[string]string{}
	attrs["label"] = GraphNodeTitle(i)
	//attrs["xlabel"] = ""
	attrs["shape"] = "record"
	attrs["style"] = `"rounded,filled"`
	attrs["color"] = "lightblue"
	attrs["href"] = escape(i.URL)

	if i.IsEpic {
		attrs["shape"] = "oval"
	}

	switch {

	case i.IsClosed():
		attrs["color"] = `"#cccccc33"`

	case i.IsEpic:
		attrs["color"] = "orange"
		attrs["style"] = `"rounded,filled,bold"`

	case i.IsReady():
		attrs["color"] = "pink"

	case i.IsOrphan || !i.HasEpic:
		attrs["color"] = "gray"
	}

	//logger().Debug("add node to graph", zap.String("url", i.URL))
	return g.AddNode(
		parent,
		escape(i.URL),
		attrs,
	)
}

func AddEdgesToGraph(g *gographviz.Graph, i *model.Issue, opts *graphOptions, existingNodes map[string]bool) error {
	if isIssueHidden(i, opts) {
		return nil
	}
	for _, dependency := range i.Parents {
		if isIssueHidden(dependency, opts) {
			continue
		}
		if _, found := existingNodes[dependency.URL]; !found {
			continue
		}
		attrs := map[string]string{}
		attrs["color"] = "lightblue"
		//attrs["label"] = "depends on"
		//attrs["style"] = "dotted"
		attrs["dir"] = "none"
		if i.IsClosed() || dependency.IsClosed() {
			attrs["color"] = "grey"
			attrs["style"] = "dashed"
		} else if dependency.IsReady() {
			attrs["color"] = "pink"
		}
		if i.IsEpic {
			attrs["color"] = "orange"
			attrs["style"] = "dashed"
		}
		//log.Print("edge", escape(i.URL), "->", escape(dependency.URL))
		//logger().Debug("add edge to graph", zap.String("url", i.URL), zap.String("dep", dependency.URL))
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

func GraphNodeName(i *model.Issue) string {
	return fmt.Sprintf(`%s#%s`, i.Path()[1:], i.Number())
}

func GraphNodeTitle(i *model.Issue) string {
	title := fmt.Sprintf("%s: %s", GraphNodeName(i), i.Title)
	title = strings.Replace(title, "|", "-", -1)
	title = strings.Replace(html.EscapeString(wrap(title, 20)), "\n", "<br/>", -1)
	labels := []string{}
	for _, label := range i.Labels {
		switch label.ID {
		case "t/step", "t/epic", "epic":
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
			assignees = append(assignees, assignee.ID)
		}
		assigneeText = fmt.Sprintf(`<tr><td><font color="purple"><i>@%s</i></font></td></tr>`, strings.Join(assignees, ", @"))
	}
	errorsText := ""
	if len(i.Errors) > 0 {
		errorsText = fmt.Sprintf(`<tr><td bgcolor="red">ERR: %s</td></tr>`, strings.Join(i.Errors, ";<br />ERR: "))
	}
	return fmt.Sprintf(`<<table><tr><td>%s</td></tr>%s%s%s</table>>`, title, labelsText, assigneeText, errorsText)
}

func panicIfErr(err error) {
	if err != nil {
		panic(err)
	}
}

func escape(input string) string {
	return fmt.Sprintf("%q", input)
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
*/
