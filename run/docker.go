package run

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/whalebrew/whalebrew/packages"
)

// Docker implements the Runner interface
type Docker struct {
	Path string
	Exec func(argv0 string, argv []string, envv []string) (err error)
}

// NewDocker creates a new default Docker runner
func NewDocker() (*Docker, error) {
	dockerPath, err := exec.LookPath("docker")
	if err != nil {
		return nil, err
	}
	return &Docker{
		Path: dockerPath,
		Exec: syscall.Exec,
	}, nil
}

// Run runs a given package until completion
func (d *Docker) Run(p *packages.Package, e *Execution) error {
	if p == nil {
		return fmt.Errorf("No package provided")
	}
	if p.Image == "" {
		return fmt.Errorf("Provided package does not contain any image")
	}
	if e == nil {
		return fmt.Errorf("No execution provided")
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
	if p.Entrypoint != nil {
		if len(p.Entrypoint) > 0 {
			dockerArgs = append(dockerArgs, "--entrypoint", p.Entrypoint[0])
			if len(p.Entrypoint) > 1 {
				args = append(p.Entrypoint[1:], args...)
			}
		}
	}
	for _, portmap := range p.Ports {
		dockerArgs = append(dockerArgs, "-p")
		dockerArgs = append(dockerArgs, portmap)
	}

	for _, network := range p.Networks {
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

	if !p.KeepContainerUser {
		if e.User != nil {
			dockerArgs = append(dockerArgs, "-u")
			if p.CustomGid != "" {
				dockerArgs = append(dockerArgs, e.User.Uid+":"+p.CustomGid)
			} else {
				// does this have test..
				dockerArgs = append(dockerArgs, e.User.Uid+":"+e.User.Gid)
			}

		}
	}
	dockerArgs = append(dockerArgs, p.Image)
	dockerArgs = append(dockerArgs, args...)
	if d.Exec == nil {
		return fmt.Errorf("No docker executable provided")
	}
	return d.Exec(d.Path, dockerArgs, os.Environ())
}
