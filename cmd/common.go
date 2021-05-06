package cmd

import (
	"context"
	"fmt"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	dockerparser "github.com/novln/docker-parser"
	"github.com/pkg/errors"
)

type ImageManager struct {
	Output       io.Writer
	Context      context.Context
	DockerClient *client.Client
	Auth         map[string]string
}

func (img *ImageManager) PullImage(image *dockerparser.Reference) error {
	_, _ = fmt.Fprintf(img.Output, "Pulling %s... ", image.Remote())

	_, err := img.DockerClient.ImagePull(img.Context, image.Remote(), types.ImagePullOptions{
		RegistryAuth: img.Auth[image.Registry()],
	})
	if err != nil {
		_, _ = fmt.Fprintln(img.Output, "")
		return errors.Wrapf(err, "failed to pull image %s", image.Remote())
	}
	_, _ = fmt.Fprintln(img.Output, "Done")
	return nil
}

func (img *ImageManager) PushImage(source, dest *dockerparser.Reference) error {
	err := img.PullImage(source)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(img.Output, "Tagging %s... ", dest.Remote())
	err = img.DockerClient.ImageTag(img.Context, source.Remote(), dest.Remote())
	if err != nil {
		_, _ = fmt.Fprintln(img.Output, "")
		return errors.Wrapf(err, "failed to tag image %s", dest.Remote())
	}
	_, _ = fmt.Fprintln(img.Output, "Done")

	_, _ = fmt.Fprintf(img.Output, "Pushing %s... ", dest.Remote())
	_, err = img.DockerClient.ImagePush(img.Context, dest.Remote(), types.ImagePushOptions{
		RegistryAuth: img.Auth[dest.Registry()],
	})
	if err != nil {
		_, _ = fmt.Fprintln(img.Output, "")
		return errors.Wrapf(err, "failed to push image %s", dest.Remote())
	}
	_, _ = fmt.Fprintln(img.Output, "Done")

	return nil
}
