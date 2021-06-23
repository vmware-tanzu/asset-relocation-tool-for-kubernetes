package lib

import (
	"fmt"
	"strings"

	"gitlab.eng.vmware.com/marketplace-partner-eng/relok8s/v2/lib/yamlops"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
)

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
			newData, err := yamlops.UpdateMap(data, a.GetSubPathToMap(), "", nil, value)
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
		newData, err := yamlops.UpdateMap(data, a.GetPathToMap(), "", nil, value)
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
