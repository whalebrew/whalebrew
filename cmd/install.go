package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"

	"github.com/bfirsh/whalebrew/client"
	"github.com/bfirsh/whalebrew/packages"
	dockerClient "github.com/docker/docker/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var customPackageName string

func init() {
	installCommand.Flags().StringVarP(&customPackageName, "name", "n", "", "Name to give installed package. Defaults to image name.")

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

		cli, err := client.NewClient()
		if err != nil {
			return err
		}
		imageInspect, _, err := cli.ImageInspectWithRaw(context.Background(), imageName)
		if err != nil {
			if dockerClient.IsErrNotFound(err) {
				fmt.Printf("Unable to find image '%s' locally\n", imageName)
				if err = pullImage(imageName); err != nil {
					return err
				}
				// retry
				imageInspect, _, err = cli.ImageInspectWithRaw(context.Background(), imageName)
				if err != nil {
					return err
				}
			} else {
				return fmt.Errorf("failed to inspect docker image: %v", err)
			}
		}
		if imageInspect.ContainerConfig.Entrypoint == nil {
			return fmt.Errorf("the image '%s' is not compatible with Whalebrew: it does not have an entrypoint", imageName)
		}

		pkg, err := packages.NewPackageFromImage(imageName, imageInspect)
		if err != nil {
			return err
		}
		if customPackageName != "" {
			pkg.Name = customPackageName
		}
		pm := packages.NewPackageManager(viper.GetString("install_path"))
		err = pm.Install(pkg)
		if err != nil {
			return err
		}
		fmt.Printf("🐳  Installed %s to %s\n", imageName, path.Join(pm.InstallPath, pkg.Name))
		return nil
	},
}

func pullImage(image string) error {
	c := exec.Command("docker", "pull", image)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
