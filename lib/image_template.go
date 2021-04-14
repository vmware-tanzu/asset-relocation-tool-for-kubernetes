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
	"gopkg.in/yaml.v2"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/proto/hapi/chart"
)

type ImageTemplate struct {
	Raw      string
	Template *template.Template
}

//go:generate counterfeiter . HelmChart
type HelmChart interface {
	GetValues() *chart.Config
}

func NewFromString(input string) (*ImageTemplate, error) {
	temp, err := template.New(input).Parse(input)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse image template \"%s\"", input)
	}

	return &ImageTemplate{
		Raw:      input,
		Template: temp,
	}, nil

}

func (t *ImageTemplate) Render(chart HelmChart, rewriteActions []*RewriteAction) (*dockerparser.Reference, error) {
	chartValues := make(map[string]interface{})
	err := yaml.Unmarshal([]byte(chart.GetValues().GetRaw()), chartValues)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load chart values")
	}

	values := chartutil.Values{
		"Values": chartValues,
	}
	for _, action := range rewriteActions {
		actionMap := action.ToMap()
		result := merge.Merge(values, actionMap)
		values, _ = result.(chartutil.Values)
	}

	//encoded, _ := yaml.Marshal(values)
	//_, _ = fmt.Fprintln(os.Stderr, string(encoded))

	output := bytes.Buffer{}
	err = t.Template.Execute(&output, values)
	if err != nil {
		return nil, errors.Wrap(err, "failed to render image")
	}

	image, err := dockerparser.Parse(output.String())
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse image reference")
	}

	return image, nil
}

var (
	TemplateRegex    = regexp.MustCompile(`{{\s*(.*?)\s*}}`)
	TagRegex         = regexp.MustCompile(`:{{\s*(.*?)\s*}}`)
	DigestRegex      = regexp.MustCompile(`@{{\s*(.*?)\s*}}`)
	TagOrDigestRegex = regexp.MustCompile(`[:|@]{{.*?}}`)
)

func (t *ImageTemplate) Apply(originalImage *dockerparser.Reference, rules *RewriteRules) ([]*RewriteAction, error) {
	tagged := false
	var rewrites []*RewriteAction

	// Tag or Digest
	tag := TagRegex.FindSubmatch([]byte(t.Raw))
	digest := DigestRegex.FindSubmatch([]byte(t.Raw))
	if len(tag) > 0 {
		tagged = true
		if rules.Tag != "" && rules.Tag != originalImage.Tag() {
			rewrites = append(rewrites, &RewriteAction{
				Path:  string(tag[1]),
				Value: rules.Tag,
			})
		}
	} else if len(digest) > 0 {
		tagged = true
		if rules.Digest != "" && rules.Digest != originalImage.Tag() {
			rewrites = append(rewrites, &RewriteAction{
				Path:  string(digest[1]),
				Value: rules.Digest,
			})
		}
	}

	// Either 1) registry + repo or 2) repo
	// Remove tag or digest from template
	templateWithoutTagDigest := TagOrDigestRegex.ReplaceAll([]byte(t.Raw), []byte(""))
	extraFragments := TemplateRegex.FindAllStringSubmatch(string(templateWithoutTagDigest), -1)

	if len(extraFragments) == 0 {
		return nil, errors.Errorf("the template \"%s\" does not include a repo or a registry fragment", t.Raw)
	}

	if len(extraFragments) > 2 {
		return nil, errors.Errorf("the template \"%s\" has more fragments than expected", t.Raw)
	}

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

	if len(extraFragments) == 1 {
		if regModified || repoModified {
			rewrites = append(rewrites, &RewriteAction{
				Path:  extraFragments[0][1],
				Value: fmt.Sprintf("%s/%s%s", registry, repository, tagString),
			})
		}
	} else {
		if regModified {
			rewrites = append(rewrites, &RewriteAction{
				Path:  extraFragments[0][1],
				Value: registry,
			})
		}

		if repoModified {
			rewrites = append(rewrites, &RewriteAction{
				Path:  extraFragments[1][1],
				Value: fmt.Sprintf("%s%s", repository, tagString),
			})
		}
	}

	return rewrites, nil
}
