package main

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/brianloveswords/airtable"
)

type AirtableBase struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created-at"`
	UpdatedAt time.Time `json:"updated-at"`
	Errors    string    `json:"errors"`
}

type airtableState int

type AirtableRecords []interface{}

type AirtableEntry interface {
	ToRecord(cache AirtableDB) interface{}
}

const (
	airtableStateUnknown airtableState = iota
	airtableStateUnchanged
	airtableStateChanged
	airtableStateNew
)

var (
	airtableStateString = map[airtableState]string{
		airtableStateUnknown:   "unknown",
		airtableStateUnchanged: "unchanged",
		airtableStateChanged:   "changed",
		airtableStateNew:       "new",
	}
)

//
// provider
//

type ProviderRecord struct {
	State airtableState `json:"-"` // internal

	airtable.Record // provides ID, CreatedTime
	Fields          struct {
		// base
		AirtableBase

		// specific
		URL    string `json:"url"`
		Driver string `json:"driver"`

		// relationship
		// n/a
	} `json:"fields,omitempty"`
}

func (r ProviderRecord) String() string {
	out, _ := json.Marshal(r)
	return string(out)
}

func (p Provider) ToRecord(cache AirtableDB) *ProviderRecord {
	record := ProviderRecord{}

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

func (r *ProviderRecord) Equals(n *ProviderRecord) bool {
	return true &&
		// base
		r.Fields.ID == n.Fields.ID &&
		isSameAirtableDate(r.Fields.CreatedAt, n.Fields.CreatedAt) &&
		isSameAirtableDate(r.Fields.UpdatedAt, n.Fields.UpdatedAt) &&
		r.Fields.Errors == n.Fields.Errors &&

		// specific
		r.Fields.URL == n.Fields.URL &&
		r.Fields.Driver == n.Fields.Driver &&

		// relationships
		// n/a

		true
}

type ProviderRecords []ProviderRecord

func (records ProviderRecords) ByID(id string) string {
	for _, record := range records {
		if record.Fields.ID == id {
			return record.ID
		}
	}
	return ""
}

//
// label
//

type LabelRecord struct {
	State airtableState `json:"-"` // internal

	airtable.Record // provides ID, CreatedTime
	Fields          struct {
		// base
		AirtableBase

		// specific
		URL         string `json:"url"`
		Name        string `json:"name"`
		Color       string `json:"color"`
		Description string `json:"description"`

		// relationship
		// n/a
	} `json:"fields,omitempty"`
}

func (r LabelRecord) String() string {
	out, _ := json.Marshal(r)
	return string(out)
}

func (p Label) ToRecord(cache AirtableDB) *LabelRecord {
	record := LabelRecord{}

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

func (r *LabelRecord) Equals(n *LabelRecord) bool {
	return true &&
		// base
		r.Fields.ID == n.Fields.ID &&
		isSameAirtableDate(r.Fields.CreatedAt, n.Fields.CreatedAt) &&
		isSameAirtableDate(r.Fields.UpdatedAt, n.Fields.UpdatedAt) &&
		r.Fields.Errors == n.Fields.Errors &&

		// specific
		r.Fields.URL == n.Fields.URL &&
		r.Fields.Name == n.Fields.Name &&
		r.Fields.Color == n.Fields.Color &&
		r.Fields.Description == n.Fields.Description &&

		// relationships
		// n/a

		true
}

type LabelRecords []LabelRecord

func (records LabelRecords) ByID(id string) string {
	for _, record := range records {
		if record.Fields.ID == id {
			return record.ID
		}
	}
	return ""
}

//
// account
//

type AccountRecord struct {
	State airtableState `json:"-"` // internal

	airtable.Record // provides ID, CreatedTime
	Fields          struct {
		// base
		AirtableBase

		// specific
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
		Provider []string `json:"provider"`
	} `json:"fields,omitempty"`
}

func (r AccountRecord) String() string {
	out, _ := json.Marshal(r)
	return string(out)
}

func (p Account) ToRecord(cache AirtableDB) *AccountRecord {
	record := AccountRecord{}
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

func (r *AccountRecord) Equals(n *AccountRecord) bool {
	return true &&

		// base
		r.Fields.ID == n.Fields.ID &&
		isSameAirtableDate(r.Fields.CreatedAt, n.Fields.CreatedAt) &&
		isSameAirtableDate(r.Fields.UpdatedAt, n.Fields.UpdatedAt) &&
		r.Fields.Errors == n.Fields.Errors &&

		// specific
		r.Fields.URL == n.Fields.URL &&
		r.Fields.Login == n.Fields.Login &&
		r.Fields.FullName == n.Fields.FullName &&
		r.Fields.Type == n.Fields.Type &&
		r.Fields.Bio == n.Fields.Bio &&
		r.Fields.Location == n.Fields.Location &&
		r.Fields.Company == n.Fields.Company &&
		r.Fields.Blog == n.Fields.Blog &&
		r.Fields.Email == n.Fields.Email &&
		r.Fields.AvatarURL == n.Fields.AvatarURL &&

		// relationships
		isSameStringSlice(r.Fields.Provider, n.Fields.Provider) &&

		true
}

type AccountRecords []AccountRecord

func (records AccountRecords) ByID(id string) string {
	for _, record := range records {
		if record.Fields.ID == id {
			return record.ID
		}
	}
	return ""
}

//
// repository
//

type RepositoryRecord struct {
	State airtableState `json:"-"` // internal

	airtable.Record // provides ID, CreatedTime
	Fields          struct {
		// base
		AirtableBase

		// specific
		URL         string    `json:"url"`
		Title       string    `json:"title"`
		Description string    `json:"description"`
		Homepage    string    `json:"homepage"`
		PushedAt    time.Time `json:"pushed-at"`
		IsFork      bool      `json:"is-fork"`

		// relationships
		Provider []string `json:"provider"`
		Owner    []string `json:"owner"`
	} `json:"fields,omitempty"`
}

func (r RepositoryRecord) String() string {
	out, _ := json.Marshal(r)
	return string(out)
}

func (p Repository) ToRecord(cache AirtableDB) *RepositoryRecord {
	record := RepositoryRecord{}

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

func (r *RepositoryRecord) Equals(n *RepositoryRecord) bool {
	return true &&

		// base
		r.Fields.ID == n.Fields.ID &&
		isSameAirtableDate(r.Fields.CreatedAt, n.Fields.CreatedAt) &&
		isSameAirtableDate(r.Fields.UpdatedAt, n.Fields.UpdatedAt) &&
		r.Fields.Errors == n.Fields.Errors &&

		// specific
		r.Fields.URL == n.Fields.URL &&
		r.Fields.Title == n.Fields.Title &&
		r.Fields.Description == n.Fields.Description &&
		r.Fields.Homepage == n.Fields.Homepage &&
		isSameAirtableDate(r.Fields.PushedAt, n.Fields.PushedAt) &&
		r.Fields.IsFork == n.Fields.IsFork &&

		// relationships
		isSameStringSlice(r.Fields.Provider, n.Fields.Provider) &&
		isSameStringSlice(r.Fields.Owner, n.Fields.Owner) &&

		true
}

type RepositoryRecords []RepositoryRecord

func (records RepositoryRecords) ByID(id string) string {
	for _, record := range records {
		if record.Fields.ID == id {
			return record.ID
		}
	}
	return ""
}

//
// milestone
//

type MilestoneRecord struct {
	State airtableState `json:"-"` // internal

	airtable.Record // provides ID, CreatedTime
	Fields          struct {
		// base
		AirtableBase

		// specific
		URL         string    `json:"url"`
		Title       string    `json:"title"`
		Description string    `json:"description"`
		ClosedAt    time.Time `json:"closed-at"`
		DueOn       time.Time `json:"due-on"`

		// relationships
		Creator    []string `json:"creator"`
		Repository []string `json:"repository"`
	} `json:"fields,omitempty"`
}

func (r MilestoneRecord) String() string {
	out, _ := json.Marshal(r)
	return string(out)
}

func (p Milestone) ToRecord(cache AirtableDB) *MilestoneRecord {
	record := MilestoneRecord{}
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

func (r *MilestoneRecord) Equals(n *MilestoneRecord) bool {
	return true &&

		// base
		r.Fields.ID == n.Fields.ID &&
		isSameAirtableDate(r.Fields.CreatedAt, n.Fields.CreatedAt) &&
		isSameAirtableDate(r.Fields.UpdatedAt, n.Fields.UpdatedAt) &&
		r.Fields.Errors == n.Fields.Errors &&

		// specific
		r.Fields.URL == n.Fields.URL &&
		r.Fields.Title == n.Fields.Title &&
		r.Fields.Description == n.Fields.Description &&
		isSameAirtableDate(r.Fields.ClosedAt, n.Fields.ClosedAt) &&
		isSameAirtableDate(r.Fields.DueOn, n.Fields.DueOn) &&

		// relationships
		isSameStringSlice(r.Fields.Creator, n.Fields.Creator) &&
		isSameStringSlice(r.Fields.Repository, n.Fields.Repository) &&

		true
}

type MilestoneRecords []MilestoneRecord

func (records MilestoneRecords) ByID(id string) string {
	for _, record := range records {
		if record.Fields.ID == id {
			return record.ID
		}
	}
	return ""
}

//
// issue
//

type IssueRecord struct {
	State airtableState `json:"-"` // internal

	airtable.Record // provides ID, CreatedTime
	Fields          struct {
		// base
		AirtableBase

		// specific
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
		Repository []string `json:"repository"`
		Milestone  []string `json:"milestone"`
		Author     []string `json:"author"`
		Labels     []string `json:"labels"`
		Assignees  []string `json:"assignees"`
		//Parents      []string    `json:"-"`
		//Children     []string    `json:"-"`
		//Duplicates   []string    `json:"-"`
	} `json:"fields,omitempty"`
}

func (r IssueRecord) String() string {
	out, _ := json.Marshal(r)
	return string(out)
}

func (p Issue) ToRecord(cache AirtableDB) *IssueRecord {
	record := IssueRecord{}
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

func (r *IssueRecord) Equals(n *IssueRecord) bool {
	return true &&

		// base
		r.Fields.ID == n.Fields.ID &&
		isSameAirtableDate(r.Fields.CreatedAt, n.Fields.CreatedAt) &&
		isSameAirtableDate(r.Fields.UpdatedAt, n.Fields.UpdatedAt) &&
		r.Fields.Errors == n.Fields.Errors &&

		// specific
		r.Fields.URL == n.Fields.URL &&
		isSameAirtableDate(r.Fields.CompletedAt, n.Fields.CompletedAt) &&
		r.Fields.Title == n.Fields.Title &&
		r.Fields.State == n.Fields.State &&
		r.Fields.Body == n.Fields.Body &&
		r.Fields.IsPR == n.Fields.IsPR &&
		r.Fields.IsLocked == n.Fields.IsLocked &&
		r.Fields.Comments == n.Fields.Comments &&
		r.Fields.Upvotes == n.Fields.Upvotes &&
		r.Fields.Downvotes == n.Fields.Downvotes &&
		r.Fields.IsOrphan == n.Fields.IsOrphan &&
		r.Fields.IsHidden == n.Fields.IsHidden &&
		r.Fields.Weight == n.Fields.Weight &&
		r.Fields.IsEpic == n.Fields.IsEpic &&
		r.Fields.HasEpic == n.Fields.HasEpic &&

		// relationships
		isSameStringSlice(r.Fields.Repository, n.Fields.Repository) &&
		isSameStringSlice(r.Fields.Milestone, n.Fields.Milestone) &&
		isSameStringSlice(r.Fields.Author, n.Fields.Author) &&
		isSameStringSlice(r.Fields.Labels, n.Fields.Labels) &&
		isSameStringSlice(r.Fields.Assignees, n.Fields.Assignees) &&

		true
}

type IssueRecords []IssueRecord

func (records IssueRecords) ByID(id string) string {
	for _, record := range records {
		if record.Fields.ID == id {
			return record.ID
		}
	}
	return ""
}
