package common

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type RewriteRules struct {
	Registry         string `json:"registry,omitempty"`
	RepositoryPrefix string `json:"repositoryPrefix,omitempty"`
	Repository       string `json:"repository,omitempty"`
	Tag              string `json:"tag,omitempty"`
	Digest           string `json:"digest,omitempty"`
}

func ParseRules(rulesFilePath string) (*RewriteRules, error) {
	content, err := ioutil.ReadFile(rulesFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read rules file: %w", err)
	}

	rewriteRules := &RewriteRules{}
	err = yaml.UnmarshalStrict(content, &rewriteRules)
	if err != nil {
		return nil, fmt.Errorf("the given rewrite rules are not in the correct format: %w", err)
	}
	return rewriteRules, nil
}

func (r *RewriteRules) IsEmpty() bool {
	return *r == RewriteRules{}
}