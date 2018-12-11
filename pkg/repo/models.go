package repo

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/lib/pq"
	"moul.io/depviz/pkg/airtabledb"
)

type Feature interface {
	String() string
	GetID() string
	ToRecord(airtabledb.DB) airtabledb.Record
}

//
// Base
//

type Base struct {
	ID        string         `gorm:"primary_key" json:"id"`
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

func (p Repository) ToRecord(cache airtabledb.DB) airtabledb.Record {
	record := airtabledb.Record{}
	features := airtabledb.RepositoryRecord{}

	// base
	record.Fields.ID = p.ID
	record.Fields.CreatedAt = p.CreatedAt
	record.Fields.UpdatedAt = p.UpdatedAt
	record.Fields.Errors = strings.Join(p.Errors, ", ")

	// specific
	features.URL = p.URL
	features.Title = p.Title
	features.Description = p.Description
	features.Homepage = p.Homepage
	features.PushedAt = p.PushedAt
	features.IsFork = p.IsFork

	// relationships
	features.Provider = []string{cache.Tables[airtabledb.ProviderIndex].FindByID(p.Provider.ID)}
	if p.Owner != nil {
		features.Owner = []string{cache.Tables[airtabledb.AccountIndex].FindByID(p.Owner.ID)}
	}

	record.Fields.Feature = features
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
	GithubDriver                         = "github"
	GitlabDriver                         = "gitlab"
)

type Provider struct {
	Base

	// base fields
	URL    string `json:"url"`
	Driver string `json:"driver"` // github, gitlab, unknown
}

func (p Provider) ToRecord(cache airtabledb.DB) airtabledb.Record {
	record := airtabledb.Record{}
	features := airtabledb.ProviderRecord{}

	// base
	record.Fields.ID = p.ID
	record.Fields.CreatedAt = p.CreatedAt
	record.Fields.UpdatedAt = p.UpdatedAt
	record.Fields.Errors = strings.Join(p.Errors, ", ")

	// specific
	features.URL = p.URL
	features.Driver = p.Driver

	// relationships
	// n/a

	record.Fields.Feature = features
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

func (p Milestone) ToRecord(cache airtabledb.DB) airtabledb.Record {
	record := airtabledb.Record{}
	features := airtabledb.MilestoneRecord{}

	// base
	record.Fields.ID = p.ID
	record.Fields.CreatedAt = p.CreatedAt
	record.Fields.UpdatedAt = p.UpdatedAt
	record.Fields.Errors = strings.Join(p.Errors, ", ")

	// specific
	features.URL = p.URL
	features.Title = p.Title
	features.Description = p.Description
	features.ClosedAt = p.ClosedAt
	features.DueOn = p.DueOn

	// relationships
	if p.Creator != nil {
		features.Creator = []string{cache.Tables[airtabledb.AccountIndex].FindByID(p.Creator.ID)}
	}
	if p.Repository != nil {
		features.Repository = []string{cache.Tables[airtabledb.RepositoryIndex].FindByID(p.Repository.ID)}
	}

	record.Fields.Feature = features

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

func (p Issue) ToRecord(cache airtabledb.DB) airtabledb.Record {
	record := airtabledb.Record{}
	features := airtabledb.IssueRecord{}

	// base
	record.Fields.ID = p.ID
	record.Fields.CreatedAt = p.CreatedAt
	record.Fields.UpdatedAt = p.UpdatedAt
	record.Fields.Errors = strings.Join(p.Errors, ", ")

	// specific
	features.URL = p.URL
	features.CompletedAt = p.CompletedAt
	features.Title = p.Title
	features.State = p.State
	features.Body = p.Body
	features.IsPR = p.IsPR
	features.IsLocked = p.IsLocked
	features.Comments = p.Comments
	features.Upvotes = p.Upvotes
	features.Downvotes = p.Downvotes
	features.IsOrphan = p.IsOrphan
	features.IsHidden = p.IsHidden
	features.Weight = p.Weight
	features.IsEpic = p.IsEpic
	features.HasEpic = p.HasEpic

	// relationships
	features.Repository = []string{cache.Tables[airtabledb.RepositoryIndex].FindByID(p.Repository.ID)}
	if p.Milestone != nil {
		features.Milestone = []string{cache.Tables[airtabledb.MilestoneIndex].FindByID(p.Milestone.ID)}
	}
	features.Author = []string{cache.Tables[airtabledb.AccountIndex].FindByID(p.Author.ID)}
	features.Labels = []string{}
	for _, label := range p.Labels {
		features.Labels = append(features.Labels, cache.Tables[airtabledb.LabelIndex].FindByID(label.ID))
	}
	features.Assignees = []string{}
	for _, assignee := range p.Assignees {
		features.Assignees = append(features.Assignees, cache.Tables[airtabledb.AccountIndex].FindByID(assignee.ID))
	}

	record.Fields.Feature = features
	return record
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

func (p Label) ToRecord(cache airtabledb.DB) airtabledb.Record {
	record := airtabledb.Record{}
	features := airtabledb.LabelRecord{}

	// base
	record.Fields.ID = p.ID
	record.Fields.CreatedAt = p.CreatedAt
	record.Fields.UpdatedAt = p.UpdatedAt
	record.Fields.Errors = strings.Join(p.Errors, ", ")

	// specific
	features.URL = p.URL
	features.Name = p.Name
	features.Color = p.Color
	features.Description = p.Description

	// relationships
	// n/a

	record.Fields.Feature = features

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

func (p Account) ToRecord(cache airtabledb.DB) airtabledb.Record {
	record := airtabledb.Record{}
	features := airtabledb.AccountRecord{}

	// base
	record.Fields.ID = p.ID
	record.Fields.CreatedAt = p.CreatedAt
	record.Fields.UpdatedAt = p.UpdatedAt
	record.Fields.Errors = strings.Join(p.Errors, ", ")

	// specific
	features.URL = p.URL
	features.Login = p.Login
	features.FullName = p.FullName
	features.Type = p.Type
	features.Bio = p.Bio
	features.Location = p.Location
	features.Company = p.Company
	features.Blog = p.Blog
	features.Email = p.Email
	features.AvatarURL = p.AvatarURL

	// relationships
	features.Provider = []string{cache.Tables[airtabledb.ProviderIndex].FindByID(p.Provider.ID)}

	record.Fields.Feature = features

	return record
}

func (a Account) String() string {
	out, _ := json.Marshal(a)
	return string(out)
}

// FIXME: create a User struct to handle multiple accounts and aliases
