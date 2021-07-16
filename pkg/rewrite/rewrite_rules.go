package rewrite

// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type Rules struct {
	Registry         string `json:"registry,omitempty"`
	RepositoryPrefix string `json:"repositoryPrefix,omitempty"`
	Repository       string `json:"repository,omitempty"`
	Tag              string `json:"tag,omitempty"`
	Digest           string `json:"digest,omitempty"`
}

func ParseRules(rulesFilePath string) (*Rules, error) {
	content, err := ioutil.ReadFile(rulesFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read rules file: %w", err)
	}

	rules := &Rules{}
	err = yaml.UnmarshalStrict(content, &rules)
	if err != nil {
		return nil, fmt.Errorf("the given rewrite rules are not in the correct format: %w", err)
	}
	return rules, nil
}

func (r *Rules) IsEmpty() bool {
	return *r == Rules{}
}
