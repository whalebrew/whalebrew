package cmd

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/whalebrew/whalebrew/packages"
	"github.com/whalebrew/whalebrew/run"
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

func getVolumes(pkg *packages.Package) ([]string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	volumes := []string{}
	for _, volume := range append(pkg.Volumes, fmt.Sprintf("%s:%s", cwd, pkg.WorkingDir)) {
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
			volumes = append(volumes, volume)
		}
	}
	return volumes, nil
}

func expandEnvVars(vars []string) []string {
	r := []string{}
	for _, v := range vars {
		r = append(r, os.ExpandEnv(v))
	}
	return r
}

// DockerCLIRun runs the package using docker CLI forwarding the command line arguments
func DockerCLIRun(args []string) error {
	docker, err := run.NewDocker()
	if err != nil {
		return err
	}
	return Run(docker, args)
}

// Run runs a package after extracting arguments
func Run(runner run.Runner, args []string) error {
	pkg, err := packages.LoadPackageFromPath(args[1])
	if err != nil {
		return err
	}
	args = args[2:]

	user, err := user.Current()
	if err != nil {
		return err
	}
	volumes, err := getVolumes(pkg)
	if err != nil {
		return err
	}
	return runner.Run(pkg, &run.Execution{
		WorkingDir:  pkg.WorkingDir,
		User:        user,
		IsTTYOpened: terminal.IsTerminal(int(os.Stdin.Fd())),
		Args:        args,
		Environment: expandEnvVars(pkg.Environment),
		Volumes:     volumes,
	})
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
