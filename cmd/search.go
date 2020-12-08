package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/whalebrew/whalebrew/config"
	"github.com/whalebrew/whalebrew/search"
)

func init() {
	RootCmd.AddCommand(searchCommand)
}

var searchCommand = &cobra.Command{
	Use:   "search [TERM]",
	Short: "Search for packages on Docker Hub",
	Long:  "Search for Whalebrew packages on Docker Hub. If no search term is provided, all packages are listed.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return cmd.Help()
		}
		errHandler := func(err error) bool {
			fmt.Println(err.Error())
			os.Exit(1)
			return true
		}
		for searcher := range search.ForRegistries(config.GetConfig().Registries, errHandler) {
			for image := range searcher.Search(args[0], errHandler) {
				fmt.Println(image)
			}
		}
		return nil
	},
}
