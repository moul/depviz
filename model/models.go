package model // import "moul.io/depviz/model"

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/lib/pq"
	"moul.io/depviz/airtabledb"
	"moul.io/depviz/airtablemodel"
)

var AllModels = []interface{}{
	Repository{},
	Provider{},
	Milestone{},
	Issue{},
	Label{},
	Account{},
}

//
// Base
//

type Base struct {
	ID        string         `gorm:"primary_key" json:"id"`
	URL       string         `json:"url"`
	CreatedAt time.Time      `json:"created-at,omitempty"`
	UpdatedAt time.Time      `json:"updated-at,omitempty"`
	Errors    pq.StringArray `json:"errors,omitempty" gorm:"type:varchar[]"`
}

func (b Base) GetID() string {
	return b.ID
}

//
// Repository
//

type Repository struct {
	Base

	// base fields
	Title       string    `json:"name"`
	Description string    `json:"description"`
	Homepage    string    `json:"homepage"`
	PushedAt    time.Time `json:"pushed-at"`
	IsFork      bool      `json:"fork"`

	// relationships
	Provider   *Provider `json:"provider"`
	ProviderID string    `json:"provider-id"`
	Owner      *Account  `json:"owner"`
	OwnerID    string    `json:"owner-id"`
}

func (r Repository) ToRecord(cache airtabledb.DB) airtabledb.Record {
	record := airtablemodel.RepositoryRecord{}
	toRecord(cache, r, &record)
	return record
}

func (r Repository) String() string {
	out, _ := json.Marshal(r)
	return string(out)
}

//
// Provider
//

type ProviderDriver string

const (
	UnknownProviderDriver ProviderDriver = "unknown"
	GithubDriver          ProviderDriver = "github"
	GitlabDriver          ProviderDriver = "gitlab"
)

type Provider struct {
	Base

	// base fields
	Driver string `json:"driver"` // github, gitlab, unknown
}

func (p Provider) ToRecord(cache airtabledb.DB) airtabledb.Record {
	record := airtablemodel.ProviderRecord{}
	toRecord(cache, p, &record)
	return record
}

func (p Provider) String() string {
	out, _ := json.Marshal(p)
	return string(out)
}

//
// Milestone
//

type Milestone struct {
	Base

	// base fields
	Title       string    `json:"title"`
	Description string    `json:"description"`
	ClosedAt    time.Time `json:"closed-at"`
	DueOn       time.Time `json:"due-on"`
	// State string      // FIXME: todo
	// StartAt time.Time // FIXME: todo

	// relationships
	Creator      *Account    `json:"creator"`
	CreatorID    string      `json:"creator-id"`
	Repository   *Repository `json:"repository"`
	RepositoryID string      `json:"repository-id"`
}

func (m Milestone) ToRecord(cache airtabledb.DB) airtabledb.Record {
	record := airtablemodel.MilestoneRecord{}
	toRecord(cache, m, &record)
	return record
}

func (m Milestone) String() string {
	out, _ := json.Marshal(m)
	return string(out)
}

//
// Issue
//

type Issue struct {
	Base

	// base fields
	CompletedAt  time.Time `json:"completed-at"`
	Title        string    `json:"title"`
	State        string    `json:"state"`
	Body         string    `json:"body"`
	IsPR         bool      `json:"is-pr"`
	IsLocked     bool      `json:"is-locked"`
	NumComments  int       `json:"num-comments"`
	NumUpvotes   int       `json:"num-upvotes"`
	NumDownvotes int       `json:"num-downvotes"`
	IsOrphan     bool      `json:"is-orphan"`
	IsHidden     bool      `json:"is-hidden"`

	// relationships
	Repository   *Repository `json:"repository"`
	RepositoryID string      `json:"repository_id"`
	Milestone    *Milestone  `json:"milestone"`
	MilestoneID  string      `json:"milestone_id"`
	Author       *Account    `json:"author"`
	AuthorID     string      `json:"author_id"`
	Labels       []*Label    `gorm:"many2many:issue_labels" json:"labels"`
	Assignees    []*Account  `gorm:"many2many:issue_assignees" json:"assignees"`
	Parents      []*Issue    `json:"-" gorm:"many2many:issue_parents;association_jointable_foreignkey:parent_id"`
	Children     []*Issue    `json:"-" gorm:"many2many:issue_children;association_jointable_foreignkey:child_id"`
	Related      []*Issue    `json:"-" gorm:"many2many:issue_related;association_jointable_foreignkey:related_id"`
}

func (i Issue) String() string {
	out, _ := json.Marshal(i)
	return string(out)
}

func (i Issue) ToRecord(cache airtabledb.DB) airtabledb.Record {
	record := airtablemodel.IssueRecord{}
	toRecord(cache, i, &record)
	return record
}

func (i *Issue) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

//
// Issues
//

type Issues []*Issue

//
// Label
//

type Label struct {
	Base

	// base fields
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
}

func (l Label) ToRecord(cache airtabledb.DB) airtabledb.Record {
	record := airtablemodel.LabelRecord{}
	toRecord(cache, l, &record)
	return record
}

func (l Label) String() string {
	out, _ := json.Marshal(l)
	return string(out)
}

//
// Account
//

type Account struct {
	Base

	// base fields
	Login     string `json:"login"`
	FullName  string `json:"fullname"`
	Type      string `json:"type"`
	Bio       string `json:"bio"`
	Location  string `json:"location"`
	Company   string `json:"company"`
	Blog      string `json:"blog"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar-url"`

	// relationships
	Provider   *Provider `json:"provider"`
	ProviderID string    `json:"provider-id"`
}

func (a Account) ToRecord(cache airtabledb.DB) airtabledb.Record {
	record := airtablemodel.AccountRecord{}
	toRecord(cache, a, &record)
	return record
}

func (a Account) String() string {
	out, _ := json.Marshal(a)
	return string(out)
}

// FIXME: create a User struct to handle multiple accounts and aliases
