package cmd

import (
	"runtime"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	viper.SetEnvPrefix("whalebrew")
	viper.SetDefault("install_path", "/usr/local/bin")
	if runtime.GOOS == "windows" {
		viper.SetDefault("install_path", "C:\\whalebrew")
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
