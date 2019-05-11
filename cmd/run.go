package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/whalebrew/whalebrew/packages"
	"golang.org/x/crypto/ssh/terminal"
)

func shouldBind(hostPath string, pkg *packages.Package) (bool, error) {
	if pkg.MountMissingVolumes {
		return true, nil
	}
	// according to docker docs, binded volumes must be provided by absolute path
	// momn abs path are handled as docker volume names
	// https://docs.docker.com/engine/reference/commandline/run/#mount-volume--v---read-only
	if filepath.IsAbs(hostPath) {
		_, err := os.Stat(hostPath)
		if err != nil && os.IsNotExist(err) {
			if pkg.SkipMissingVolumes {
				return false, nil
			}
			return false, err
		}
	}
	return true, nil
}

func appendVolumes(dockerArgs []string, pkg *packages.Package) ([]string, error) {
	for _, volume := range pkg.Volumes {
		// special case expanding home directory
		if strings.HasPrefix(volume, "~/") {
			user, err := user.Current()
			if err != nil {
				return nil, err
			}
			volume = user.HomeDir + volume[1:]
		}
		volume = os.ExpandEnv(volume)
		b, err := shouldBind(strings.Split(volume, ":")[0], pkg)
		if err != nil {
			return nil, err
		}
		if b {
			dockerArgs = append(dockerArgs, "-v")
			dockerArgs = append(dockerArgs, volume)
		}
	}
	return dockerArgs, nil
}

// Run runs a package after extracting arguments
func Run(args []string) error {
	pkg, err := packages.LoadPackageFromPath(args[1])
	if err != nil {
		return err
	}
	dockerPath, err := exec.LookPath("docker")
	if err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	dockerArgs := []string{
		dockerPath,
		"run",
		"--interactive",
		"--rm",
		"--workdir", os.ExpandEnv(pkg.WorkingDir),
		"-v", fmt.Sprintf("%s:%s", cwd, os.ExpandEnv(pkg.WorkingDir)),
		"--init",
	}
	if terminal.IsTerminal(int(os.Stdin.Fd())) {
		dockerArgs = append(dockerArgs, "--tty")
	}
	dockerArgs, err = appendVolumes(dockerArgs, pkg)
	if err != nil {
		return err
	}
	for _, envvar := range pkg.Environment {
		dockerArgs = append(dockerArgs, "-e")
		dockerArgs = append(dockerArgs, os.ExpandEnv(envvar))
	}
	for _, portmap := range pkg.Ports {
		dockerArgs = append(dockerArgs, "-p")
		dockerArgs = append(dockerArgs, portmap)
	}
	for _, network := range pkg.Networks {
		dockerArgs = append(dockerArgs, "--net")
		dockerArgs = append(dockerArgs, network)
	}

	if !pkg.KeepContainerUser {
		user, err := user.Current()
		if err != nil {
			return err
		}
		dockerArgs = append(dockerArgs, "-u")
		dockerArgs = append(dockerArgs, user.Uid+":"+user.Gid)
	}

	dockerArgs = append(dockerArgs, pkg.Image)
	dockerArgs = append(dockerArgs, args[1:]...)

	return syscall.Exec(dockerPath, dockerArgs, os.Environ())
}

// IsShellbang returns whether the arguments should be interpreted as a shellbang run
func IsShellbang(args []string) bool {
	if len(args) < 2 {
		// a shellbang #!/usr/bin/env whalebrew
		// will always have at least <pathTo>/whalebrew <file>
		return false
	}
	// args are like <pathTo>/whalebrew <file>
	// When used as shellbang, the user ran <file> which leaded
	// to open it, read the shellbang line and run prefxing the
	// extended absolute <file> path with the shellbang command.
	// We are also sure that it cannot be a sub command as no sub-command starts with /
	// This disables the option to `whalebrew ./package.yaml`
	return strings.HasPrefix(args[1], "/")
}
