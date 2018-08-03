package cmd

import (
	"fmt"
	"regexp"
	"os"
	"os/exec"
	"os/user"
	"strings"
	"syscall"

	"github.com/bfirsh/whalebrew/packages"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"
)

func init() {
	RootCmd.AddCommand(runCommand)
}

var runCommand = &cobra.Command{
	Use:                "run PACKAGEPATH [ARGS ...]",
	Short:              "Run a package",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return cmd.Help()
		}

		pkgPath := args[0]
		cmdArgs := args[1:]
		entrypoint := ""
		entrypointRegEx := regexp.MustCompile("--entrypoint=(.+)")
		entrypointMatches := entrypointRegEx.FindStringSubmatch(pkgPath)
		if entrypointMatches != nil {
			entrypoint = entrypointMatches[1]
			pkgPath = args[1]
			cmdArgs = args[2:]
		}

		pkg, err := packages.LoadPackageFromPath(pkgPath)
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
		}
		if terminal.IsTerminal(int(os.Stdin.Fd())) {
			dockerArgs = append(dockerArgs, "--tty")
		}
		for _, volume := range pkg.Volumes {
			// special case expanding home directory
			if strings.HasPrefix(volume, "~/") {
				user, err := user.Current()
				if err != nil {
					return err
				}
				volume = user.HomeDir + volume[1:]
			}
			dockerArgs = append(dockerArgs, "-v")
			dockerArgs = append(dockerArgs, os.ExpandEnv(volume))
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

		user, err := user.Current()
		if err != nil {
			return err
		}
		dockerArgs = append(dockerArgs, "-u")
		dockerArgs = append(dockerArgs, user.Uid+":"+user.Gid)

		if entrypoint != "" {
			dockerArgs = append(dockerArgs, "--entrypoint")
			dockerArgs = append(dockerArgs, entrypoint)
		}
		dockerArgs = append(dockerArgs, pkg.Image)
		dockerArgs = append(dockerArgs, cmdArgs...)

		return syscall.Exec(dockerPath, dockerArgs, os.Environ())
	},
}
