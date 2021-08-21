// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package internal

import (
	"fmt"
	"regexp"
	"text/template"

	"gopkg.in/yaml.v2"
)

var (
	TemplateRegex    = regexp.MustCompile(`{{\s*(.*?)\s*}}`)
	TagRegex         = regexp.MustCompile(`:{{\s*(.*?)\s*}}`)
	DigestRegex      = regexp.MustCompile(`@{{\s*(.*?)\s*}}`)
	TagOrDigestRegex = regexp.MustCompile(`[:|@]{{.*?}}`)
)

type ImageTemplate struct {
	Raw      string
	Template *template.Template

	RegistryTemplate              string
	RepositoryTemplate            string
	RegistryAndRepositoryTemplate string
	TagTemplate                   string
	DigestTemplate                string
}

func (t *ImageTemplate) String() string {
	return t.Raw
}

func NewFromString(input string) (*ImageTemplate, error) {
	temp, err := template.New(input).Parse(input)
	if err != nil {
		return nil, fmt.Errorf("failed to parse image template \"%s\": %w", input, err)
	}

	imageTemplate := &ImageTemplate{
		Raw:      input,
		Template: temp,
	}

	tagMatches := TagRegex.FindAllStringSubmatch(input, -1)
	if len(tagMatches) == 1 {
		imageTemplate.TagTemplate = tagMatches[0][1]
	} else if len(tagMatches) > 1 {
		return nil, fmt.Errorf("failed to parse image template \"%s\": too many tag template matches", input)
	}

	digestMatches := DigestRegex.FindAllStringSubmatch(input, -1)
	if len(digestMatches) == 1 {
		imageTemplate.DigestTemplate = digestMatches[0][1]
	} else if len(digestMatches) > 1 {
		return nil, fmt.Errorf("failed to parse image template \"%s\": too many digest template matches", input)
	}

	templateWithoutTagDigest := TagOrDigestRegex.ReplaceAllString(input, "")
	extraFragments := TemplateRegex.FindAllStringSubmatch(templateWithoutTagDigest, -1)

	switch len(extraFragments) {
	case 0:
		return nil, fmt.Errorf("failed to parse image template \"%s\": missing repo or a registry fragment", input)
	case 1:
		imageTemplate.RegistryAndRepositoryTemplate = extraFragments[0][1]
	case 2:
		imageTemplate.RegistryTemplate = extraFragments[0][1]
		imageTemplate.RepositoryTemplate = extraFragments[1][1]
	default:
		return nil, fmt.Errorf("failed to parse image template \"%s\": more fragments than expected", input)
	}

	return imageTemplate, nil
}

func ParseImagePatterns(patterns string) ([]*ImageTemplate, error) {
	var templateStrings []string
	err := yaml.Unmarshal(([]byte)(patterns), &templateStrings)
	if err != nil {
		return nil, fmt.Errorf("image pattern file is not in the correct format: %w", err)
	}

	imagePatterns := []*ImageTemplate{}
	for _, line := range templateStrings {
		temp, err := NewFromString(line)
		if err != nil {
			return nil, err
		}
		imagePatterns = append(imagePatterns, temp)
	}

	return imagePatterns, nil
}
