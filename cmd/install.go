package cmd

import (
	"context"
	"fmt"
	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/whalebrew/whalebrew/packages"
	"os"
	"os/exec"
	"path"
	"strings"
)

var packageName string

func init() {
	installCommand.Flags().StringVarP(&packageName, "name", "n", "", "Name to give installed package. Defaults to image name.")

	RootCmd.AddCommand(installCommand)
}

var installCommand = &cobra.Command{
	Use:   "install IMAGENAME",
	Short: "Install a package",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return cmd.Help()
		}
		if len(args) > 1 {
			return fmt.Errorf("Only one image can be installed at a time")
		}

		imageName := args[0]
		if packageName == "" {
			packageName = imageName
			if strings.Contains(packageName, "/") {
				packageName = strings.SplitN(packageName, "/", 2)[1]
			}
		}
		// TODO (bfirsh): validate names

		cli, err := client.NewEnvClient()
		if err != nil {
			return err
		}
		img, _, err := cli.ImageInspectWithRaw(context.Background(), imageName)
		if err != nil {
			if client.IsErrNotFound(err) {
				fmt.Printf("Unable to find image '%s' locally\n", imageName)
				if err = pullImage(imageName); err != nil {
					return err
				}
				// retry
				img, _, err = cli.ImageInspectWithRaw(context.Background(), imageName)
				if err != nil {
					return err
				}
			} else {
				return err
			}
		}
		if img.ContainerConfig.Entrypoint == nil {
			return fmt.Errorf("The image '%s' is not compatible with Whalebrew: it does not have an entrypoint.", imageName)
		}

		pm := packages.NewPackageManager(viper.GetString("install_path"))
		err = pm.Install(imageName, packageName)
		if err != nil {
			return err
		}
		fmt.Printf("🐳  Installed %s to %s\n", imageName, path.Join(pm.InstallPath, packageName))
		return nil
	},
}

func pullImage(image string) error {
	c := exec.Command("docker", "pull", image)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
