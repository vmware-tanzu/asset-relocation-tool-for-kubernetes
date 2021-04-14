package lib

import (
	"strings"
)

type RewriteRules struct {
	Registry         string `json:"registry"`
	RepositoryPrefix string `json:"repositoryPrefix"`
	Repository       string `json:"repository"`
	Tag              string `json:"tag"`
	Digest           string `json:"digest"`
}

type RewriteAction struct {
	Path  string `json:"path"`
	Value string `json:"value"`
}

func (a *RewriteAction) ToMap() map[string]interface{} {
	keys := strings.Split(strings.TrimPrefix(a.Path, "."), ".")
	var node map[string]interface{}
	var value interface{} = a.Value

	for i := len(keys) - 1; i >= 0; i -= 1 {
		key := keys[i]
		node = make(map[string]interface{})
		node[key] = value
		value = node
	}

	return node
}
