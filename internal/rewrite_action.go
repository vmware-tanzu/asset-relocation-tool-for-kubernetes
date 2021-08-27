// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package internal

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/divideandconquer/go-merge/merge"
	"github.com/google/go-containerregistry/pkg/name"
	yamlops2 "github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/internal/yamlops"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
)

type OCIImageLocation struct {
	Registry         string
	RepositoryPrefix string
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

	for i := len(keys) - 1; i >= 0; i-- {
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
	values := chart.Values
	if values == nil {
		values = map[string]interface{}{}
	}

	// Add values for chart dependencies
	for _, dependency := range chart.Dependencies() {
		values[dependency.Name()] = merge.Merge(dependency.Values, values[dependency.Name()])
	}

	// Apply rewrite actions
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

func (t *ImageTemplate) Apply(originalImage name.Repository, imageDigest string, rules *OCIImageLocation) ([]*RewriteAction, error) {
	var rewrites []*RewriteAction

	registry := originalImage.Registry.Name()
	if rules.Registry != "" {
		registry = rules.Registry
	}

	// Repository path should contain the repositoryPrefix + imageName
	repository := originalImage.RepositoryStr()
	if rules.RepositoryPrefix != "" {
		repoParts := strings.Split(originalImage.RepositoryStr(), "/")
		imageName := repoParts[len(repoParts)-1]
		repository = fmt.Sprintf("%s/%s", rules.RepositoryPrefix, imageName)
	}

	// Append the image digest unless the tag or digest are explicitly encoded in the template
	// By doing so, we default to immutable references
	if t.TagTemplate == "" && t.DigestTemplate == "" {
		repository = fmt.Sprintf("%s@%s", repository, imageDigest)
	}

	registryChanged := originalImage.Registry.Name() != registry
	repoChanged := originalImage.RepositoryStr() != repository

	// The registry and the repository as encoded in a single template placeholder
	if t.RegistryAndRepositoryTemplate != "" && (registryChanged || repoChanged) {
		rewrites = append(rewrites, &RewriteAction{
			Path:  t.RegistryAndRepositoryTemplate,
			Value: fmt.Sprintf("%s/%s", registry, repository),
		})
	} else {
		// Explicitly override the registry
		if t.RegistryTemplate != "" && registryChanged {
			rewrites = append(rewrites, &RewriteAction{
				Path:  t.RegistryTemplate,
				Value: registry,
			})
		}

		// Explicitly override the repository
		if t.RepositoryTemplate != "" && repoChanged {
			rewrites = append(rewrites, &RewriteAction{
				Path:  t.RepositoryTemplate,
				Value: repository,
			})
		}
	}

	return rewrites, nil
}
