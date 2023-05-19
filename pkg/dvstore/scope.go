package dvstore

import (
	"github.com/cayleygraph/cayley/graph/path"
	"github.com/cayleygraph/quad"
)

func generateCombinationsWithRepetition[T interface{}](n int, elements []T, currentCombination []T, combinations *[][]T) {
	if n == 0 {
		*combinations = append(*combinations, append([]T(nil), currentCombination...))
	} else {
		for _, element := range elements {
			currentCombination = append(currentCombination, element)
			generateCombinationsWithRepetition(n-1, elements, currentCombination, combinations)
			currentCombination = currentCombination[:len(currentCombination)-1]
		}
	}
}

func fold[T interface{}, Y interface{}](initial Y, arr []T, f func(Y, T) Y) Y {
	for _, v := range arr {
		initial = f(initial, v)
	}
	return initial
}

func scopeIssue(p *path.Path, scopeSize int, possibilities []quad.IRI) *path.Path {
	if scopeSize == 0 {
		return p
	}
	var perms [][]quad.IRI
	var currentPerms []quad.IRI
	generateCombinationsWithRepetition(scopeSize, possibilities, currentPerms, &perms)

	for _, perm := range perms {
		p = p.Or(fold(p, perm, func(_path *path.Path, _link quad.IRI) *path.Path { return _path.Both(_link) }))
	}
	return p
}
