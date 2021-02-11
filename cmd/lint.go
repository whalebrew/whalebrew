package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/whalebrew/whalebrew/client"
	"github.com/whalebrew/whalebrew/packages"
)

func init() {
	RootCmd.AddCommand(lintCommand)
}

type ErrorWithImage struct {
	Image string
	Err   error
}

func (e ErrorWithImage) Error() string {
	return fmt.Sprintf("with image %s: %v", e.Image, e.Err)
}

var lintCommand = &cobra.Command{
	Use:   "lint IMAGENAME",
	Short: "lints a package",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return cmd.Help()
		}
		cli, err := client.NewClient()
		if err != nil {
			return err
		}

		var errors multipleErrors
		for _, imageName := range args {
			ctx := context.Background()

			imageInspect, err := cli.ImageInspect(ctx, imageName)
			if err != nil {
				errors = append(errors, ErrorWithImage{Image: imageName, Err: err})
				return err
			}
			packages.LintImage(*imageInspect, func(e error) {
				if s, ok := e.(packages.StrictError); strict == true || !ok || s.Strict() {
					errors = append(errors, ErrorWithImage{Image: imageName, Err: e})
				}
			})
		}
		if errors != nil {
			return errors
		}
		return nil
	},
}
