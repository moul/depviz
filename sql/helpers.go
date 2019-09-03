package sql

import (
	"github.com/jinzhu/gorm"
	"go.uber.org/zap"
	"moul.io/depviz/model"
)

func LoadAllIssues(db *gorm.DB) (model.Issues, error) {
	query := db.Model(model.Issue{}).Order("created_at")
	perPage := 100
	var allIssues model.Issues
	for page := 0; ; page++ {
		var newIssues []*model.Issue
		if err := query.Limit(perPage).Offset(perPage * page).Find(&newIssues).Error; err != nil {
			return nil, err
		}
		allIssues = append(allIssues, newIssues...)
		if len(newIssues) < perPage {
			break
		}
	}
	zap.L().Debug("fetched issues", zap.Int("quantity", len(allIssues)))
	return allIssues, nil
}
