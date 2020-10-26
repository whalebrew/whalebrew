package dockerregistry

import (
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type RoundTripperFunc func(*http.Request) (*http.Response, error)

func (r RoundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return r(req)
}

func TestRegistryCatalogWithoutAuthentication(t *testing.T) {
	if err := exec.Command("docker", "pull", "alpine").Run(); err != nil {
		t.Skipf("Unable to pull alpine image: %v, this test needs a running docker daemon", err)
		return
	}
	c := exec.Command(
		"docker",
		"run",
		"-d",
		"-p", "5000:5000",
		"--name", "registry",
		"registry:2",
	)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	assert.NoError(t, c.Run())
	defer func() {
		c := exec.Command("docker", "kill", "registry")
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		c.Run()
		c = exec.Command("docker", "rm", "registry")
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		c.Run()
	}()
	c = exec.Command("docker", "build", "-t", "localhost:5000/some/image:latest", "-")
	c.Stdin = strings.NewReader("FROM scratch\nLABEL some=value\n")
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	assert.NoError(t, c.Run())
	maxRetry := 50
	for retry := 0; retry <= maxRetry; retry++ {
		if retry == maxRetry {
			t.Errorf("timeout trying to push image to test registry")
			t.FailNow()
		}
		c := exec.Command("docker", "push", "localhost:5000/some/image:latest")
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if c.Run() == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Run("When there is no authentication required", func(t *testing.T) {
		r := &Registry{
			Host:    "localhost:5000",
			UseHTTP: true,
		}
		cat, err := r.Catalog()
		assert.NoError(t, err)
		assert.Equal(t, len(cat.Repositories), 1)
	})
	t.Run("When authentication is required", func(t *testing.T) {
		transport := http.DefaultTransport
		defer func() {
			http.DefaultTransport = transport
		}()
		http.DefaultTransport = RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			// Fake an authorisation protocol defined here:
			// https://docs.docker.com/registry/spec/api/#api-version-check
			switch req.URL.Path {
			case "/v2/_catalog":
				if req.Header.Get("Authorization") == "Bearer my-access-token" {
					return transport.RoundTrip(req)
				}
				h := http.Header{}
				h.Set("www-authenticate", `Bearer realm="https://localhost:5000/auth", scope="hello-world"`)
				return &http.Response{
					StatusCode: http.StatusUnauthorized,
					Header:     h,
					Request:    req,
					Body:       ioutil.NopCloser(strings.NewReader("")),
				}, nil
			case "/auth":
				return &http.Response{
					StatusCode: http.StatusOK,
					Request:    req,
					Body:       ioutil.NopCloser(strings.NewReader(`{"token":"token","access_token":"my-access-token","expires_in":300}`)),
				}, nil
			default:
				return &http.Response{
					StatusCode: http.StatusBadRequest,
					Request:    req,
					Body:       ioutil.NopCloser(strings.NewReader("unexpected call to path " + req.URL.Path)),
				}, nil
			}
		})
		r := &Registry{
			Host:    "localhost:5000",
			UseHTTP: true,
		}
		cat, err := r.Catalog()
		assert.NoError(t, err)
		assert.Equal(t, len(cat.Repositories), 1)
	})
}
