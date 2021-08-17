package internal

// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/divideandconquer/go-merge/merge"
	"github.com/google/go-containerregistry/pkg/name"
	yamlops2 "github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/v2/internal/yamlops"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
)

type OCIImageLocation struct {
	Registry         string
	RepositoryPrefix string
	Digest           string
}

type RewriteAction struct {
	Path  string `json:"path"`
	Value string `json:"value"`
}

func (a *RewriteAction) TopLevelKey() string {
	return strings.Split(a.Path, ".")[1]
}

func (a *RewriteAction) GetPathToMap() string {
	pathParts := strings.Split(a.Path, ".")
	return strings.Join(pathParts[:len(pathParts)-1], ".")
}

func (a *RewriteAction) GetSubPathToMap() string {
	pathParts := strings.Split(a.Path, ".")
	return "." + strings.Join(pathParts[2:len(pathParts)-1], ".")
}

func (a *RewriteAction) GetKey() string {
	pathParts := strings.Split(a.Path, ".")
	return pathParts[len(pathParts)-1]
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

func (a *RewriteAction) Apply(input *chart.Chart) (*chart.Chart, error) {
	dependencies := input.Dependencies()
	foundInDependency := false
	for dependencyIndex, dependency := range dependencies {
		if dependency.Name() == a.TopLevelKey() {
			foundInDependency = true
			valuesIndex, data := GetChartValues(dependency)
			value := map[string]string{
				a.GetKey(): a.Value,
			}
			newData, err := yamlops2.UpdateMap(data, a.GetSubPathToMap(), "", nil, value)
			if err != nil {
				return nil, fmt.Errorf("failed to apply modification to %s: %w", dependency.Name(), err)
			}

			dependencies[dependencyIndex].Raw[valuesIndex].Data = newData
		}
	}

	if foundInDependency {
		input.SetDependencies(dependencies...)
	} else {
		valuesIndex, data := GetChartValues(input)
		value := map[string]string{
			a.GetKey(): a.Value,
		}
		newData, err := yamlops2.UpdateMap(data, a.GetPathToMap(), "", nil, value)
		if err != nil {
			return nil, fmt.Errorf("failed to apply modification to %s: %w", input.Name(), err)
		}

		input.Raw[valuesIndex].Data = newData
	}

	return input, nil
}

func (a *RewriteAction) FindChartDestination(parentChart *chart.Chart) *chart.Chart {
	for _, subchart := range parentChart.Dependencies() {
		if subchart.Name() == a.TopLevelKey() {
			return subchart
		}
	}

	return parentChart
}

func GetChartValues(chart *chart.Chart) (int, []byte) {
	for fileIndex, file := range chart.Raw {
		if file.Name == chartutil.ValuesfileName {
			return fileIndex, file.Data
		}
	}
	return -1, nil
}

type ValuesMap map[string]interface{}

func BuildValuesMap(chart *chart.Chart, rewriteActions []*RewriteAction) map[string]interface{} {
	// Add values for chart dependencies
	for _, dependency := range chart.Dependencies() {
		chart.Values[dependency.Name()] = merge.Merge(dependency.Values, chart.Values[dependency.Name()])
	}

	// Apply rewrite actions
	values := chart.Values
	for _, action := range rewriteActions {
		actionMap := action.ToMap()
		result := merge.Merge(values, actionMap)
		var ok bool
		values, ok = result.(map[string]interface{})
		if !ok {
			return nil
		}
	}

	return values
}

func (t *ImageTemplate) Render(chart *chart.Chart, rewriteActions ...*RewriteAction) (name.Reference, error) {
	values := BuildValuesMap(chart, rewriteActions)

	output := bytes.Buffer{}
	err := t.Template.Execute(&output, values)
	if err != nil {
		return nil, fmt.Errorf("failed to render image: %w", err)
	}

	image, err := name.ParseReference(output.String())
	if err != nil {
		return nil, fmt.Errorf("failed to parse image reference: %w", err)
	}

	return image, nil
}

func (t *ImageTemplate) Apply(originalImage name.Reference, rules *OCIImageLocation) ([]*RewriteAction, error) {
	var rewrites []*RewriteAction
	var newRepository, newRepositorySuffix string

	// TODO(miguel): If any pattern can not be interpolated we should return an error
	if t.DigestTemplate != "" && rules.Digest != "" {
		rewrites = append(rewrites, &RewriteAction{
			Path:  t.DigestTemplate,
			Value: rules.Digest,
		})
	}

	// Either 1) newRegistry + repo or 2) repo
	newRegistry := rules.Registry
	currentRepo := originalImage.Context().RepositoryStr()

	if rules.RepositoryPrefix != "" {
		repoParts := strings.Split(currentRepo, "/")
		newRepository = rules.RepositoryPrefix + "/" + repoParts[len(repoParts)-1]
	}

	// Suffix to be added to the repository name if needed
	// Append the digest unless the tag and digest are explicitely encoded
	if t.TagTemplate == "" && t.DigestTemplate == "" && rules.Digest != "" {
		newRepositorySuffix = "@" + rules.Digest
	}

	if t.RegistryAndRepositoryTemplate != "" && newRegistry != "" && newRepository != "" {
		rewrites = append(rewrites, &RewriteAction{
			Path:  t.RegistryAndRepositoryTemplate,
			Value: fmt.Sprintf("%s/%s%s", newRegistry, newRepository, newRepositorySuffix),
		})
	}

	if t.RegistryTemplate != "" && newRegistry != "" {
		rewrites = append(rewrites, &RewriteAction{
			Path:  t.RegistryTemplate,
			Value: newRegistry,
		})
	}

	if t.RepositoryTemplate != "" && newRepository != "" {
		rewrites = append(rewrites, &RewriteAction{
			Path:  t.RepositoryTemplate,
			Value: fmt.Sprintf("%s%s", newRepository, newRepositorySuffix),
		})
	}

	return rewrites, nil
}
