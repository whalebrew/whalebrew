package cmd

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func home() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func init() {
	viper.SetEnvPrefix("whalebrew")
	viper.SetDefault("install_path", "/usr/local/bin")
	viper.SetDefault("config_dir", filepath.Join(home(), ".whalebrew"))
	viper.BindEnv("install_path")
	viper.BindEnv("config_dir")
}

// RootCmd is the root CLI command
var RootCmd = &cobra.Command{
	Use:           "whalebrew",
	Short:         "Install Docker images as native commands",
	SilenceUsage:  true,
	SilenceErrors: true,
}
