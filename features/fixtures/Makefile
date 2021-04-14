CHART_FILES=$(shell find sample-chart -type f)

sample-chart-0.1.0.tgz: $(CHART_FILES)
	helm package sample-chart

dependent-chart/charts/sample-chart-0.1.0.tgz:
	cd dependent-chart && helm dependency update

.PHONY: build clean rebuild
build: sample-chart-0.1.0.tgz dependent-chart/charts/sample-chart-0.1.0.tgz
rebuild: clean build
clean:
	rm -f sample-chart-0.1.0.tgz dependent-chart/charts/sample-chart-0.1.0.tgz

default: build
