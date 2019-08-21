package client

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// DefaultVersion is the Engine API version used by Whalebrew
const DefaultVersion string = "1.20"

type Client struct {
	*client.Client
}

// NewClient returns a Docker client configured for Whalebrew
func NewClient() (*Client, error) {
	dockerClient, err := client.NewClientWithOpts(client.WithVersion(DefaultVersion), client.FromEnv)
	if err != nil {
		return nil, err
	}

	return &Client{
		dockerClient,
	}, nil
}

func(c *Client) ImageInspect(ctx context.Context, imageName string) (*types.ImageInspect, error) {
	imageInspect, _, err := c.ImageInspectWithRaw(ctx, imageName)
	if err == nil {
		return &imageInspect, nil
	}

	if client.IsErrNotFound(err) {
		fmt.Printf("Unable to find image '%s' locally\n", imageName)
		if err = pullImage(imageName); err != nil {
			return nil, err
		}
		// retry
		imageInspect, _, err = c.ImageInspectWithRaw(ctx, imageName)
		if err != nil {
			return nil, err
		}

		return &imageInspect, nil
	} else {
		return nil, fmt.Errorf("failed to inspect docker image: %v", err)
	}
}

func pullImage(image string) error {
	c := exec.Command("docker", "pull", image)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
