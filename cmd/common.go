package cmd

import (
	"context"
	"io"
	"io/ioutil"
	"regexp"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	dockerparser "github.com/novln/docker-parser"
	"github.com/pkg/errors"
)

type ImageManager struct {
	Context      context.Context
	DockerClient *client.Client
	Auth         map[string]string
}

var digestRegex = regexp.MustCompile(`{"status":"Digest: (sha256:[a-f0-9]*)"}`)

func GetDigestFromOutput(output io.ReadCloser) string {
	bytes, _ := ioutil.ReadAll(output)
	_ = output.Close()

	matches := digestRegex.FindAllStringSubmatch(string(bytes), -1)
	if len(matches) > 0 {
		return matches[0][1]
	}
	return ""
}

func (img *ImageManager) PullImage(image *dockerparser.Reference) (string, error) {
	output, err := img.DockerClient.ImagePull(img.Context, image.Remote(), types.ImagePullOptions{
		RegistryAuth: img.Auth[image.Registry()],
	})
	if err != nil {
		return "", errors.Wrapf(err, "failed to pull image %s", image.Remote())
	}

	return GetDigestFromOutput(output), nil
}

func (img *ImageManager) CheckImage(digest string, image *dockerparser.Reference) (bool, error) {
	output, err := img.DockerClient.ImagePull(img.Context, image.Remote(), types.ImagePullOptions{
		RegistryAuth: img.Auth[image.Registry()],
	})

	if err != nil {
		// Return true if failed to pull the image.
		// We see different errors if the image does not exist, or if the specific tag does not exist
		// It is simpler to attempt to push, which will catch legitimate issues (lack of authorization),
		// than it is to try and handle every error case here.
		return true, nil
	}

	remoteDigest := GetDigestFromOutput(output)
	if remoteDigest != digest {
		return false, errors.Errorf("remote image \"%s\" exists with a different digest: %s. Will not overwrite", image.Remote(), remoteDigest)
	} else {
		return false, nil
	}
}

func (img *ImageManager) PushImage(source, dest *dockerparser.Reference) error {
	if img.Auth[dest.Registry()] == "" {
		return errors.Errorf("not authorized to push to %s. Please retry with --registry-auth <url=username:password>", dest.Registry())
	}

	err := img.DockerClient.ImageTag(img.Context, source.Remote(), dest.Remote())
	if err != nil {
		return errors.Wrapf(err, "failed to tag image %s", dest.Remote())
	}

	_, err = img.DockerClient.ImagePush(img.Context, dest.Remote(), types.ImagePushOptions{
		RegistryAuth: img.Auth[dest.Registry()],
	})
	if err != nil {
		return errors.Wrapf(err, "failed to push image %s", dest.Remote())
	}

	return nil
}
