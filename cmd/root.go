package cmd

import (
	"os/user"
	"path"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	viper.SetEnvPrefix("whalebrew")
	if runtime.GOOS == "windows" {
		currentUser, _ := user.Current()
		viper.SetDefault("install_path", path.Join(currentUser.HomeDir, "whalebrew"))
	} else {
		viper.SetDefault("install_path", "/usr/local/bin")
	}
	viper.BindEnv("install_path")
}

// RootCmd is the root CLI command
var RootCmd = &cobra.Command{
	Use:           "whalebrew",
	Short:         "Install Docker images as native commands",
	SilenceUsage:  true,
	SilenceErrors: true,
}
