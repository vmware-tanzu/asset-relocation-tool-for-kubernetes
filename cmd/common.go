package cmd

import (
	"context"
	"fmt"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
)

type ImageManager struct {
	Output       io.Writer
	Context      context.Context
	DockerClient *client.Client
	// TODO: Add authentication pieces for one or two registries
}

func (img *ImageManager) PullImage(image string) error {
	_, _ = fmt.Fprintf(img.Output, "Pulling %s... ", image)
	_, err := img.DockerClient.ImagePull(img.Context, image, types.ImagePullOptions{})
	if err != nil {
		_, _ = fmt.Fprintln(img.Output, "")
		return errors.Wrapf(err, "failed to pull image %s", image)
	}
	_, _ = fmt.Fprintln(img.Output, "Done")
	return nil
}
