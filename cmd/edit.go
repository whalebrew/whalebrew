package cmd

import (
	"os"
	"os/exec"
	"runtime"
	"syscall"

	"github.com/bfirsh/whalebrew/packages"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	RootCmd.AddCommand(editCommand)
}

var editCommand = &cobra.Command{
	Use:                "edit PACKAGENAME",
	Short:              "Edit a package file",
	Long:               "Edit a package file using your default editor ($EDITOR).",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return cmd.Help()
		}

		pkgName := args[0]
		pm := packages.NewPackageManager(viper.GetString("install_path"))
		_, err := pm.Load(pkgName)
		if err != nil {
			return err
		}

		editor, ok := os.LookupEnv("EDITOR")
		if !ok {
			editor, ok = os.LookupEnv("GIT_EDITOR")
			if !ok {
				if runtime.GOOS == "windows" {
					editor = "notepad"
				} else {
					editor = "vi"
				}
			}
		}

		editorPath, err := exec.LookPath(editor)
		if err != nil {
			return err
		}

		editorArgs := []string{
			editorPath,
			pm.MakePackagePath(pkgName),
		}

		if runtime.GOOS == "windows" {
			editorCmd := exec.Command(editorPath, editorArgs[1:]...)
			editorCmd.Env = os.Environ()
			editorCmd.Stdin = os.Stdin
			editorCmd.Stdout = os.Stdout
			editorCmd.Stderr = os.Stderr

			exitStatus := 1
			if err := editorCmd.Run(); err != nil {
				if exitError, ok := err.(*exec.ExitError); ok {
					if ws, ok := exitError.Sys().(syscall.WaitStatus); ok {
						exitStatus = ws.ExitStatus()
					}
				}
			} else {
				ws := editorCmd.ProcessState.Sys().(syscall.WaitStatus)
				exitStatus = ws.ExitStatus()
			}
			os.Exit(exitStatus)
		}

		return syscall.Exec(editorPath, editorArgs, os.Environ())
	},
}
