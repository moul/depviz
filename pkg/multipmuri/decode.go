package multipmuri

import (
	"fmt"
	"net/url"
	"strings"
)

func DecodeString(input string) (Entity, error) {
	return NewUnknownEntity().RelDecodeString(input)
}

type unknownEntity struct{}

func NewUnknownEntity() Entity { return &unknownEntity{} }

func (unknownEntity) Kind() Kind           { return UnknownKind }
func (unknownEntity) Provider() Provider   { return UnknownProvider }
func (unknownEntity) String() string       { return "" }
func (unknownEntity) Equals(Entity) bool   { return false }
func (unknownEntity) Contains(Entity) bool { return false }
func (unknownEntity) RelDecodeString(input string) (Entity, error) {
	// FIXME: support more providers' cloning URLs
	if strings.HasPrefix(input, "git@github.com:") {
		input = strings.Replace(input, "git@github.com:", "https://github.com/", 1)
	}
	u, err := url.Parse(input)
	if err != nil {
		return nil, err
	}

	if isProviderScheme(u.Scheme) {
		input = input[len(u.Scheme)+3:]
		switch u.Scheme {
		case string(GitHubProvider):
			return gitHubRelDecodeString(getHostname(input), "", "", input, true)
		case string(GitLabProvider):
			return gitLabRelDecodeString(getHostname(input), "", "", input, true)
			//case string(JiraProvider):
			//case string(TrelloProvider):
		}
	}

	if u.Scheme == "" && u.Host == "" && u.Path != "" { // github.com/x/x
		u.Host = strings.Split(u.Path, "/")[0]
		// u.Path = u.Path[len(u.Host)+1:]
	}

	switch u.Scheme {
	case "", "https", "http":
		switch u.Host {
		case "github.com", "api.github.com":
			return gitHubRelDecodeString("", "", "", input, true)
		case "gitlab.com":
			return gitLabRelDecodeString("", "", "", input, true)
		case "trello.com":
			return trelloRelDecodeString(input, true)
			// case "jira.com", "atlassian.com":
		}
	}

	return nil, fmt.Errorf("ambiguous uri %q", input)
}
func (unknownEntity) LocalID() string { return "" }
