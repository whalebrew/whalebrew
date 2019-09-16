package client

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	dockerConfig "github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/credentials"
	cliTypes "github.com/docker/cli/cli/config/types"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/term"
	"github.com/docker/docker/registry"
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

func (c *Client) ImageInspect(ctx context.Context, imageName string) (*types.ImageInspect, error) {
	imageInspect, _, err := c.ImageInspectWithRaw(ctx, imageName)
	if err == nil {
		return &imageInspect, nil
	}

	if client.IsErrNotFound(err) {
		fmt.Printf("Unable to find image '%s' locally\n", imageName)
		if err = c.PullImage(ctx, imageName); err != nil {
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

func (c *Client) PullImage(ctx context.Context, image string) error {
	pullOptions, err := buildPullOptions(image)
	if err != nil {
		return err
	}

	response, err := c.ImagePull(ctx, image, *pullOptions)

	if err != nil {
		return err
	}

	out := os.Stdout

	inFd, isTerm := term.GetFdInfo(out)

	if err := jsonmessage.DisplayJSONMessagesStream(response, out, inFd, isTerm, nil); err != nil {
		return err
	}

	return nil
}

func buildPullOptions(image string) (*types.ImagePullOptions, error) {
	store, err := getRepoAuth(image)
	if err !=  nil {
		return nil, err
	}

	hostname := registry.ConvertToHostname(image)

	privilegeFunc := func() (string, error) {
		authConfig, err := store.Get(hostname)
		if err != nil {
			return "", err
		}

		buf, err := json.Marshal(authConfig)
		if err != nil {
			return "", err
		}

		encodedAuth := base64.URLEncoding.EncodeToString(buf)

		return encodedAuth, nil
	}

	auth, _ := privilegeFunc()

	return &types.ImagePullOptions{
		RegistryAuth:  auth,
		PrivilegeFunc: privilegeFunc,
	}, nil
}

func getRepoAuth(image string) (credentials.Store, error) {
	distributionRef, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		return nil, err
	} else if reference.IsNameOnly(distributionRef) {
		distributionRef = reference.TagNameOnly(distributionRef)
		if tagged, ok := distributionRef.(reference.Tagged); ok {
			fmt.Printf("Using default tag: %s\n", tagged.Tag())
		}
	}

	repoInfo, err := registry.ParseRepositoryInfo(distributionRef)
	if err != nil {
		return nil, err
	}

	configFile := dockerConfig.LoadDefaultConfigFile(os.Stderr)

	configKey := registry.GetAuthConfigKey(repoInfo.Index)
	// First try the happy case
	if c, found := configFile.GetAuthConfigs()[configKey]; found || repoInfo.Index.Official {
		return &staticAuthStore{AuthConfig: c}, nil
	}

	hostname := registry.ConvertToHostname(image)

	return configFile.GetCredentialsStore(hostname), nil
}

var notImplemented = errors.New("not implemented")
type staticAuthStore struct {
	AuthConfig cliTypes.AuthConfig
}

func (s staticAuthStore) Erase(serverAddress string) error {
	return notImplemented
}

func (s staticAuthStore) Get(serverAddress string) (cliTypes.AuthConfig, error) {
	return s.AuthConfig, nil
}

func (s staticAuthStore) GetAll() (map[string]cliTypes.AuthConfig, error) {
	result := map[string]cliTypes.AuthConfig{}

	result[s.AuthConfig.ServerAddress] = s.AuthConfig
	return result, nil
}

func (s staticAuthStore) Store(authConfig cliTypes.AuthConfig) error {
	return notImplemented
}

