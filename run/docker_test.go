package run_test

import (
	"errors"
	"os"
	"os/exec"
	"os/user"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/whalebrew/whalebrew/run"
)

func TestDockerImageInspect(t *testing.T) {
	for _, candidate := range run.GetCandidates() {
		t.Run("With docker like command "+candidate, func(t *testing.T) {
			path, err := exec.LookPath(candidate)
			if err != nil {
				t.Skipf("unable to find command %s", candidate)
			}
			c := exec.Command(path, "image", "build", "-t", "my-image", "-")
			c.Stdin = strings.NewReader("FROM scratch\nENTRYPOINT [\"hello-world\"]\nLABEL foo=bar")
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			require.NoError(t, c.Run())
			d := run.Docker{
				Path:       path,
				RunCommand: run.RunComand,
			}
			inspect, err := d.ImageInspect("my-image")
			require.NoError(t, err)
			assert.Equal(t, []string{"hello-world"}, inspect.Config.Entrypoint)
			require.NotNil(t, inspect.Config.Labels)

			assert.Equal(t, "bar", inspect.Config.Labels["foo"])
		})
	}
}

func TestDockerRun(t *testing.T) {
	d := run.Docker{
		Path: "docker",
		Exec: nil,
	}
	assert.Error(t, d.Run(nil))
	assert.Error(t, d.Run(nil))
	assert.Error(t, d.Run(&run.Execution{
		Image: "alpine",
	}))
	assert.Error(t, d.Run(&run.Execution{
		Image: "alpine",
	}))
	d.Exec = func(argv0 string, argv []string, envv []string) (err error) { return errors.New("test error") }
	assert.Error(t, d.Run(&run.Execution{
		Image: "alpine",
	}))
	d.Exec = func(argv0 string, argv []string, envv []string) (err error) { return nil }
	assert.NoError(t, d.Run(&run.Execution{
		Image: "alpine",
	}))

	d.Exec = func(argv0 string, argv []string, envv []string) (err error) {
		assert.Equal(t, "docker", argv0)
		assert.Equal(
			t,
			[]string{
				"docker", "run", "--interactive", "--rm",
				"--workdir", "/workdir", "--init",
				"--entrypoint", "sh",
				"-p", "8080:8080", "-p", "1024:1024",
				"--net", "default",
				"--tty",
				"-e", "HELLO=world", "-e", "world",
				"-v", "/tmp:/tmp", "-v", "/var/run/docker.sock:/var/run/docker.sock",
				"-u", "2048:4086",
				"alpine",
				"-c",
				"-h", "hello", "world",
			},
			argv,
		)
		assert.Equal(t, os.Environ(), envv)
		return nil
	}
	assert.NoError(t, d.Run(&run.Execution{
		Image:       "alpine",
		Ports:       []string{"8080:8080", "1024:1024"},
		Networks:    []string{"default"},
		Entrypoint:  []string{"sh", "-c"},
		IsTTYOpened: true,
		WorkingDir:  "/workdir",
		Environment: []string{"HELLO=world", "world"},
		Volumes:     []string{"/tmp:/tmp", "/var/run/docker.sock:/var/run/docker.sock"},
		User: &user.User{
			Uid: "2048",
			Gid: "4086",
		},
		Args: []string{"-h", "hello", "world"},
	}))

	d.Exec = func(argv0 string, argv []string, envv []string) (err error) {
		assert.Equal(t, "docker", argv0)
		assert.Equal(
			t,
			[]string{
				"docker", "run", "--interactive", "--rm",
				"--workdir", "/workdir", "--init",
				"alpine",
			},
			argv,
		)
		assert.Equal(t, os.Environ(), envv)
		return nil
	}
	assert.NoError(t, d.Run(&run.Execution{
		Image:             "alpine",
		KeepContainerUser: true,
		WorkingDir:        "/workdir",
		User: &user.User{
			Uid: "2048",
			Gid: "4086",
		},
	}))
}
