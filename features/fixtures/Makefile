CHART_FILES=$(shell find samplechart -type f)

samplechart-0.1.0.tgz: $(CHART_FILES)
	helm package samplechart

dependentchart/charts/samplechart-0.1.0.tgz:
	cd dependentchart && helm dependency update

.PHONY: build clean rebuild
build: samplechart-0.1.0.tgz dependentchart/charts/samplechart-0.1.0.tgz
rebuild: clean build
clean:
	rm -f samplechart-0.1.0.tgz dependentchart/charts/samplechart-0.1.0.tgz

default: build
