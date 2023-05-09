package multipmuri

import (
	"net/url"
	"strings"
)

func getHostname(input string) string {
	u, err := url.Parse(input)
	if err != nil {
		return ""
	}
	if u.Host != "" {
		return u.Host
	}
	if u.Path != "" {
		hostname := strings.Split(u.Path, "/")[0]
		if isHostname(hostname) {
			return hostname
		}
	}
	return ""
}

func isHostname(input string) bool {
	return strings.Contains(input, ".")
}

func isProviderScheme(scheme string) bool {
	switch scheme {
	case string(GitHubProvider),
		string(TrelloProvider),
		string(JiraProvider),
		string(GitLabProvider):
		return true
	}
	return false
}
