package airtablemodel // import "moul.io/depviz/airtablemodel"

import (
	"encoding/json"
	"time"

	"github.com/brianloveswords/airtable"
	"moul.io/depviz/airtabledb"
)

//
// provider
//

type ProviderRecord struct {
	State airtabledb.State `json:"-"` // internal

	airtable.Record // provides ID, CreatedTime
	Fields          struct {
		// base
		airtabledb.Base

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

//
// label
//

type LabelRecord struct {
	State airtabledb.State `json:"-"` // internal

	airtable.Record // provides ID, CreatedTime
	Fields          struct {
		// base
		airtabledb.Base

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

//
// account
//

type AccountRecord struct {
	State airtabledb.State `json:"-"` // internal

	airtable.Record // provides ID, CreatedTime
	Fields          struct {
		// base
		airtabledb.Base

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

//
// repository
//

type RepositoryRecord struct {
	State airtabledb.State `json:"-"` // internal

	airtable.Record // provides ID, CreatedTime
	Fields          struct {
		// base
		airtabledb.Base

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

//
// milestone
//

type MilestoneRecord struct {
	State airtabledb.State `json:"-"` // internal

	airtable.Record // provides ID, CreatedTime
	Fields          struct {
		// base
		airtabledb.Base

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

//
// issue
//

type IssueRecord struct {
	State airtabledb.State `json:"-"` // internal

	airtable.Record // provides ID, CreatedTime
	Fields          struct {
		// base
		airtabledb.Base

		// specific
		URL          string    `json:"url"`
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
		// Weight  int       `json:"weight"`
		// IsEpic  bool `json:"is-epic"`
		// HasEpic bool `json:"has-epic"`

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
