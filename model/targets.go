package model

import "moul.io/multipmuri"

func ParseTargets(args []string) ([]multipmuri.Entity, error) {
	targets := []multipmuri.Entity{}
	defaultContext := multipmuri.NewGitHubService("")
	for _, arg := range args {
		entity, err := defaultContext.RelDecodeString(arg)
		if err != nil {
			return nil, err
		}
		targets = append(targets, entity)
	}
	return targets, nil
}
