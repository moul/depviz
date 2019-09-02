package airtabledb // import "moul.io/depviz/airtabledb"

import (
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

func isSameAirtableDate(a, b time.Time) bool {
	return a.Truncate(time.Millisecond).UTC() == b.Truncate(time.Millisecond).UTC()
}

type Table struct {
	Elems interface{}
}

func (t Table) SetState(idx int, state State) {
	s := reflect.ValueOf(t.Elems).Elem().Index(idx).FieldByName("State")
	s.SetInt(int64(state))
}

func (t Table) GetState(idx int) State {
	return State(reflect.ValueOf(t.Elems).Elem().Index(idx).FieldByName("State").Int())
}

// CopyFields copies the 'Fields' struct from srcRecord into the Record at idx in the Tabel t.
// Will panic necessary fields do not exist.
func (t Table) CopyFields(idx int, srcRecord interface{}) {
	dstF := reflect.ValueOf(t.Elems).Elem().Index(idx).FieldByName("Fields")
	srcF := reflect.ValueOf(srcRecord).FieldByName("Fields")
	dstF.Set(srcF)
}

// GetFieldID returns the ID field of the Fields struct of the record at idx in the Table t.
// Will panic necessary fields do not exist.
func (t Table) GetFieldID(idx int) string {
	return reflect.ValueOf(t.Elems).Elem().Index(idx).FieldByName("Fields").FieldByName("ID").String()
}

// GetID returns the ID field of the record at idx in the Table t.
func (t Table) GetID(idx int) string {
	return reflect.ValueOf(t.Elems).Elem().Index(idx).FieldByName("ID").String()
}

// Len returns the number of records in the table.
func (t Table) Len() int {
	return reflect.ValueOf(t.Elems).Elem().Len()
}

// Append appends the given record to the table. Will panic if the given record is not of the right type.
func (t Table) Append(record interface{}) {
	a := reflect.Append(reflect.ValueOf(t.Elems).Elem(), reflect.ValueOf(record))
	reflect.ValueOf(t.Elems).Elem().Set(a)
}

// Fetch retrieves the airtable table records from at over the network and inserts the records into the table.
func (t Table) Fetch(at airtable.Table) error {
	return at.List(t.Elems, &airtable.Options{})
}

// FindByID searches the table for a record with Fields.ID equal to id.
// Returns the record's ID if a match is found. Otherwise, returns the empty string.
func (t Table) FindByID(id string) string {
	slice := reflect.ValueOf(t.Elems).Elem()
	for i := 0; i < slice.Len(); i++ {
		record := slice.Index(i)
		fieldID := record.FieldByName("Fields").FieldByName("ID").String()
		if fieldID == id {
			return record.FieldByName("ID").String()
		}
	}
	return ""
}

// GetPtr returns an interface containing a pointer to the record in the table at index idx.
func (t Table) GetPtr(idx int) interface{} {
	return reflect.ValueOf(t.Elems).Elem().Index(idx).Addr().Interface()
}

// Get returns an interface to the record in the table at idx.
func (t Table) Get(idx int) interface{} {
	return reflect.ValueOf(t.Elems).Elem().Index(idx).Interface()
}

// StringAt returns a JSON string of the record in the table at idx.
func (t Table) StringAt(idx int) string {
	out := reflect.ValueOf(t.Elems).Elem().Index(idx).MethodByName("String").Call(nil)
	return out[0].String()
}

type DB struct {
	Tables []Table
}

type Base struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created-at"`
	UpdatedAt time.Time `json:"updated-at"`
	Errors    string    `json:"errors"`
}

type State int

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
