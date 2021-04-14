CHART_FILES=$(shell find sample-chart -type f)

sample-chart-0.1.0.tgz: $(CHART_FILES)
	helm package sample-chart

.PHONY: build clean rebuild
build: sample-chart-0.1.0.tgz
rebuild: clean sample-chart-0.1.0.tgz
clean:
	rm -f sample-chart-0.1.0.tgz

default: build
