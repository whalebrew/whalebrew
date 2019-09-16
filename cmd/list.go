package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/Masterminds/semver"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/distribution"
	"github.com/docker/docker/errdefs"
	"github.com/docker/docker/registry"
	"github.com/docker/go-units"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/whalebrew/whalebrew/client"
	"github.com/whalebrew/whalebrew/packages"
	registryClient "github.com/docker/distribution/registry/client"
	 "github.com/docker/distribution/registry/client/transport"
	"github.com/docker/distribution/registry/client/auth"
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
	Short: "List installed packages, or tags for an image",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		reg, err := registryClient.NewRegistry("https://gcr.io", transport.NewTransport(http.DefaultTransport, ))
		if err != nil {
			return err
		}
		repos := make([]string, 200)
		_, err = reg.Repositories(ctx, repos, "")
		if err !=  nil {
			return  err
		}

		if arglen := len(args); arglen > 0 {
			if arglen > 1 {
				return fmt.Errorf("invalid usage, this command accepts 0 or 1 arguments")
			}
			image := args[0]
			ref, err := reference.ParseAnyReference(image)
			if err != nil {
				return err
			}
			namedRef, ok := ref.(reference.Named)
			if !ok {
				if _, ok := ref.(reference.Digested); ok {
					return nil
				}
				return errors.Errorf("unknown image reference format: %s", image)
			}
			// only query registry if not a canonical reference (i.e. with digest)
			if _, ok := namedRef.(reference.Canonical); !ok {
				namedRef = reference.TagNameOnly(namedRef)

				taggedRef, ok := namedRef.(reference.NamedTagged)
				if !ok {
					return errors.Errorf("image reference not tagged: %s", image)
				}

				repoInfo, err := registry.ParseRepositoryInfo(taggedRef)
				if err != nil {
					return err
				}

				if err := distribution.ValidateRepoName(repoInfo.Name); err != nil {
					return errdefs.InvalidParameter(err)
				}
				repository, confirmedV2, lastError = distribution.NewV2Repository(ctx, repoInfo, endpoint, nil, authConfig, "pull")
				repo, _, err := c.config.ImageBackend.GetRepository(ctx, taggedRef, authConfig)
			}

			cli, err := client.NewClient()
			if err != nil {
				return err
			}

			return listTagsForImage(ctx, cli, args[0])
		} else {
			return listInstalled()
		}
	},
}

func listTagsForImage(ctx context.Context, cli *client.Client, image string) error {
	summaries, err := cli.ImageList(ctx, types.ImageListOptions{
		Filters: filters.NewArgs(filters.Arg("reference", image)),
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

	// pre filter non semver images to back for less swapping
	infos = append(infos, unsortableInfos...)
	sort.Sort(listInfos(infos))

	w := tabwriter.NewWriter(os.Stdout, 10, 2, 2, ' ', 0)
	fmt.Fprintln(w, "TAG\tSIZE")
	for _, info := range infos {
		fmt.Fprintf(w, "%s\t%v\n", info.tag, units.HumanSizeWithPrecision(float64(info.imageSummary.Size), 3))
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
