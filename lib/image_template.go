package lib

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"text/template"

	"github.com/divideandconquer/go-merge/merge"
	dockerparser "github.com/novln/docker-parser"
	"github.com/pkg/errors"
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

func NewFromString(input string) (*ImageTemplate, error) {
	temp, err := template.New(input).Parse(input)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse image template \"%s\"", input)
	}

	imageTemplate := &ImageTemplate{
		Raw:      input,
		Template: temp,
	}

	tagMatches := TagRegex.FindAllSubmatch([]byte(input), -1)
	if len(tagMatches) == 1 {
		imageTemplate.TagTemplate = string(tagMatches[0][1])
	} else if len(tagMatches) > 1 {
		return nil, errors.Errorf("failed to parse image template \"%s\": too many tag template matches", input)
	}

	digestMatches := DigestRegex.FindAllSubmatch([]byte(input), -1)
	if len(digestMatches) == 1 {
		imageTemplate.DigestTemplate = string(digestMatches[0][1])
	} else if len(digestMatches) > 1 {
		return nil, errors.Errorf("failed to parse image template \"%s\": too many digest template matches", input)
	}

	templateWithoutTagDigest := TagOrDigestRegex.ReplaceAll([]byte(input), []byte(""))
	extraFragments := TemplateRegex.FindAllStringSubmatch(string(templateWithoutTagDigest), -1)

	switch len(extraFragments) {
	case 0:
		return nil, errors.Errorf("failed to parse image template \"%s\": missing repo or a registry fragment", input)
	case 1:
		imageTemplate.RegistryAndRepositoryTemplate = extraFragments[0][1]
	case 2:
		imageTemplate.RegistryTemplate = extraFragments[0][1]
		imageTemplate.RepositoryTemplate = extraFragments[1][1]
	default:
		return nil, errors.Errorf("failed to parse image template \"%s\": more fragments than expected", input)
	}

	return imageTemplate, nil
}

func (t *ImageTemplate) Render(chart *chart.Chart, rewriteActions []*RewriteAction) (*dockerparser.Reference, error) {
	values := map[string]interface{}{
		"Values": chart.Values,
	}
	for _, action := range rewriteActions {
		actionMap := action.ToMap()
		result := merge.Merge(values, actionMap)
		values, _ = result.(map[string]interface{})
	}

	output := bytes.Buffer{}
	err := t.Template.Execute(&output, values)
	if err != nil {
		return nil, errors.Wrap(err, "failed to render image")
	}

	image, err := dockerparser.Parse(output.String())
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse image reference")
	}

	return image, nil
}

func (t *ImageTemplate) Apply(originalImage *dockerparser.Reference, rules *RewriteRules) ([]*RewriteAction, error) {
	tagged := false
	var rewrites []*RewriteAction

	// Tag or Digest
	if t.TagTemplate != "" {
		tagged = true
		if rules.Tag != "" && rules.Tag != originalImage.Tag() {
			rewrites = append(rewrites, &RewriteAction{
				Path:  t.TagTemplate,
				Value: rules.Tag,
			})
		}
	} else if t.DigestTemplate != "" {
		tagged = true
		if rules.Digest != "" && rules.Digest != originalImage.Tag() {
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
	registry := originalImage.Registry()
	if rules.Registry != "" {
		regModified = true
		registry = rules.Registry
	}

	tagString := strings.ReplaceAll(originalImage.Name(), originalImage.ShortName(), "")
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

	repository := originalImage.ShortName()
	if rules.Repository != "" {
		repoModified = true
		repository = rules.Repository
	} else if strings.HasPrefix(repository, "library") {
		repoModified = true
	}

	if rules.RepositoryPrefix != "" {
		repoModified = true
		repository = rules.RepositoryPrefix + "/" + repository
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
