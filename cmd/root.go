package cmd

import (
	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:   "whalebrew",
	Short: "Install Docker images as native commands",
}
