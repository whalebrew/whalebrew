package run_test

import (
	"errors"
	"os"
	"os/user"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/whalebrew/whalebrew/run"

	"github.com/whalebrew/whalebrew/packages"
)

func TestDockerRun(t *testing.T) {
	d := run.Docker{
		Path: "docker",
		Exec: nil,
	}
	assert.Error(t, d.Run(nil, nil))
	assert.Error(t, d.Run(&packages.Package{}, nil))
	assert.Error(t, d.Run(&packages.Package{
		Image: "alpine",
	}, nil))
	assert.Error(t, d.Run(&packages.Package{
		Image: "alpine",
	}, &run.Execution{}))
	d.Exec = func(argv0 string, argv []string, envv []string) (err error) { return errors.New("test error") }
	assert.Error(t, d.Run(&packages.Package{
		Image: "alpine",
	}, &run.Execution{}))
	d.Exec = func(argv0 string, argv []string, envv []string) (err error) { return nil }
	assert.NoError(t, d.Run(&packages.Package{
		Image: "alpine",
	}, &run.Execution{}))

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
	assert.NoError(t, d.Run(&packages.Package{
		Image:      "alpine",
		Ports:      []string{"8080:8080", "1024:1024"},
		Networks:   []string{"default"},
		Entrypoint: []string{"sh", "-c"},
	}, &run.Execution{
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
	assert.NoError(t, d.Run(&packages.Package{
		Image:             "alpine",
		KeepContainerUser: true,
	}, &run.Execution{
		WorkingDir: "/workdir",
		User: &user.User{
			Uid: "2048",
			Gid: "4086",
		},
	}))

	// Test customgid functionality.
	d.Exec = func(argv0 string, argv []string, envv []string) (err error) {
		assert.Equal(t, "docker", argv0)
		assert.Equal(
			t,
			[]string{
				"docker", "run", "--interactive", "--rm",
				"--workdir", "/workdir", "--init",
				"-u", "2048:1234",
				"alpine",
				"-h", "hello", "world",
			},
			argv,
		)
		assert.Equal(t, os.Environ(), envv)
		return nil
	}
	assert.NoError(t, d.Run(&packages.Package{
		Image:     "alpine",
		CustomGid: "1234",
	}, &run.Execution{
		WorkingDir: "/workdir",
		User: &user.User{
			Uid: "2048",
			Gid: "4086",
		},
		Args: []string{"-h", "hello", "world"},
	}))
}
