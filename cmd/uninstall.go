package cmd

import (
	"fmt"

	"github.com/Songmu/prompter"
	"github.com/bfirsh/whalebrew/packages"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	RootCmd.AddCommand(uninstallCommand)
}

var uninstallCommand = &cobra.Command{
	Use:   "uninstall PACKAGENAME",
	Short: "Uninstall a package",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return cmd.Help()
		}
		if len(args) > 1 {
			return fmt.Errorf("Only one image can be uninstalled at a time")
		}

		pm := packages.NewPackageManager(viper.GetString("install_path"))
		packageName := args[0]

		path := pm.MakePackagePath(packageName)

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
