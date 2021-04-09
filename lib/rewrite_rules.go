package lib

import (
	"fmt"
	"strings"

	dockerparser "github.com/novln/docker-parser"
)

type RewriteRules struct {
	Registry         string `json:"registry"`
	RepositoryPrefix string `json:"repositoryPrefix"`
	Repository       string `json:"repository"`
	Tag              string `json:"tag"`
	Digest           string `json:"digest"`
}

func (r *RewriteRules) RewriteImage(image *dockerparser.Reference) *dockerparser.Reference {
	registry := image.Registry()
	if r.Registry != "" {
		registry = r.Registry
	}

	repository := image.ShortName()
	if r.Repository != "" {
		repository = r.Repository
	}

	if r.RepositoryPrefix != "" {
		repository = r.RepositoryPrefix + "/" + repository
	}

	tag := strings.ReplaceAll(image.Name(), image.ShortName(), "")
	if r.Tag != "" {
		tag = ":" + r.Tag
	}
	if r.Digest != "" {
		tag = "@" + r.Digest
	}

	imageString := fmt.Sprintf("%s/%s%s", registry, repository, tag)
	newImage, _ := dockerparser.Parse(imageString)

	return newImage
}

type RewriteAction struct {
	Path  string `json:"path"`
	Value string `json:"value"`
}
