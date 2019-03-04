package client

import (
	"github.com/docker/docker/client"
)

// DefaultVersion is the Engine API version used by Whalebrew
const DefaultVersion string = "1.20"

// NewClient returns a Docker client configured for Whalebrew
func NewClient() (*client.Client, error) {
	return client.NewClientWithOpts(client.WithVersion(DefaultVersion), client.FromEnv)
}
