package client

import (
	"github.com/docker/docker/client"
)

// DefaultVersion is the Engine API version used by Whalebrew
const DefaultVersion string = "1.20"

// NewClient returns a Docker client configured for Whalebrew
func NewClient() (*client.Client, error) {
	cli, err := client.NewClientWithOpts(client.WithVersion("1.39"))
	if err != nil {
		return cli, err
	}

	err = client.FromEnv(cli)
	if err != nil {
		return cli, err
	}

	return cli, nil
}
