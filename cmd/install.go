package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/whalebrew/whalebrew/packages"
	"path"
	"strings"
)

func init() {
	RootCmd.AddCommand(installCommand)
}

var installCommand = &cobra.Command{
	Use:   "install [image name]",
	Short: "Install a package",
	Long:  `install a Docker image as a Whalebrew package in your install path.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		pm := packages.NewPackageManager("/usr/local/bin")
		imageName := args[0]
		packageName := imageName
		if strings.Contains(packageName, "/") {
			packageName = strings.SplitN(packageName, "/", 2)[1]
		}
		err := pm.Install(imageName, packageName)
		if err != nil {
			return err
		}
		fmt.Printf("🐳  Installed %s to %s\n", imageName, path.Join(pm.InstallPath, packageName))
		return nil
	},
}
