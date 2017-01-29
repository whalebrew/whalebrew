package cmd

import (
	"fmt"

	"github.com/bfirsh/whalebrew/version"
	"github.com/spf13/cobra"
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
