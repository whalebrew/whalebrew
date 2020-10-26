package dockerregistry

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/containerd/containerd/remotes/docker"
	dockercliconfig "github.com/docker/cli/cli/config"
)

var dockerHubIndex = "index.docker.io"

type Registry struct {
	Host    string
	UseHTTP bool
}

func (r *Registry) Do(req *http.Request) (*http.Response, error) {
	if r == nil {
		// default to standard registry
		r = &Registry{}
	}
	authUrl := req.Clone(context.Background())
	authUrl.Body = nil
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		auth, err := dockercliconfig.LoadDefaultConfigFile(os.Stderr).GetAuthConfig(r.HostName())
		if err != nil {
			return nil, err
		}
		authorizer := docker.NewDockerAuthorizer(
			docker.WithAuthClient(http.DefaultClient),
			docker.WithAuthCreds(func(_ string) (string, string, error) {
				if auth.IdentityToken != "" {
					return "", auth.IdentityToken, nil
				}
				return auth.Username, auth.Password, nil
			}),
		)
		authorizer.AddResponses(context.Background(), []*http.Response{resp})
		authorizer.Authorize(context.Background(), req)
	}
	return http.DefaultClient.Do(req)
}

func (r *Registry) HostName() string {
	if r == nil || r.Host == "" {
		return dockerHubIndex
	}
	return r.Host
}

func (r *Registry) Scheme() string {
	if r == nil || !r.UseHTTP {
		return "https"
	}
	return "http"
}

func (r *Registry) NewRequest(method, path string, body io.Reader) (*http.Request, error) {
	u := url.URL{
		Scheme: r.Scheme(),
		Path:   path,
		Host:   r.HostName(),
	}
	return http.NewRequest(method, u.String(), body)
}

func (r *Registry) Get(path string, out interface{}) error {
	req, err := r.NewRequest("GET", path, nil)
	if err != nil {
		return err
	}
	resp, err := r.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Unexpected status %d, expecting %d", resp.StatusCode, http.StatusOK)
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

func (r *Registry) ImageName(path string) string {
	return fmt.Sprintf("%s/%s", r.Host, path)
}
