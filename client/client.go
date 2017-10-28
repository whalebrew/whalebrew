package client

import (
	"context"

	"github.com/docker/docker/client"
)

// NewClient returns a Docker client configured for Whalebrew
func NewClient() (*client.Client, error) {
	cli, err := client.NewEnvClient()
	if err != nil {
		return cli, err
	}
	cli.NegotiateAPIVersion(context.Background())
	return cli, nil
}
