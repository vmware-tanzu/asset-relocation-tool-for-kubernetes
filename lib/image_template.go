package lib

import (
	"bytes"
	"fmt"
	"regexp"
	"text/template"

	dockerparser "github.com/novln/docker-parser"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/proto/hapi/chart"
)

type ImageTemplate struct {
	Raw           string
	Template      *template.Template
	OriginalImage *dockerparser.Reference
	NewImage      *dockerparser.Reference
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

func (t *ImageTemplate) Render(chart HelmChart) error {
	chartValues := &chartutil.Values{}
	err := yaml.Unmarshal([]byte(chart.GetValues().Raw), chartValues)
	if err != nil {
		return errors.Wrap(err, "failed to load chart values")
	}

	output := bytes.Buffer{}

	err = t.Template.Execute(&output, map[string]*chartutil.Values{
		"Values": chartValues,
	})
	if err != nil {
		return errors.Wrap(err, "failed to render image")
	}

	t.OriginalImage, err = dockerparser.Parse(output.String())
	if err != nil {
		return errors.Wrap(err, "failed to parse image reference")
	}

	return nil
}

var (
	TemplateRegex    = regexp.MustCompile(`{{\s*(.*?)\s*}}`)
	TagRegex         = regexp.MustCompile(`:{{\s*(.*?)\s*}}`)
	DigestRegex      = regexp.MustCompile(`@{{\s*(.*?)\s*}}`)
	TagOrDigestRegex = regexp.MustCompile(`[:|@]{{.*?}}`)
)

func (t *ImageTemplate) Apply(rules *RewriteRules) ([]*RewriteAction, error) {
	t.NewImage = rules.RewriteImage(t.OriginalImage)

	var rewrites []*RewriteAction

	// Tag or Digest
	tag := TagRegex.FindSubmatch([]byte(t.Raw))
	digest := DigestRegex.FindSubmatch([]byte(t.Raw))
	if len(tag) > 0 && rules.Tag != "" {
		rewrites = append(rewrites, &RewriteAction{
			Path:  string(tag[1]),
			Value: rules.Tag,
		})
	} else if len(digest) > 0 {
		rewrites = append(rewrites, &RewriteAction{
			Path:  string(digest[1]),
			Value: rules.Digest,
		})
	}

	// Either 1) registry + repo or 2) repo
	// Remove tag or digest from template
	templateWithoutTagDigest := TagOrDigestRegex.ReplaceAll([]byte(t.Raw), []byte(""))
	extraFragments := TemplateRegex.FindAllStringSubmatch(string(templateWithoutTagDigest), -1)

	switch len(extraFragments) {
	case 0:
		return nil, fmt.Errorf("the template \"%s\" does not include a repo or a registry fragment", t.Raw)
	case 1:
		// Set registry + repo
		rewrites = append(rewrites, &RewriteAction{
			Path:  extraFragments[0][1],
			Value: t.NewImage.Remote(),
		})
	case 2:
		// [registry]/[repo]:@[tag|digest]
		// Set registry
		rewrites = append(rewrites, &RewriteAction{
			Path:  extraFragments[0][1],
			Value: rules.Registry,
		})
		// Set repo
		rewrites = append(rewrites, &RewriteAction{
			Path:  extraFragments[1][1],
			Value: t.NewImage.Name(),
		})
	}

	// Retrieve parts
	return rewrites, nil
}
