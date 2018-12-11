package airtabledb

import (
	"encoding/json"
	"reflect"
	"sort"
	"time"

	"github.com/brianloveswords/airtable"
)

type FeatureFields interface {
	String() string
}

type Record struct {
	State State `json:"-"` // internal

	airtable.Record // provides ID, CreatedTime

	Fields struct {
		ID        string    `json:"id"`
		CreatedAt time.Time `json:"created-at"`
		UpdatedAt time.Time `json:"updated-at"`
		Errors    string    `json:"errors"`

		Feature FeatureFields
	} `json:"fields,omitempty"`
}

func (r Record) Equals(other Record) bool {
	if r.Fields.ID != other.Fields.ID {
		return false
	}
	if !isSameAirtableDate(r.Fields.CreatedAt, other.Fields.CreatedAt) {
		return false
	}
	if !isSameAirtableDate(r.Fields.UpdatedAt, other.Fields.UpdatedAt) {
		return false
	}
	if r.Fields.Errors != other.Fields.Errors {
		return false
	}
	rV := reflect.ValueOf(r.Fields.Feature)
	oV := reflect.ValueOf(other.Fields.Feature)
	if rV.Type() != oV.Type() {
		return false
	}
	if rV.NumField() != oV.NumField() {
		return false
	}
	for i := 0; i < rV.NumField(); i++ {
		rF := rV.Field(i)
		oF := oV.Field(i)
		if rF.Type() != oF.Type() {
			return false
		}
		if rF.Type().String() == "time.Time" {
			if !isSameAirtableDate(rF.Interface().(time.Time), oF.Interface().(time.Time)) {
				return false
			}
		}
		if rF.Type().String() == "[]string" {
			a, b := rF.Interface().([]string), oF.Interface().([]string)
			if a == nil {
				a = []string{}
			}
			if b == nil {
				b = []string{}
			}
			sort.Strings(a)
			sort.Strings(b)
			if !reflect.DeepEqual(a, b) {
				return false
			}
		}
		if !reflect.DeepEqual(rF.Interface(), oF.Interface()) {
			return false
		}
	}
	return true
}

func (r Record) String() string {
	out, _ := json.Marshal(r)
	return string(out)
}

type Records []Record

func (r Records) FindByID(id string) string {
	for _, record := range r {
		if record.Fields.ID == id {
			return record.ID
		}
	}
	return ""
}

type base struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created-at"`
	UpdatedAt time.Time `json:"updated-at"`
	Errors    string    `json:"errors"`
}

type State int

type DB struct {
	Tables []Records
}

func NewDB() DB {
	return DB{
		Tables: make([]Records, NumTables),
	}
}

const (
	IssueIndex = iota
	RepositoryIndex
	AccountIndex
	LabelIndex
	MilestoneIndex
	ProviderIndex
	NumTables
)

const (
	StateUnknown State = iota
	StateUnchanged
	StateChanged
	StateNew
)

var (
	StateString = map[State]string{
		StateUnknown:   "unknown",
		StateUnchanged: "unchanged",
		StateChanged:   "changed",
		StateNew:       "new",
	}
)

//
// provider
//

type ProviderRecord struct {
	// specific
	URL    string `json:"url"`
	Driver string `json:"driver"`
}

func (r ProviderRecord) String() string {
	out, _ := json.Marshal(r)
	return string(out)
}

//
// label
//
type LabelRecord struct {
	// specific
	URL         string `json:"url"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`

	// relationship
	// n/a
}

func (r LabelRecord) String() string {
	out, _ := json.Marshal(r)
	return string(out)
}

//
// account
//

type AccountRecord struct {
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
}

func (r AccountRecord) String() string {
	out, _ := json.Marshal(r)
	return string(out)
}

//
// repository
//

type RepositoryRecord struct {
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
}

func (r RepositoryRecord) String() string {
	out, _ := json.Marshal(r)
	return string(out)
}

//
// milestone
//

type MilestoneRecord struct {
	URL         string    `json:"url"`
	Title       string    `json:"title"`	
	Description string    `json:"description"`
	ClosedAt    time.Time `json:"closed-at"`
	DueOn       time.Time `json:"due-on"`

	// relationships
	Creator    []string `json:"creator"`
	Repository []string `json:"repository"`
}

func (r MilestoneRecord) String() string {
	out, _ := json.Marshal(r)
	return string(out)
}

//
// issue
//

type IssueRecord struct {
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
}

func (r IssueRecord) String() string {
	out, _ := json.Marshal(r)
	return string(out)
}

func isSameAirtableDate(a, b time.Time) bool {
	return a.Truncate(time.Millisecond).UTC() == b.Truncate(time.Millisecond).UTC()
}
