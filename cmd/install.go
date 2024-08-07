package cmd

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/Songmu/prompter"
	"github.com/spf13/cobra"
	"github.com/whalebrew/whalebrew/config"
	"github.com/whalebrew/whalebrew/hooks"
	"github.com/whalebrew/whalebrew/packages"
	"github.com/whalebrew/whalebrew/run"
)

var customPackageName string
var customEntrypoint string
var forceInstall bool
var assumeYes bool
var strict bool

type multipleErrors []error

func (e multipleErrors) Error() string {
	r := ""
	for _, err := range e {
		r = r + fmt.Sprintf("%s\n", err.Error())
	}
	return r
}

func init() {
	installCommand.Flags().StringVarP(&customPackageName, "name", "n", "", "Name to give installed package. Defaults to image name.")
	installCommand.Flags().StringVarP(&customEntrypoint, "entrypoint", "e", "", "Custom entrypoint to run the image with. Defaults to image entrypoint.")
	installCommand.Flags().BoolVarP(&forceInstall, "force", "f", false, "Replace existing package if already exists. Defaults to false.")
	installCommand.Flags().BoolVarP(&assumeYes, "assume-yes", "y", false, "Assume 'yes' as answer to all prompts and run non-interactively. Defaults to false.")
	installCommand.Flags().BoolVar(&strict, "strict", false, "Fail installing the image if it contains any skippable error. Defaults to false.")

	RootCmd.AddCommand(installCommand)
}

var installCommand = &cobra.Command{
	Use:   "install IMAGENAME",
	Short: "Install a package",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		if len(args) < 1 {
			return cmd.Help()
		}
		if len(args) > 1 {
			return fmt.Errorf("Only one image can be installed at a time")
		}

		imageName := args[0]

		docker, err := run.NewDockerLikeRunner()
		if err != nil {
			return err
		}

		imageInspect, err := docker.ImageInspect(imageName)
		if err != nil {
			return err
		}

		var errorList multipleErrors
		packages.LintImage(imageInspect, func(e error) {
			switch e.(type) {
			case packages.NoEntrypointError:
				// Exception is done for entrypoint, install offers the ability to customise its value
				if customEntrypoint != "" {
					return
				}
			}
			if s, ok := e.(packages.StrictError); strict == true || !ok || s.Strict() {
				errorList = append(errorList, e)
			}
		})
		if errorList != nil {
			return errorList
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

		installDir := config.GetConfig().InstallPath
		// we have introduced a breaking change when releasing whalebrew 0.5.0
		// Possibly, previous installations on darwin arm64 were using /usr/local/bin.
		// Emmit a gentle warning to our users.
		if config.GetConfig().IsDefaultInstallPath() && installDir == "/opt/whalebrew/bin" {
			info, err := os.Stat("/usr/local/bin")
			if err == nil {
				if stat, ok := info.Sys().(*syscall.Stat_t); ok {
					currentUser, err := user.Current()
					if err == nil {
						if strconv.FormatUint(uint64(stat.Uid), 10) == currentUser.Uid {
							fmt.Println("📌  Default whalebrew installation path on darwin arm64 was changed from /usr/local/bin to /opt/whalebrew/bin")
							fmt.Println(`To keep using /usr/local/bin, set the environment variable 'WHALEBREW_INSTALL_PATH=/usr/local/bin' or add install_path: "/usr/local/bin" to your config path`, config.ConfigPath())
							fmt.Println(`This message will be removed in whalebrew 0.6.0`)
						}
					}
				}
			}
		}
		pm := packages.NewPackageManager(installDir)

		_, err = os.Stat(installDir)
		if err != nil && os.IsNotExist(err) {
			err := os.MkdirAll(installDir, 0755)
			if err != nil {
				fmt.Println("ℹ️   Install directory", installDir, "is missing and requires elevated privileges to be created. Creating it with sudo")
				c := exec.Command("sudo", "mkdir", "-m", "0755", "-p", installDir)
				c.Stdout = os.Stdout
				c.Stderr = os.Stderr
				err = c.Run()
				if err != nil {
					return fmt.Errorf("failed to create non-existing installation directory: %v", err)
				}
				currentUser, err := user.Current()
				if err != nil {
					return fmt.Errorf("failed to change ownership of install directory to current user: %v", err)
				}

				c = exec.Command("sudo", "chown", "-R", currentUser.Username+":"+currentUser.Gid, strings.TrimSuffix(installDir, "/bin"))
				c.Stdout = os.Stdout
				c.Stderr = os.Stderr
				err = c.Run()
				if err != nil {
					return fmt.Errorf("failed to create non-existing installation directory: %v", err)
				}
			}
		}

		var installed *packages.Package
		hasInstall := pm.HasInstallation(pkg.Name)
		if hasInstall {
			installed, err = pm.Load(pkg.Name)
			if !forceInstall && err != nil {
				return fmt.Errorf("there's already an installation of %s, but there was an error loading the package, err: %s", pkg.Name, err.Error())
			}

			fmt.Printf("Looks like you already have %s installed as %s.\n", installed.Image, path.Join(installDir, pkg.Name))

			if !assumeYes {
				if changed, diff, err := installed.HasChanges(ctx, docker); err != nil {
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
			var patherr *fs.PathError
			if errors.As(err, &patherr) {
				return fmt.Errorf("Installation path is not writable: %s\n\nSet WHALEBREW_INSTALL_PATH environment variable to writable location.\nOr set 'install_path` option in '~/.whalebrew/config.yaml`. Make sure\nthe location is added to PATH. For details, see\nhttps://github.com/whalebrew/whalebrew#configuration\n", installDir)
			}
			return err
		}

		if err := hooks.Run("post-install", pkg.Name); err != nil {
			return fmt.Errorf("post install script failed: %s", err.Error())
		}

		installPath := filepath.Clean(path.Join(pm.InstallPath, pkg.Name))
		if hasInstall {
			fmt.Printf("🐳  Modified %s to use %s\n", installPath, imageName)
		} else {
			fmt.Printf("🐳  Installed %s to %s\n", imageName, installPath)
		}

		cmdPath, err := exec.LookPath(pkg.Name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❗️  Installed command %s does not seem to be available after install. Ensure you add %v to your $PATH to be able to use it\n", pkg.Name, pm.InstallPath)
		} else if cmdPath != installPath {
			fmt.Fprintf(os.Stderr, "❗️  Installed command %s does not point to installed path %s but to %s. Ensure %v is in the relevant poistion of your $PATH to be able to use it\n", pkg.Name, installPath, cmdPath, pm.InstallPath)
		}
		return nil
	},
}
