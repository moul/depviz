package airtabledb

import (
	"encoding/json"
	"reflect"
	"sort"
	"time"

	"github.com/brianloveswords/airtable"
)

type Record interface {
	String() string
}

func (t Table) RecordsEqual(idx int, b Record) bool {
	sf, ok := reflect.TypeOf(t.Get(idx)).FieldByName("Fields")
	if !ok {
		panic("No struct field Fields in Record")
	}
	aTF := sf.Type
	aVF := reflect.ValueOf(t.Get(idx)).FieldByName("Fields")
	bVF := reflect.ValueOf(b).FieldByName("Fields")

	if aVF.NumField() != bVF.NumField() {
		return false
	}
	for i := 0; i < aVF.NumField(); i++ {
		aiSF := aTF.Field(i)
		aiF := aVF.Field(i)
		biF := bVF.FieldByName(aiSF.Name)
		if aiF.Type() != biF.Type() {
			return false
		}
		if aiF.Type().String() == "time.Time" {
			if !isSameAirtableDate(aiF.Interface().(time.Time), biF.Interface().(time.Time)) {
				return false
			}
		} else if aiF.Type().String() == "[]string" {
			aS, bS := aiF.Interface().([]string), biF.Interface().([]string)
			if aS == nil {
				aS = []string{}
			}
			if bS == nil {
				bS = []string{}
			}
			sort.Strings(aS)
			sort.Strings(bS)
			if !reflect.DeepEqual(aS, bS) {
				return false
			}
			continue
		} else {
			if !reflect.DeepEqual(aiF.Interface(), biF.Interface()) {
				return false
			}
		}
	}
	return true
}

type Table struct {
	elems interface{}
}

func (t Table) SetState(idx int, state State) {
	s := reflect.ValueOf(t.elems).Elem().Index(idx).FieldByName("State")
	s.SetInt(int64(state))
}

func (t Table) GetState(idx int) State {
	return State(reflect.ValueOf(t.elems).Elem().Index(idx).FieldByName("State").Int())
}

func (t Table) CopyFields(idx int, src interface{}) {
	dstF := reflect.ValueOf(t.elems).Elem().Index(idx).FieldByName("Fields")
	srcF := reflect.ValueOf(src).FieldByName("Fields")
	dstF.Set(srcF)
}

func (t Table) GetFieldID(idx int) string {
	return reflect.ValueOf(t.elems).Elem().Index(idx).FieldByName("Fields").FieldByName("ID").String()
}

func (t Table) GetID(idx int) string {
	return reflect.ValueOf(t.elems).Elem().Index(idx).FieldByName("ID").String()
}

func (t Table) Len() int {
	return reflect.ValueOf(t.elems).Elem().Len()
}

func (t Table) Append(r interface{}) {
	a := reflect.Append(reflect.ValueOf(t.elems).Elem(), reflect.ValueOf(r))
	reflect.ValueOf(t.elems).Elem().Set(a)
}

func (t Table) Fetch(at airtable.Table) error {
	return at.List(t.elems, &airtable.Options{})
}

func (t Table) FindByID(id string) string {
	slice := reflect.ValueOf(t.elems).Elem()
	for i := 0; i < slice.Len(); i++ {
		record := slice.Index(i)
		fieldID := record.FieldByName("Fields").FieldByName("ID").String()
		if fieldID == id {
			return record.FieldByName("ID").String()
		}
	}
	return ""
}

func (t Table) GetPtr(idx int) interface{} {
	return reflect.ValueOf(t.elems).Elem().Index(idx).Addr().Interface()
}

func (t Table) Get(idx int) interface{} {
	return reflect.ValueOf(t.elems).Elem().Index(idx).Interface()
}

func (t Table) StringAt(idx int) string {
	out := reflect.ValueOf(t.elems).Elem().Index(idx).MethodByName("String").Call(nil)
	return out[0].String()
}

type DB struct {
	Tables []Table
}

func NewDB() DB {
	db := DB{
		Tables: make([]Table, NumTables),
	}
	db.Tables[IssueIndex].elems = &[]IssueRecord{}
	db.Tables[RepositoryIndex].elems = &[]RepositoryRecord{}
	db.Tables[AccountIndex].elems = &[]AccountRecord{}
	db.Tables[LabelIndex].elems = &[]LabelRecord{}
	db.Tables[MilestoneIndex].elems = &[]MilestoneRecord{}
	db.Tables[ProviderIndex].elems = &[]ProviderRecord{}
	if len(db.Tables) != NumTables {
		panic("missing an airtabledb Table")
	}
	return db
}

type Base struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created-at"`
	UpdatedAt time.Time `json:"updated-at"`
	Errors    string    `json:"errors"`
}

type State int

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
	State State `json:"-"` // internal

	airtable.Record // provides ID, CreatedTime
	Fields          struct {
		// base
		Base

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
	State State `json:"-"` // internal

	airtable.Record // provides ID, CreatedTime
	Fields          struct {
		// base
		Base

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
	State State `json:"-"` // internal

	airtable.Record // provides ID, CreatedTime
	Fields          struct {
		// base
		Base

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
	State State `json:"-"` // internal

	airtable.Record // provides ID, CreatedTime
	Fields          struct {
		// base
		Base

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
	State State `json:"-"` // internal

	airtable.Record // provides ID, CreatedTime
	Fields          struct {
		// base
		Base

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
	State State `json:"-"` // internal

	airtable.Record // provides ID, CreatedTime
	Fields          struct {
		// base
		Base

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

func isSameAirtableDate(a, b time.Time) bool {
	return a.Truncate(time.Millisecond).UTC() == b.Truncate(time.Millisecond).UTC()
}
