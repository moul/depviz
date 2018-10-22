package main

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"
)

func wrap(text string, lineWidth int) string {
	words := strings.Fields(strings.TrimSpace(text))
	if len(words) == 0 {
		return text
	}
	wrapped := words[0]
	spaceLeft := lineWidth - len(wrapped)
	for _, word := range words[1:] {
		if len(word)+1 > spaceLeft {
			wrapped += "\n" + word
			spaceLeft = lineWidth - len(word)
		} else {
			wrapped += " " + word
			spaceLeft -= 1 + len(word)
		}
	}

	return wrapped
}

func escape(input string) string {
	return fmt.Sprintf("%q", input)
}

func panicIfErr(err error) {
	if err != nil {
		panic(err)
	}
}

func uniqueStrings(input []string) []string {
	u := make([]string, 0, len(input))
	m := make(map[string]bool)

	for _, val := range input {
		if _, ok := m[val]; !ok {
			m[val] = true
			u = append(u, val)
		}
	}

	return u
}

func normalizeURL(input string) string {
	parts := strings.Split(input, "://")
	output := fmt.Sprintf("%s://%s", parts[0], strings.Replace(parts[1], "//", "/", -1))
	output = strings.TrimRight(output, "#")
	output = strings.TrimRight(output, "/")
	return output
}

var rxDNSName = regexp.MustCompile(`^([a-zA-Z0-9_]{1}[a-zA-Z0-9_-]{0,62}){1}(\.[a-zA-Z0-9_]{1}[a-zA-Z0-9_-]{0,62})*[\._]?$`)

func isDNSName(input string) bool {
	return rxDNSName.MatchString(input)
}

func isSameStringSlice(a, b []string) bool {
	if a == nil {
		a = []string{}
	}
	if b == nil {
		b = []string{}
	}
	sort.Strings(a)
	sort.Strings(b)
	return reflect.DeepEqual(a, b)
}

func isSameAirtableDate(a, b time.Time) bool {
	return a.Truncate(time.Millisecond).UTC() == b.Truncate(time.Millisecond).UTC()
}
