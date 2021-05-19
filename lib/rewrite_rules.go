package lib

import (
	"strings"

	"gitlab.eng.vmware.com/marketplace-partner-eng/relok8s/v2/lib/yamlops"
)

type RewriteRules struct {
	Registry         string `json:"registry,omitempty"`
	RepositoryPrefix string `json:"repositoryPrefix,omitempty"`
	Repository       string `json:"repository,omitempty"`
	Tag              string `json:"tag,omitempty"`
	Digest           string `json:"digest,omitempty"`
}

type RewriteAction struct {
	Path  string `json:"path"`
	Value string `json:"value"`
}

func (a *RewriteAction) ToMap() map[string]interface{} {
	keys := strings.Split(strings.TrimPrefix(a.Path, "."), ".")
	var node ValuesMap
	var value interface{} = a.Value

	for i := len(keys) - 1; i >= 0; i -= 1 {
		key := keys[i]
		node = make(ValuesMap)
		node[key] = value
		value = node
	}

	return node
}

func (a *RewriteAction) Apply(input []byte) ([]byte, error) {
	pathParts := strings.Split(a.Path, ".")
	actionPath := strings.Join(pathParts[:len(pathParts)-1], ".")
	value := map[string]string{
		pathParts[len(pathParts)-1]: a.Value,
	}
	return yamlops.UpdateMap(input, actionPath, "", nil, value)
}
