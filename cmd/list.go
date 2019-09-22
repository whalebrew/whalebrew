package cmd

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/Masterminds/semver"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/go-units"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/whalebrew/whalebrew/client"
	"github.com/whalebrew/whalebrew/packages"
)

func init() {
	RootCmd.AddCommand(listCommand)
}

type listInfo struct {
	version      *semver.Version
	imageSummary *types.ImageSummary
	tag          string
}

type listInfos []*listInfo

func (c listInfos) Len() int {
	return len(c)
}

// Less is needed for the sort interface to compare two Version objects on the
// slice. If checks if one is less than the other.
func (c listInfos) Less(i, j int) bool {
	// Sort tags that aren't semver to front,
	// this shouldn't happen since we pre filter those to the back
	if c[i].version == nil {
		return false
	} else if c[j].version == nil {
		return true
	} else {
		return c[i].version.LessThan(c[j].version)
	}
}

// Swap is needed for the sort interface to replace the Version objects
// at two different positions in the slice.
func (c listInfos) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

var listCommand = &cobra.Command{
	Use:   "list [image]",
	Short: "List installed packages, or installable images for a package",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		if arglen := len(args); arglen > 0 {
			if arglen > 1 {
				return fmt.Errorf("invalid usage, this command accepts 0 or 1 arguments")
			}

			cli, err := client.NewClient()
			if err != nil {
				return err
			}

			return listImagesForPackage(ctx, cli, args[0])
		} else {
			return listInstalled()
		}
	},
}

func listImagesForPackage(ctx context.Context, cli *client.Client, pkgName string) error {
	pm := packages.NewPackageManager(viper.GetString("install_path"))
	if installed, err := pm.HasInstallation(pkgName); !installed {
		return err
	}

	pkg, err := pm.Load(pkgName)
	if err != nil {
		return err
	}

	ref, err := reference.ParseAnyReference(pkg.Image)
	if err != nil {
		return err
	}

	installedRef, ok := ref.(reference.NamedTagged)
	if !ok {
		return nil
	}

	installedTag := installedRef.Tag()

	name := strings.TrimPrefix(installedRef.Name(), "docker.io/library/")

	summaries, err := cli.ImageList(ctx, types.ImageListOptions{
		Filters: filters.NewArgs(filters.Arg("reference", name)),
	})

	if err != nil {
		return err
	}

	var infos []*listInfo
	var unsortableInfos []*listInfo

	for i, summary := range summaries {
		for _, tag := range summary.RepoTags {
			ref, _ := reference.Parse(tag)

			if namedTagged, ok := ref.(reference.NamedTagged); ok {
				v, err := semver.NewVersion(namedTagged.Tag())
				if err == nil {
					infos = append(infos, &listInfo{
						version:      v,
						imageSummary: &summaries[i],
						tag:          namedTagged.Tag(),
					})
				} else {
					unsortableInfos = append(unsortableInfos, &listInfo{
						version:      nil,
						imageSummary: &summaries[i],
						tag:          namedTagged.Tag(),
					})
				}
			}
		}
	}

	// pre filter non semver images to back for less swapping during sort
	infos = append(infos, unsortableInfos...)
	sort.Sort(listInfos(infos))

	w := tabwriter.NewWriter(os.Stdout, 10, 2, 2, ' ', 0)
	fmt.Fprintln(w, "INSTALLED\tTAG\tSIZE")
	for _, info := range infos {
		if info.tag == installedTag {
			fmt.Fprintf(w, "%s\t%s\t%v\n", ">", info.tag, units.HumanSizeWithPrecision(float64(info.imageSummary.Size), 3))
		} else {
			fmt.Fprintf(w, "%s\t%s\t%v\n", "", info.tag, units.HumanSizeWithPrecision(float64(info.imageSummary.Size), 3))
		}
	}
	w.Flush()

	return nil
}

func listInstalled() error {
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
}
