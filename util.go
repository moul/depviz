package main

import (
	"fmt"
	"os"
	"strings"
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

func getReposFromTargets(targets []string) []string {
	reposMap := map[string]bool{}

	for _, target := range targets {
		if _, err := os.Stat(target); err == nil {
			logger().Fatal("filesystem target are not yet supported")
		}
		repo := strings.Split(target, "/issues")[0]
		repo = strings.Split(target, "#")[0]
		reposMap[repo] = true
	}
	repos := []string{}
	for repo := range reposMap {
		repos = append(repos, repo)
	}
	return uniqueStrings(repos)
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
