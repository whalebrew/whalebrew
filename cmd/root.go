package cmd

import (
	"github.com/spf13/cobra"
)

// RootCmd is the root CLI command
var RootCmd = &cobra.Command{
	Use:           "whalebrew",
	Short:         "Install Docker images as native commands",
	SilenceUsage:  true,
	SilenceErrors: true,
}
