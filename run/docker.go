package run

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"

	imagev1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// Docker implements the Runner interface
type Docker struct {
	Path       string
	Exec       func(argv0 string, argv []string, envv []string) (err error)
	RunCommand func(argv0 string, argv []string, envv []string, stdout io.Writer, stderr io.Writer) (err error)
}

var (
	_          Runner         = &Docker{}
	_          ImageInspecter = &Docker{}
	candidates                = []string{"docker", "podman"}
)

func RunComand(argv0 string, argv []string, envv []string, stdout io.Writer, stderr io.Writer) (err error) {
	c := exec.Command(argv0, argv...)
	c.Stdout = stdout
	c.Stderr = stderr
	c.Env = envv
	return c.Run()
}

// NewDockerLikeRunner creates a new default Docker runner
func NewDockerLikeRunner() (*Docker, error) {
	var err error
	var dockerPath string

	for _, candidate := range candidates {
		dockerPath, err = exec.LookPath(candidate)
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, fmt.Errorf("unable to find any of %v executables: %w", candidates, err)
	}
	return &Docker{
		Path:       dockerPath,
		Exec:       syscall.Exec,
		RunCommand: RunComand,
	}, nil
}

func (d *Docker) ImageInspect(imageName string) (*imagev1.Image, error) {
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	err := d.RunCommand(d.Path, []string{"image", "inspect", imageName}, os.Environ(), stdout, stderr)
	if err != nil {
		err = d.RunCommand(d.Path, []string{"image", "pull", imageName}, os.Environ(), os.Stdout, os.Stderr)
		if err != nil {
			return nil, fmt.Errorf("failed to download image %s: %w", imageName, err)
		}
		stdout.Reset()
		stderr.Reset()
		err = d.RunCommand(d.Path, []string{"image", "inspect", imageName}, os.Environ(), stdout, stderr)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to inspect image %s: %w", imageName, err)
	}
	_, err = io.Copy(os.Stderr, stderr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to dump image inspect error:", err)
	}

	images := []imagev1.Image{}
	err = json.NewDecoder(stdout).Decode(&images)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect image %s: %w", imageName, err)
	}
	if len(images) == 0 {
		return nil, fmt.Errorf("failed to inspect image %s: %s image inspect returned no image", imageName, d.Path)
	}
	if len(images) > 1 {
		return nil, fmt.Errorf("failed to inspect image %s: %s image inspect returned more than one image", imageName, d.Path)
	}

	return &images[0], nil
}

// Run runs a given package until completion
func (d *Docker) Run(e *Execution) error {
	if e == nil {
		return fmt.Errorf("no execution provided")
	}
	if e.Image == "" {
		return fmt.Errorf("no image to run")
	}
	dockerArgs := []string{
		d.Path,
		"run",
		"--interactive",
		"--rm",
		"--workdir", e.WorkingDir,
		"--init",
	}
	args := e.Args
	if e.Entrypoint != nil {
		if len(e.Entrypoint) > 0 {
			dockerArgs = append(dockerArgs, "--entrypoint", e.Entrypoint[0])
			if len(e.Entrypoint) > 1 {
				args = append(e.Entrypoint[1:], args...)
			}
		}
	}
	for _, portmap := range e.Ports {
		dockerArgs = append(dockerArgs, "-p")
		dockerArgs = append(dockerArgs, portmap)
	}

	for _, network := range e.Networks {
		dockerArgs = append(dockerArgs, "--net")
		dockerArgs = append(dockerArgs, network)
	}
	if e.IsTTYOpened {
		dockerArgs = append(dockerArgs, "--tty")
	}
	for _, envvar := range e.Environment {
		dockerArgs = append(dockerArgs, "-e")
		dockerArgs = append(dockerArgs, envvar)
	}
	for _, volume := range e.Volumes {
		dockerArgs = append(dockerArgs, "-v")
		dockerArgs = append(dockerArgs, volume)
	}
	if !e.KeepContainerUser {
		if e.User != nil {
			dockerArgs = append(dockerArgs, "-u")
			dockerArgs = append(dockerArgs, e.User.Uid+":"+e.User.Gid)
		}
	}
	dockerArgs = append(dockerArgs, e.Image)
	dockerArgs = append(dockerArgs, args...)
	if d.Exec == nil {
		return fmt.Errorf("no docker executable provided")
	}
	return d.Exec(d.Path, dockerArgs, os.Environ())
}
