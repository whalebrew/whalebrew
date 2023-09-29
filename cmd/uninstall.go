package cmd

import (
	"fmt"
	"os"
	"path"

	"github.com/Songmu/prompter"
	"github.com/spf13/cobra"
	"github.com/whalebrew/whalebrew/config"
	"github.com/whalebrew/whalebrew/hooks"
	"github.com/whalebrew/whalebrew/packages"
)

var forceUninstall bool

func init() {
	uninstallCommand.Flags().BoolVarP(&assumeYes, "assume-yes", "y", false, "Assume 'yes' as answer to all prompts and run non-interactively. Defaults to false.")

	RootCmd.AddCommand(uninstallCommand)
}

type deleteReason string

var (
	deleteReasonPackageNameMatches  deleteReason = "package name matches"
	deleteReasonPackageImageMatches deleteReason = "package image matches"
)

type deletionCandidate struct {
	pkg    *packages.Package
	reason deleteReason
}

var uninstallCommand = &cobra.Command{
	Use:   "uninstall PACKAGENAME|IMAGENAME",
	Short: "Uninstall a package",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return cmd.Help()
		}
		if len(args) > 1 {
			return fmt.Errorf("Only one image can be uninstalled at a time")
		}

		pm := packages.NewPackageManager(config.GetConfig().InstallPath)
		packageNameOrImage := args[0]

		if err := hooks.Run("pre-uninstall", packageNameOrImage); err != nil {
			return fmt.Errorf("pre-uninstall install script failed: %s", err.Error())
		}
		packages, err := pm.List()
		if err != nil {
			return fmt.Errorf("unable to list packages: %v", err)
		}

		candidates := []deletionCandidate{}
		for _, pkg := range packages {
			if pkg.Name == packageNameOrImage {
				candidates = append(candidates, deletionCandidate{
					pkg:    pkg,
					reason: deleteReasonPackageNameMatches,
				})
			} else if pkg.Image == packageNameOrImage {
				candidates = append(candidates, deletionCandidate{
					pkg:    pkg,
					reason: deleteReasonPackageImageMatches,
				})
			}
		}
		if len(candidates) > 1 {
			fmt.Fprintln(os.Stderr, "Many mathcing packages found. Run:")
			for _, candidate := range candidates {
				fmt.Fprintln(os.Stderr, "'whalebrew uninstall ", candidate.pkg.Name, "' to uninstall image ", candidate.pkg.Image)
			}
			os.Exit(1)
		}
		if len(candidates) == 0 {
			fmt.Fprintf(os.Stderr, "Unable to find a package with name or image %s\n", packageNameOrImage)
			return nil
		}

		path := path.Join(pm.InstallPath, candidates[0].pkg.Name)
		if !assumeYes {
			if !prompter.YN(fmt.Sprintf("This will permanently delete '%s'. Are you sure?", path), false) {
				return nil
			}
		}

		err = pm.Uninstall(packageNameOrImage)
		if err != nil {
			return err
		}

		if err := hooks.Run("post-uninstall", packageNameOrImage); err != nil {
			return fmt.Errorf("post-uninstall install script failed: %s", err.Error())
		}
		fmt.Printf("🚽  Uninstalled %s\n", path)

		return nil
	},
}
