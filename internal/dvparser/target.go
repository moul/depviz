package dvparser

import "moul.io/multipmuri"

func ParseTargets(args []string) ([]multipmuri.Entity, error) {
	targets := []multipmuri.Entity{}
	// defaultContext := multipmuri.NewGitHubService("")
	defaultContext := multipmuri.NewTrelloService()
	for _, arg := range args {
		entity, err := defaultContext.RelDecodeString(arg)
		if err != nil {
			return nil, err
		}
		targets = append(targets, entity)
	}
	return targets, nil
}

func ParseTarget(arg string) (multipmuri.Entity, error) {
	// defaultContext := multipmuri.NewGitHubService("")
	defaultContext := multipmuri.NewTrelloService()
	return defaultContext.RelDecodeString(arg)
}
