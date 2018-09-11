package main

import (
	"fmt"
	"net/url"
	"strings"
)

type Repo string

func NewRepo(path string) Repo {
	parts := strings.Split(path, "/")
	if len(parts) < 3 {
		return Repo(fmt.Sprintf("https://github.com/%s", path))
	}
	if !strings.Contains(path, "://") {
		return Repo(fmt.Sprintf("https://%s", path))
	}
	return Repo(path)
}

// FIXME: make something more reliable
func (r Repo) Provider() Provider {
	if strings.Contains(string(r), "github.com") {
		return GitHubProvider
	}
	return GitLabProvider
}

func (r Repo) Namespace() string {
	u, err := url.Parse(string(r))
	if err != nil {
		return ""
	}
	parts := strings.Split(u.Path, "/")[1:]
	return strings.Join(parts[:len(parts)-1], "/")
}

func (r Repo) Project() string {
	parts := strings.Split(string(r), "/")
	return parts[len(parts)-1]
}

func (r Repo) Canonical() string {
	// FIXME: use something smarter (the shortest unique response)
	return string(r)
}

func (r Repo) SiteURL() string {
	switch r.Provider() {
	case GitHubProvider:
		return "https://github.com"
	case GitLabProvider:
		u, err := url.Parse(string(r))
		if err != nil {
			return ""
		}
		return fmt.Sprintf("%s://%s", u.Scheme, u.Host)
	}
	panic("should not happen")
}

func (r Repo) RepoPath() string {
	u, err := url.Parse(string(r))
	if err != nil {
		panic(err)
	}
	return u.Path[1:]
}
