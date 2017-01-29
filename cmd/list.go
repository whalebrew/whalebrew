package cmd

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/bfirsh/whalebrew/packages"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	RootCmd.AddCommand(listCommand)
}

var listCommand = &cobra.Command{
	Use:   "list",
	Short: "List installed packages",
	RunE: func(cmd *cobra.Command, args []string) error {
		pm := packages.NewPackageManager(viper.GetString("install_path"))
		packages, err := pm.List()
		if err != nil {
			return err
		}

		packageNames := make([]string, 0, len(packages))
		for k := range packages {
			packageNames = append(packageNames, k)
		}
		sort.Strings(packageNames)

		w := tabwriter.NewWriter(os.Stdout, 10, 2, 2, ' ', 0)
		fmt.Fprintln(w, "COMMAND\tIMAGE")
		for _, name := range packageNames {
			fmt.Fprintf(w, "%s\t%s\n", name, packages[name].Image)
		}
		w.Flush()
		return nil
	},
}
