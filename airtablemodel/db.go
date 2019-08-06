package airtablemodel // import "moul.io/depviz/airtablemodel"

import "moul.io/depviz/airtabledb"

// Unfortunately, the order matters here.
// We must first compute Records which are referenced by other Records...
const (
	ProviderIndex = iota
	LabelIndex
	AccountIndex
	RepositoryIndex
	MilestoneIndex
	IssueIndex
	NumTables
)

var (
	TableNameToIndex = map[string]int{
		"provider":   ProviderIndex,
		"label":      LabelIndex,
		"account":    AccountIndex,
		"repository": RepositoryIndex,
		"milestone":  MilestoneIndex,
		"issue":      IssueIndex,
	}
)

func NewDB() airtabledb.DB {
	db := airtabledb.DB{
		Tables: make([]airtabledb.Table, NumTables),
	}
	db.Tables[IssueIndex].Elems = &[]IssueRecord{}
	db.Tables[RepositoryIndex].Elems = &[]RepositoryRecord{}
	db.Tables[AccountIndex].Elems = &[]AccountRecord{}
	db.Tables[LabelIndex].Elems = &[]LabelRecord{}
	db.Tables[MilestoneIndex].Elems = &[]MilestoneRecord{}
	db.Tables[ProviderIndex].Elems = &[]ProviderRecord{}
	if len(db.Tables) != NumTables {
		panic("missing an airtabledb Table")
	}
	return db
}
