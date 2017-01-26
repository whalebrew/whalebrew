package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/bfirsh/whalebrew/cmd"
)

func main() {
	// HACK: if first argument starts with "/", prefix the subcommand run.
	// This allows us to use this command as a shebang, because we can't pass
	// the argument "run" in the shebang on Linux.
	if len(os.Args) > 1 && strings.HasPrefix(os.Args[1], "/") {
		cmd.RootCmd.SetArgs(append([]string{"run"}, os.Args[1:]...))
	}

	if err := cmd.RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}
