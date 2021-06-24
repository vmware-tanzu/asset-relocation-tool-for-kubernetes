package test

import (
	"helm.sh/helm/v3/pkg/chart"
)

type ChartSeed struct {
	Name         string
	Values       map[string]interface{}
	Dependencies []*ChartSeed
}

func MakeChart(seed *ChartSeed) *chart.Chart {
	newChart := &chart.Chart{
		Values: seed.Values,
	}
	for _, dependency := range seed.Dependencies {
		newChart.AddDependency(&chart.Chart{
			Metadata: &chart.Metadata{
				Name: dependency.Name,
			},
			Values: dependency.Values,
		})
	}

	return newChart
}
