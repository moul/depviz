package compute

import (
	"github.com/jinzhu/gorm"
	"go.uber.org/zap"
	"moul.io/depviz/sql"
	"moul.io/multipmuri"
)

type multipmuriRepo interface {
	RepoEntity() multipmuri.Entity
}

type multipmuriOwner interface {
	OwnerEntity() multipmuri.Entity
}

type multipmuriService interface {
	ServiceEntity() multipmuri.Entity
}

// FIXME: loadIssuesByAuthor
// FIXME: handle github search

func LoadIssuesByTargets(db *gorm.DB, targets []multipmuri.Entity) (*Computed, error) {
	byRepo := []string{}
	byOwner := []string{}
	byService := []string{}
	useFilters := true
	for _, target := range targets {
		switch v := target.(type) {
		case multipmuriRepo:
			byRepo = append(byRepo, v.RepoEntity().String())
		case multipmuriOwner:
			byOwner = append(byOwner, v.OwnerEntity().String())
		case multipmuriService:
			byService = append(byService, v.ServiceEntity().String())
		default:
			zap.L().Warn("unsupported target filter", zap.Any("target", target))
			useFilters = false
		}
	}

	// FIXME: add a owner field on issue
	filteredDB := db
	if useFilters {
		filteredDB = filteredDB.Where(
			"repository_id IN (?) OR repository_owner_id IN (?) OR service_id IN (?)",
			byRepo,
			byOwner,
			byService,
		)
	}

	allIssues, err := sql.LoadAllIssues(filteredDB)
	if err != nil {
		return nil, err
	}

	computed := Compute(allIssues)
	computed.FilterByTargets(targets) // in most cases, this step is optional as we are already filtering by targets when querying the database

	return &computed, nil
}
