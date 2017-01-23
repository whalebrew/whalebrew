package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/whalebrew/whalebrew/packages"
	"os"
	"os/exec"
)

func init() {
	RootCmd.AddCommand(runCommand)
}

var runCommand = &cobra.Command{
	Use:                "run PACKAGEPATH [ARGS ...]",
	Short:              "Run a package",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		pkg, err := packages.LoadPackageFromPath(args[0])
		if err != nil {
			return err
		}
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		dockerArgs := []string{
			"run",
			"--interactive",
			"--tty",
			"--workdir", "/workdir",
			"-v", fmt.Sprintf("%s:/workdir", cwd),
			pkg.Image,
		}
		dockerArgs = append(dockerArgs, args[1:]...)
		c := exec.Command("docker", dockerArgs...)
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	},
}
