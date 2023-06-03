package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/whalebrew/whalebrew/version"
)

func init() {
	RootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of Whalebrew",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Whalebrew %s\n", version.Version)
	},
}
