package run

import (
	"os/user"

	"github.com/whalebrew/whalebrew/packages"
)

// Execution defunes elements that depends on the current runtime request
type Execution struct {
	Environment []string
	IsTTYOpened bool
	Args        []string
	User        *user.User
	WorkingDir  string
	Volumes     []string
}

// Runner must run until compoletion and return an error wether something failed
type Runner interface {
	Run(p *packages.Package, e *Execution) error
}
