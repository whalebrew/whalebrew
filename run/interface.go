package run

import (
	"os/user"

	imagev1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// Execution defunes elements that depends on the current runtime request
type Execution struct {
	Image             string
	Entrypoint        []string
	Ports             []string
	Networks          []string
	KeepContainerUser bool
	Environment       []string
	IsTTYOpened       bool
	Args              []string
	User              *user.User
	WorkingDir        string
	Volumes           []string
}

// Runner must run until compoletion and return an error wether something failed
type Runner interface {
	Run(e *Execution) error
}

type ImageInspecter interface {
	ImageInspect(imageName string) (*imagev1.Image, error)
}
