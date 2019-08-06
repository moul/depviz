package model // import "moul.io/depviz/model"

import (
	"reflect"
	"strings"

	"github.com/lib/pq"
	"moul.io/depviz/airtabledb"
	"moul.io/depviz/airtablemodel"
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
				tableIndex, ok := airtablemodel.TableNameToIndex[strings.ToLower(srcFieldTypeName)]
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
