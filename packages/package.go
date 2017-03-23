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
	WorkingDir  string   `yaml:"working_dir,omitempty"`
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

	if imageInspect.ContainerConfig != nil {

		if imageInspect.ContainerConfig.WorkingDir != "" {
			pkg.WorkingDir = imageInspect.ContainerConfig.WorkingDir
		}

		if imageInspect.ContainerConfig.Labels != nil {
			labels := imageInspect.ContainerConfig.Labels

			if name, ok := labels["io.whalebrew.name"]; ok {
				pkg.Name = name
			}

			if workingDir, ok := labels["io.whalebrew.config.working_dir"]; ok {
				pkg.WorkingDir = workingDir
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
	}

	return pkg, nil
}

// LoadPackageFromPath reads a package from the given path
func LoadPackageFromPath(path string) (*Package, error) {
	d, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	pkg := &Package{
		WorkingDir: "/workdir",
	}
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

// PreinstallMessage returns the preinstall message for the package
func (pkg *Package) PreinstallMessage() string {
	if len(pkg.Environment) == 0 && len(pkg.Volumes) == 0 && len(pkg.Ports) == 0 {
		return ""
	}

	out := []string{"This package needs additional access to your system. It wants to:", ""}
	for _, env := range pkg.Environment {
		out = append(out, fmt.Sprintf("* Read the environment variable %s", env))
	}

	if len(pkg.Ports) > 0 {
		for _, port := range pkg.Ports {
			// no support for interfaces (e.g. 127.0.0.1:80:80)
			portNumber := strings.Split(port, ":")[0]
			proto := "TCP"
			if strings.HasSuffix(port, "udp") {
				proto = "UDP"
			}
			out = append(out, fmt.Sprintf("* Listen on %s port %s", proto, portNumber))
		}
	}

	for _, vol := range pkg.Volumes {
		if len(strings.Split(vol, ":")) > 1 {
			text := "* Read and write to the file or directory %q"
			if strings.HasSuffix(vol, "ro") {
				text = "* Read the file or directory %q"
			}
			out = append(out, fmt.Sprintf(text, strings.Split(vol, ":")[0]))
		}
	}

	return strings.Join(out, "\n") + "\n"
}
