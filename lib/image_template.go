package lib

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"text/template"

	"github.com/divideandconquer/go-merge/merge"
	"github.com/google/go-containerregistry/pkg/name"
	"helm.sh/helm/v3/pkg/chart"
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

func (t *ImageTemplate) Apply(originalImage name.Reference, rules *RewriteRules) ([]*RewriteAction, error) {
	tagged := false
	var rewrites []*RewriteAction

	// Tag or Digest
	if t.TagTemplate != "" {
		tagged = true
		if rules.Tag != "" && rules.Tag != originalImage.Identifier() {
			rewrites = append(rewrites, &RewriteAction{
				Path:  t.TagTemplate,
				Value: rules.Tag,
			})
		}
	} else if t.DigestTemplate != "" {
		tagged = true
		if rules.Digest != "" && rules.Digest != originalImage.Identifier() {
			rewrites = append(rewrites, &RewriteAction{
				Path:  t.DigestTemplate,
				Value: rules.Digest,
			})
		}
	}

	// Either 1) registry + repo or 2) repo
	// Remove tag or digest from template
	regModified := false
	repoModified := false
	registry := originalImage.Context().Registry.Name()
	if rules.Registry != "" {
		regModified = true
		registry = rules.Registry
	}

	tagString := strings.ReplaceAll(originalImage.Name(), originalImage.Context().Name(), "")
	if tagged {
		tagString = ""
	} else {
		if rules.Tag != "" {
			repoModified = true
			tagString = ":" + rules.Tag
		}
		if rules.Digest != "" {
			repoModified = true
			tagString = "@" + rules.Digest
		}
	}

	repository := originalImage.Context().RepositoryStr()
	if rules.Repository != "" {
		repoModified = true
		repository = rules.Repository
	} else if strings.HasPrefix(repository, "library") {
		repoModified = true
	}

	if rules.RepositoryPrefix != "" {
		repoModified = true
		repoParts := strings.Split(repository, "/")
		repository = rules.RepositoryPrefix + "/" + repoParts[len(repoParts)-1]
	}

	if t.RegistryAndRepositoryTemplate != "" {
		if regModified || repoModified {
			rewrites = append(rewrites, &RewriteAction{
				Path:  t.RegistryAndRepositoryTemplate,
				Value: fmt.Sprintf("%s/%s%s", registry, repository, tagString),
			})
		}
	} else {
		if regModified {
			rewrites = append(rewrites, &RewriteAction{
				Path:  t.RegistryTemplate,
				Value: registry,
			})
		}

		if repoModified {
			rewrites = append(rewrites, &RewriteAction{
				Path:  t.RepositoryTemplate,
				Value: fmt.Sprintf("%s%s", repository, tagString),
			})
		}
	}

	return rewrites, nil
}
