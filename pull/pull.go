package pull

import (
	"encoding/json"
	"sync"

	"github.com/jinzhu/gorm"

	"go.uber.org/zap"
	"moul.io/depviz/github"
	"moul.io/depviz/gitlab"
	"moul.io/depviz/model"
	"moul.io/depviz/sql"
	"moul.io/multipmuri"
)

type Options struct {
	// FIXME: find a way of handling multiple gitlab/github instances, somethine like .netrc maybe?
	GithubToken string `mapstructure:"github-token"`
	GitlabToken string `mapstructure:"gitlab-token"`

	SQL sql.Options // inherited with sql.GetOptions()

	Targets []multipmuri.Entity `mapstructure:"targets"` // parsed from Args
}

func (opts Options) String() string {
	out, _ := json.Marshal(opts)
	return string(out)
}

func Pull(opts *Options) error {
	zap.L().Debug("pull", zap.Stringer("opts", *opts))

	db, err := sql.FromOpts(&opts.SQL)
	if err != nil {
		return err
	}

	if err := pull(opts, db); err != nil {
		return err
	}
	// FIXME: compute

	return nil
}

func pull(opts *Options, db *gorm.DB) error {
	// FIXME: handle the special '@me' target
	var (
		wg        sync.WaitGroup
		allIssues []*model.Issue
		out       = make(chan []*model.Issue, 101) // chan should always be bigger than the biggest paginate possible
	)

	// parallel fetches
	wg.Add(len(opts.Targets))
	for _, target := range opts.Targets {
		switch target.Provider() {
		case multipmuri.GitHubProvider:
			go github.Pull(target, &wg, opts.GithubToken, db, out)
		case multipmuri.GitLabProvider:
			go gitlab.Pull(target, &wg, opts.GitlabToken, db, out)
		default:
			panic("should not happen")
		}
	}
	go func() {
		wg.Wait()
		close(out)
	}()

	for issues := range out {
		allIssues = append(allIssues, issues...)
	}

	// save
	for _, issue := range allIssues {
		if err := db.Save(issue).Error; err != nil {
			return err
		}
	}

	//return Compute(db)
	return nil
}
