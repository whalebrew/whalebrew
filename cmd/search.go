package cmd

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(searchCommand)
}

var searchCommand = &cobra.Command{
	Use:   "search TERM",
	Short: "Search for packages on Docker Hub",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return cmd.Help()
		}
		if len(args) > 1 {
			return fmt.Errorf("Only one search term is supported")
		}

		cli, err := client.NewEnvClient()
		if err != nil {
			return err
		}

		term := "whalebrew/" + args[0]
		options := types.ImageSearchOptions{Limit: 100}
		results, err := cli.ImageSearch(context.Background(), term, options)
		if err != nil {
			return err
		}

		for _, res := range results {
			fmt.Println(res.Name)
		}

		return nil
	},
}
