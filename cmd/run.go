package cmd

import (
	"github.com/spf13/cobra"
	"github.com/whalebrew/whalebrew/packages"
	"os"
	"os/exec"
)

func init() {
	RootCmd.AddCommand(runCommand)
}

var runCommand = &cobra.Command{
	Use:                "run [path to package] [args ...]",
	Short:              "Run a package",
	Long:               `Run a package`,
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		pkg, err := packages.LoadPackageFromPath(args[0])
		if err != nil {
			return err
		}
		dockerArgs := []string{
			"run",
			"--interactive",
			"--tty",
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
