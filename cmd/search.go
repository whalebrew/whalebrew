package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/spf13/cobra"
)

type imageResult struct {
	User string `json:"user"`
	Name string `json:"name"`
}

type searchAnswer struct {
	Results []imageResult `json:"results"`
}

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

		params := url.Values{}
		params.Set("page_size", "100")
		params.Set("ordering", "last_updated")
		if len(args) > 0 {
			params.Set("name", args[0])
		}
		u := url.URL{
			Scheme:   "https",
			Host:     "hub.docker.com",
			Path:     "/v2/repositories/whalebrew/",
			RawQuery: params.Encode(),
		}

		r, err := http.Get(u.String())
		if err != nil {
			return err
		}
		answer := searchAnswer{}
		err = json.NewDecoder(r.Body).Decode(&answer)
		if err != nil {
			return err
		}
		for _, image := range answer.Results {
			fmt.Printf("%s/%s\n", image.User, image.Name)
		}
		return nil
	},
}
