package internal

import (
	"fmt"

	"gopkg.in/yaml.v2"
)

type RewriteRules struct {
	Registry         string `json:"registry,omitempty"`
	RepositoryPrefix string `json:"repositoryPrefix,omitempty"`
	Repository       string `json:"repository,omitempty"`
	Tag              string `json:"tag,omitempty"`
	Digest           string `json:"digest,omitempty"`
}

func ParseRules(rules string) (*RewriteRules, error) {
	rewriteRules := &RewriteRules{}
	err := yaml.UnmarshalStrict(([]byte)(rules), &rewriteRules)
	if err != nil {
		return nil, fmt.Errorf("the given rewrite rules are not in the correct format: %w", err)
	}
	return rewriteRules, nil
}
