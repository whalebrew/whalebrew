package cmd

import (
	"context"
	"fmt"
	"path"

	"github.com/Songmu/prompter"
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
	installCommand.Flags().StringVarP(&customEntrypoint, "entrypoint", "e", "", "Custom entrypoint to run the image with. Defaults to image entrypoint.")
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

		ctx := context.Background()

		imageInspect, err := cli.ImageInspect(ctx, imageName)
		if err != nil {
			return err
		}
		
		if imageInspect.Config.Entrypoint == nil && customEntrypoint == "" {
			return fmt.Errorf("The image '%s' does not have an entrypoint, please provide a custom entrypoint", imageName)
		}

		pkg, err := packages.NewPackageFromImage(imageName, *imageInspect)
		if err != nil {
			return err
		}
		if customPackageName != "" {
			pkg.Name = customPackageName
		}

		if customEntrypoint != "" {
			pkg.Entrypoint = []string{customEntrypoint}
		}

		installPath := viper.GetString("install_path")
		pm := packages.NewPackageManager(installPath)

		var installed *packages.Package
		hasInstall := pm.HasInstallation(pkg.Name)
		if hasInstall {
			installed, err = pm.Load(pkg.Name)
			if !forceInstall && err != nil {
				return fmt.Errorf("there's already an installation of %s, but there was an error loading the package, err: %s", pkg.Name, err.Error())
			}

			fmt.Printf("Looks like you already have %s installed as %s.\n", installed.Image, path.Join(installPath, pkg.Name))

			if !assumeYes {
				if changed, diff, err := installed.HasChanges(ctx, cli); err != nil {
					return err
				} else if changed {
					fmt.Println("There are differences between the installed version of the package and the image:")
					fmt.Println(diff)

					if !prompter.YN("Are you sure you would like to overwrite these changes?", false) {
						return fmt.Errorf("Not installing package")
					}
				} else if pkg.Image == installed.Image {
					fmt.Printf("%s would generate the same package, nothing to do\n", pkg.Image)
					return nil
				}

				if pkg.Image != installed.Image && !prompter.YN(fmt.Sprintf("Would you like to change %s to %s?", installed.Image, pkg.Image), true) {
					return fmt.Errorf("Not installing package")
				}
			}
			forceInstall = true
		}

		preinstallMessage := pkg.PreinstallMessage(installed)
		if preinstallMessage != "" {
			fmt.Println(preinstallMessage)
			if !assumeYes {
				if !prompter.YN("Is this okay?", true) {
					return fmt.Errorf("Not installing package")
				}
			}
		}

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

		if hasInstall {
			fmt.Printf("🐳  Modified %s to use %s\n", path.Join(pm.InstallPath, pkg.Name), imageName)
		} else {
			fmt.Printf("🐳  Installed %s to %s\n", imageName, path.Join(pm.InstallPath, pkg.Name))
		}
		return nil
	},
}

