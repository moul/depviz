package repo

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/lib/pq"
	"moul.io/depviz/pkg/airtableDB"
)

//
// Base
//

type Base struct {
	ID        string         `gorm:"primary_key" json:"id"`
	CreatedAt time.Time      `json:"created-at,omitempty"`
	UpdatedAt time.Time      `json:"updated-at,omitempty"`
	Errors    pq.StringArray `json:"errors,omitempty" gorm:"type:varchar[]"`
}

//
// Repository
//

type Repository struct {
	Base

	// base fields
	URL         string    `json:"url"`
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

func (p Repository) ToRecord(cache airtableDB.DB) *airtableDB.RepositoryRecord {
	record := airtableDB.RepositoryRecord{}

	// base
	record.Fields.ID = p.ID
	record.Fields.CreatedAt = p.CreatedAt
	record.Fields.UpdatedAt = p.UpdatedAt
	record.Fields.Errors = strings.Join(p.Errors, ", ")

	// specific
	record.Fields.URL = p.URL
	record.Fields.Title = p.Title
	record.Fields.Description = p.Description
	record.Fields.Homepage = p.Homepage
	record.Fields.PushedAt = p.PushedAt
	record.Fields.IsFork = p.IsFork

	// relationships
	record.Fields.Provider = []string{cache.Providers.ByID(p.Provider.ID)}
	if p.Owner != nil {
		record.Fields.Owner = []string{cache.Accounts.ByID(p.Owner.ID)}
	}

	return &record
}

//
// Provider
//

type ProviderDriver string

const (
	UnknownProviderDriver ProviderDriver = "unknown"
	GithubDriver                         = "github"
	GitlabDriver                         = "gitlab"
)

type Provider struct {
	Base

	// base fields
	URL    string `json:"url"`
	Driver string `json:"driver"` // github, gitlab, unknown
}

func (p Provider) ToRecord(cache airtableDB.DB) *airtableDB.ProviderRecord {
	record := airtableDB.ProviderRecord{}

	// base
	record.Fields.ID = p.ID
	record.Fields.CreatedAt = p.CreatedAt
	record.Fields.UpdatedAt = p.UpdatedAt
	record.Fields.Errors = strings.Join(p.Errors, ", ")

	// specific
	record.Fields.URL = p.URL
	record.Fields.Driver = p.Driver

	// relationships
	// n/a

	return &record
}

//
// Milestone
//

type Milestone struct {
	Base

	// base fields
	URL         string    `json:"url"`
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

func (p Milestone) ToRecord(cache airtableDB.DB) *airtableDB.MilestoneRecord {
	record := airtableDB.MilestoneRecord{}
	// base
	record.Fields.ID = p.ID
	record.Fields.CreatedAt = p.CreatedAt
	record.Fields.UpdatedAt = p.UpdatedAt
	record.Fields.Errors = strings.Join(p.Errors, ", ")

	// specific
	record.Fields.URL = p.URL
	record.Fields.Title = p.Title
	record.Fields.Description = p.Description
	record.Fields.ClosedAt = p.ClosedAt
	record.Fields.DueOn = p.DueOn

	// relationships
	if p.Creator != nil {
		record.Fields.Creator = []string{cache.Accounts.ByID(p.Creator.ID)}
	}
	if p.Repository != nil {
		record.Fields.Repository = []string{cache.Repositories.ByID(p.Repository.ID)}
	}

	return &record
}

//
// Issue
//

type Issue struct {
	Base

	// base fields
	URL         string    `json:"url"`
	CompletedAt time.Time `json:"completed-at"`
	Title       string    `json:"title"`
	State       string    `json:"state"`
	Body        string    `json:"body"`
	IsPR        bool      `json:"is-pr"`
	IsLocked    bool      `json:"is-locked"`
	Comments    int       `json:"comments"`
	Upvotes     int       `json:"upvotes"`
	Downvotes   int       `json:"downvotes"`
	IsOrphan    bool      `json:"is-orphan"`
	IsHidden    bool      `json:"is-hidden"`
	Weight      int       `json:"weight"`
	IsEpic      bool      `json:"is-epic"`
	HasEpic     bool      `json:"has-epic"`

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
	Duplicates   []*Issue    `json:"-" gorm:"many2many:issue_duplicates;association_jointable_foreignkey:duplicate_id"`

	// internal
	ChildIDs     []string `json:"child_ids" gorm:"-"`
	ParentIDs    []string `json:"parent_ids" gorm:"-"`
	DuplicateIDs []string `json:"duplicate_ids" gorm:"-"`
}

func (i Issue) String() string {
	out, _ := json.Marshal(i)
	return string(out)
}

func (p Issue) ToRecord(cache airtableDB.DB) *airtableDB.IssueRecord {
	record := airtableDB.IssueRecord{}
	// base
	record.Fields.ID = p.ID
	record.Fields.CreatedAt = p.CreatedAt
	record.Fields.UpdatedAt = p.UpdatedAt
	record.Fields.Errors = strings.Join(p.Errors, ", ")

	// specific
	record.Fields.URL = p.URL
	record.Fields.CompletedAt = p.CompletedAt
	record.Fields.Title = p.Title
	record.Fields.State = p.State
	record.Fields.Body = p.Body
	record.Fields.IsPR = p.IsPR
	record.Fields.IsLocked = p.IsLocked
	record.Fields.Comments = p.Comments
	record.Fields.Upvotes = p.Upvotes
	record.Fields.Downvotes = p.Downvotes
	record.Fields.IsOrphan = p.IsOrphan
	record.Fields.IsHidden = p.IsHidden
	record.Fields.Weight = p.Weight
	record.Fields.IsEpic = p.IsEpic
	record.Fields.HasEpic = p.HasEpic

	// relationships
	record.Fields.Repository = []string{cache.Repositories.ByID(p.Repository.ID)}
	if p.Milestone != nil {
		record.Fields.Milestone = []string{cache.Milestones.ByID(p.Milestone.ID)}
	}
	record.Fields.Author = []string{cache.Accounts.ByID(p.Author.ID)}
	record.Fields.Labels = []string{}
	for _, label := range p.Labels {
		record.Fields.Labels = append(record.Fields.Labels, cache.Labels.ByID(label.ID))
	}
	record.Fields.Assignees = []string{}
	for _, assignee := range p.Assignees {
		record.Fields.Assignees = append(record.Fields.Assignees, cache.Accounts.ByID(assignee.ID))
	}

	return &record
}

//
// Label
//

type Label struct {
	Base

	// base fields
	URL         string `json:"url"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
}

func (p Label) ToRecord(cache airtableDB.DB) *airtableDB.LabelRecord {
	record := airtableDB.LabelRecord{}

	// base
	record.Fields.ID = p.ID
	record.Fields.CreatedAt = p.CreatedAt
	record.Fields.UpdatedAt = p.UpdatedAt
	record.Fields.Errors = strings.Join(p.Errors, ", ")

	// specific
	record.Fields.URL = p.URL
	record.Fields.Name = p.Name
	record.Fields.Color = p.Color
	record.Fields.Description = p.Description

	// relationships
	// n/a

	return &record
}

//
// Account
//

type Account struct {
	Base

	// base fields
	URL       string `json:"url"`
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

func (p Account) ToRecord(cache airtableDB.DB) *airtableDB.AccountRecord {
	record := airtableDB.AccountRecord{}
	// base
	record.Fields.ID = p.ID
	record.Fields.CreatedAt = p.CreatedAt
	record.Fields.UpdatedAt = p.UpdatedAt
	record.Fields.Errors = strings.Join(p.Errors, ", ")

	// specific
	record.Fields.URL = p.URL
	record.Fields.Login = p.Login
	record.Fields.FullName = p.FullName
	record.Fields.Type = p.Type
	record.Fields.Bio = p.Bio
	record.Fields.Location = p.Location
	record.Fields.Company = p.Company
	record.Fields.Blog = p.Blog
	record.Fields.Email = p.Email
	record.Fields.AvatarURL = p.AvatarURL

	// relationships
	record.Fields.Provider = []string{cache.Providers.ByID(p.Provider.ID)}

	return &record
}

// FIXME: create a User struct to handle multiple accounts and aliases
