package cmd

import (
	"context"
	"fmt"
	"sort"

	"github.com/bfirsh/whalebrew/client"
	"github.com/docker/docker/api/types"
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(searchCommand)
}

var searchCommand = &cobra.Command{
	Use:   "search [TERM]",
	Short: "Search for packages on Docker Hub",
	Long:  "Search for Whalebrew packages on Docker Hub. If no search term is provided, all packages are listed.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 1 {
			return fmt.Errorf("Only one search term is supported")
		}

		cli, err := client.NewClient()
		if err != nil {
			return err
		}

		term := "whalebrew/"
		if len(args) == 1 {
			term = term + args[0]
		}

		options := types.ImageSearchOptions{Limit: 100}
		results, err := cli.ImageSearch(context.Background(), term, options)
		if err != nil {
			return err
		}

		names := make([]string, len(results))
		for i, result := range results {
			names[i] = result.Name
		}
		sort.Strings(names)

		for _, name := range names {
			fmt.Println(name)
		}

		return nil
	},
}
