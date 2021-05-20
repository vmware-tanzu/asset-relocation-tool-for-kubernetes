package cmd

import (
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/pkg/errors"
)

func PullImage(imageReference name.Reference) (v1.Image, string, error) {
	image, err := remote.Image(imageReference, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		return nil, "", errors.Wrapf(err, "failed to pull image %s", imageReference.Name())
	}

	digest, err := image.Digest()
	if err != nil {
		return nil, "", errors.Wrapf(err, "failed to get image digest for %s", imageReference.Name())
	}

	return image, digest.String(), nil
}

func CheckImage(digest string, imageReference name.Reference) (bool, error) {
	_, remoteDigest, err := PullImage(imageReference)
	if err != nil {
		// Return true if failed to pull the image.
		// We see different errors if the image does not exist, or if the specific tag does not exist
		// It is simpler to attempt to push, which will catch legitimate issues (lack of authorization),
		// than it is to try and handle every error case here.
		return true, nil
	}

	if remoteDigest != digest {
		return false, errors.Errorf("remote image \"%s\" exists with a different digest: %s. Will not overwrite", imageReference.Name(), remoteDigest)
	} else {
		return false, nil
	}
}

func PushImage(image v1.Image, dest name.Reference) error {
	err := remote.Write(dest, image, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		return errors.Wrapf(err, "failed to push image %s", dest.Name())
	}

	return nil
}
