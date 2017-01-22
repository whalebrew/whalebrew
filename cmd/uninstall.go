package cmd

import (
	"fmt"
	"github.com/Songmu/prompter"
	"github.com/spf13/cobra"
	"github.com/whalebrew/whalebrew/packages"
	"path"
)

func init() {
	RootCmd.AddCommand(uninstallCommand)
}

var uninstallCommand = &cobra.Command{
	Use:   "uninstall [package name]",
	Short: "Uninstall a package",
	Long:  `Uninstall a package`,
	RunE: func(cmd *cobra.Command, args []string) error {
		pm := packages.NewPackageManager("/usr/local/bin")
		packageName := args[0]

		path := path.Join(pm.InstallPath, packageName)

		if !prompter.YN(fmt.Sprintf("This will permanently delete '%s'. Are you sure?", path), false) {
			return nil
		}

		err := pm.Uninstall(packageName)
		if err != nil {
			return err
		}

		fmt.Printf("🚽  Uninstalled %s\n", path)
		return nil
	},
}
