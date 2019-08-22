package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"

	"github.com/Songmu/prompter"
	dockerClient "github.com/docker/docker/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/whalebrew/whalebrew/client"
	"github.com/whalebrew/whalebrew/hooks"
	"github.com/whalebrew/whalebrew/packages"
)

var customPackageName string
var customEntrypoint string
var forceInstall bool
var assumeYes bool

func init() {
	installCommand.Flags().StringVarP(&customPackageName, "name", "n", "", "Name to give installed package. Defaults to image name.")
	installCommand.Flags().StringVarP(&customEntrypoint, "entrypoint", "e", "", "Alternate entrypoint to run the image with. Defaults to image entrypoint.")
	installCommand.Flags().BoolVarP(&forceInstall, "force", "f", false, "Replace existing package if already exists. Defaults to false.")
	installCommand.Flags().BoolVarP(&assumeYes, "assume-yes", "y", false, "Assume 'yes' as answer to all prompts and run non-interactively. Defaults to false.")

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
		if imageInspect.Config.Entrypoint == nil {
			return fmt.Errorf("the image '%s' is not compatible with Whalebrew: it does not have an entrypoint", imageName)
		}

		pkg, err := packages.NewPackageFromImage(imageName, imageInspect)
		if err != nil {
			return err
		}
		if customPackageName != "" {
			pkg.Name = customPackageName
		}

		if customEntrypoint != "" {
			pkg.Entrypoint = []string{customEntrypoint}
		}

		preinstallMessage := pkg.PreinstallMessage()
		if preinstallMessage != "" {
			fmt.Println(preinstallMessage)
			if !assumeYes {
				if !prompter.YN("Is this okay?", true) {
					return fmt.Errorf("Not installing package")
				}
			}
		}

		pm := packages.NewPackageManager(viper.GetString("install_path"))

		if err := hooks.Run("pre-install", imageName, pkg.Name); err != nil {
			return fmt.Errorf("pre install script failed: %s", err.Error())
		}

		if forceInstall {
			err = pm.ForceInstall(pkg)
		} else {
			err = pm.Install(pkg)
		}
		if err != nil {
			return err
		}

		if err := hooks.Run("post-install", pkg.Name); err != nil {
			return fmt.Errorf("post install script failed: %s", err.Error())
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
