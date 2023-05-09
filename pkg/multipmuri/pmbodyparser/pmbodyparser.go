package pmbodyparser

import (
	"fmt"
	"regexp"
	"sort"

	"moul.io/depviz/v3/pkg/multipmuri"
)

type Kind string

const (
	Blocks      Kind = "blocks"
	DependsOn   Kind = "depends-on"
	Fixes       Kind = "fixes"
	Closes      Kind = "closes"
	Addresses   Kind = "addresses"
	RelatedWith Kind = "related-with"
	PartOf      Kind = "part-of"
	ParentOf    Kind = "parent-of"
)

type Relationship struct {
	Kind   Kind
	Target multipmuri.Entity
}

func (r Relationship) String() string {
	return fmt.Sprintf("%s %s", r.Kind, r.Target)
}

// FIXME: add isDependent / isDepending helpers

type Relationships []Relationship

func (r Relationships) Less(i, j int) bool {
	if r[i].Kind < r[j].Kind {
		return true
	}
	if r[j].Kind < r[i].Kind {
		return false
	}
	return r[i].Target.String() < r[j].Target.String()
}

func (r Relationships) Len() int {
	return len(r)
}

func (r Relationships) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func ParseString(body string) (Relationships, []error) {
	return RelParseString(multipmuri.NewUnknownEntity(), body)
}

var (
	fixesRegex, _       = regexp.Compile(`(?im)^\s*(fix|fixes)\s*[:= ]\s*([^\s,.]+).?\s*$`)
	blocksRegex, _      = regexp.Compile(`(?im)^\s*(block|blocks)\s*[:= ]\s*([^\s,.]+).?\s*$`)
	closesRegex, _      = regexp.Compile(`(?im)^\s*(close|closes)\s*[:= ]\s*([^\s,.]+).?\s*$`)
	parentOfRegex, _    = regexp.Compile(`(?im)^\s*(parent of|parent)\s*[:= ]\s*([^\s,.]+).?\s*$`)
	partOfRegex, _      = regexp.Compile(`(?im)^\s*(part of|part)\s*[:= ]\s*([^\s,.]+).?\s*$`)
	relatedWithRegex, _ = regexp.Compile(`(?im)^\s*(related|related with)\s*[:= ]\s*([^\s,.]+).?\s*$`)
	addressesRegex, _   = regexp.Compile(`(?im)^\s*(address|addresses)\s*[:= ]\s*([^\s,.]+).?\s*$`)
	dependsOnRegex, _   = regexp.Compile(`(?im)^\s*(depend|depends|depend on|depends on)\s*[:= ]\s*([^\s,.]+).?\s*$`)
)

func RelParseString(context multipmuri.Entity, body string) (Relationships, []error) {
	relationships := Relationships{}
	errs := []error{}

	for kind, regex := range map[Kind]*regexp.Regexp{
		Fixes:       fixesRegex,
		Blocks:      blocksRegex,
		Closes:      closesRegex,
		DependsOn:   dependsOnRegex,
		ParentOf:    parentOfRegex,
		PartOf:      partOfRegex,
		RelatedWith: relatedWithRegex,
		Addresses:   addressesRegex,
	} {
		for _, match := range regex.FindAllStringSubmatch(body, -1) {
			decoded, err := context.RelDecodeString(match[len(match)-1])
			if err != nil {
				errs = append(errs, err)
				continue
			}
			relationships = append(
				relationships,
				Relationship{Kind: kind, Target: decoded},
			)
		}
	}

	sort.Sort(relationships)
	return relationships, errs
}
