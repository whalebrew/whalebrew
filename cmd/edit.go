package cmd

import (
	"os"
	"os/exec"
	"path"
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
	DisableFlagParsing: true,
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
				editor = "vi"
			}
		}

		editorPath, err := exec.LookPath(editor)
		if err != nil {
			return err
		}

		editorArgs := []string{
			editorPath,
			path.Join(pm.InstallPath, pkgName),
		}

		return syscall.Exec(editorPath, editorArgs, os.Environ())
	},
}
