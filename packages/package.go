package packages

import (
	"context"
	"io/ioutil"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"gopkg.in/yaml.v2"
)

// Package represents a Whalebrew package
type Package struct {
	Name        string   `yaml:"-"`
	Environment []string `yaml:"environment,omitempty"`
	Image       string   `yaml:"image"`
	Volumes     []string `yaml:"volumes,omitempty"`
	Ports       []string `yaml:"ports,omitempty"`
}

// NewPackageFromImage creates a package from a given image name,
// inspecting the image to fetch the package configuration
func NewPackageFromImage(image string, imageInspect types.ImageInspect) (*Package, error) {
	name := image
	if strings.Contains(name, "/") {
		name = strings.SplitN(name, "/", 2)[1]
	}
	if strings.Contains(name, ":") {
		name = strings.SplitN(name, ":", 2)[0]
	}
	pkg := &Package{
		Name:  name,
		Image: image,
	}

	if imageInspect.ContainerConfig != nil && imageInspect.ContainerConfig.Labels != nil {
		labels := imageInspect.ContainerConfig.Labels

		if name, ok := labels["io.whalebrew.name"]; ok {
			pkg.Name = name
		}

		if env, ok := labels["io.whalebrew.config.environment"]; ok {
			if err := yaml.Unmarshal([]byte(env), &pkg.Environment); err != nil {
				return pkg, err
			}
		}

		if volumesStr, ok := labels["io.whalebrew.config.volumes"]; ok {
			if err := yaml.Unmarshal([]byte(volumesStr), &pkg.Volumes); err != nil {
				return pkg, err
			}
		}

		if ports, ok := labels["io.whalebrew.config.ports"]; ok {
			if err := yaml.Unmarshal([]byte(ports), &pkg.Ports); err != nil {
				return pkg, err
			}
		}
	}

	return pkg, nil
}

// LoadPackageFromPath reads a package from the given path
func LoadPackageFromPath(path string) (*Package, error) {
	d, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	pkg := &Package{}
	if err = yaml.Unmarshal(d, pkg); err != nil {
		return pkg, err
	}
	return pkg, nil
}

// ImageInspect inspects the image associated with this package
func (pkg *Package) ImageInspect() (*types.ImageInspect, error) {
	cli, err := client.NewEnvClient()
	if err != nil {
		return nil, err
	}
	img, _, err := cli.ImageInspectWithRaw(context.Background(), pkg.Image)
	return &img, err
}
