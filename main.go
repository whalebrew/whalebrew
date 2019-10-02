package main

import (
	"fmt"
	"os"

	"github.com/whalebrew/whalebrew/cmd"
)

func main() {
	var err error
	if cmd.IsShellbang(os.Args) {
		err = cmd.DockerCLIRun(os.Args)
	} else {
		err = cmd.RootCmd.Execute()
	}

	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}
