package issues

import (
	"encoding/json"
	"reflect"
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

// toRecord attempts to automatically convert between an issues.Feature and an airtable Record.
// It's not particularly robust, but it works for structs following the format of Features and Records.
func toRecord(cache airtabledb.DB, src Feature, dst interface{}) {
	dV := reflect.ValueOf(dst).Elem().FieldByName("Fields")
	sV := reflect.ValueOf(src)
	copyFields(cache, sV, dV)
}

func copyFields(cache airtabledb.DB, src reflect.Value, dst reflect.Value) {
	dT := dst.Type()
	for i := 0; i < dst.NumField(); i++ {
		dFV := dst.Field(i)
		dSF := dT.Field(i)
		fieldName := dSF.Name
		// Recursively copy the embedded struct Base.
		if fieldName == "Base" {
			copyFields(cache, src, dFV)
			continue
		}
		sFV := src.FieldByName(fieldName)
		if fieldName == "Errors" {
			dFV.Set(reflect.ValueOf(strings.Join(sFV.Interface().(pq.StringArray), ", ")))
			continue
		}
		if dFV.Type().String() == "[]string" {
			if sFV.Pointer() != 0 {
				tableIndex := 0
				srcFieldTypeName := strings.Split(strings.Trim(sFV.Type().String(), "*[]"), ".")[1]
				tableIndex, ok := airtabledb.TableNameToIndex[strings.ToLower(srcFieldTypeName)]
				if !ok {
					panic("toRecord: could not find index for table name " + strings.ToLower(srcFieldTypeName))
				}
				if sFV.Kind() == reflect.Slice {
					for i := 0; i < sFV.Len(); i++ {
						idV := sFV.Index(i).Elem().FieldByName("ID")
						id := idV.String()
						dFV.Set(reflect.Append(dFV, reflect.ValueOf(cache.Tables[tableIndex].FindByID(id))))
					}
				} else {
					idV := sFV.Elem().FieldByName("ID")
					id := idV.String()
					dFV.Set(reflect.ValueOf([]string{cache.Tables[tableIndex].FindByID(id)}))
				}
			}
		} else {
			dFV.Set(sFV)
		}
	}
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

func (r Repository) ToRecord(cache airtabledb.DB) airtabledb.Record {
	record := airtabledb.RepositoryRecord{}
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
	URL    string `json:"url"`
	Driver string `json:"driver"` // github, gitlab, unknown
}

func (p Provider) ToRecord(cache airtabledb.DB) airtabledb.Record {
	record := airtabledb.ProviderRecord{}
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

func (m Milestone) ToRecord(cache airtabledb.DB) airtabledb.Record {
	record := airtabledb.MilestoneRecord{}
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

func (i Issue) ToRecord(cache airtabledb.DB) airtabledb.Record {
	record := airtabledb.IssueRecord{}
	toRecord(cache, i, &record)
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

func (l Label) ToRecord(cache airtabledb.DB) airtabledb.Record {
	record := airtabledb.LabelRecord{}
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

func (a Account) ToRecord(cache airtabledb.DB) airtabledb.Record {
	record := airtabledb.AccountRecord{}
	toRecord(cache, a, &record)
	return record
}

func (a Account) String() string {
	out, _ := json.Marshal(a)
	return string(out)
}

// FIXME: create a User struct to handle multiple accounts and aliases
