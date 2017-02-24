package packages

import (
	"context"
	"fmt"
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
	Networks    []string `yaml:"networks,omitempty"`
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

		if networks, ok := labels["io.whalebrew.config.networks"]; ok {
			if err := yaml.Unmarshal([]byte(networks), &pkg.Networks); err != nil {
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

// DisplayPreinstallMessage displays the preinstall message of the package
func (pkg *Package) DisplayPreinstallMessage() bool {
	if len(pkg.Environment) == 0 && len(pkg.Volumes) == 0 && len(pkg.Ports) == 0 {
		return false
	}

	fmt.Println("This package needs additional access to your system:")
	if len(pkg.Environment) > 0 {
		fmt.Println("Environment Variables:")
		for _, env := range pkg.Environment {
			fmt.Printf("  * %s\n", env)
		}
	}

	if len(pkg.Volumes) > 0 {
		fmt.Println("Mounts:")
		for _, vol := range pkg.Volumes {
			fmt.Printf("  * %s\n", vol)
		}
	}

	if len(pkg.Ports) > 0 {
		fmt.Println("Ports:")
		for _, port := range pkg.Ports {
			fmt.Printf("  * %s\n", port)
		}
	}
	return true
}
