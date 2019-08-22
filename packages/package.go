package packages

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/whalebrew/whalebrew/version"
	"github.com/whalebrew/whalebrew/client"
	"github.com/google/go-cmp/cmp"

	"github.com/docker/docker/api/types"
	"gopkg.in/yaml.v2"
)

const DefaultWorkingDir = "/workdir"

// Package represents a Whalebrew package
type Package struct {
	Name                string   `yaml:"-"`
	Entrypoint          []string `yaml:"entrypoint,omitempty"`
	Environment         []string `yaml:"environment,omitempty"`
	Image               string   `yaml:"image"`
	Volumes             []string `yaml:"volumes,omitempty"`
	Ports               []string `yaml:"ports,omitempty"`
	Networks            []string `yaml:"networks,omitempty"`
	WorkingDir          string   `yaml:"working_dir,omitempty"`
	KeepContainerUser   bool     `yaml:"keep_container_user,omitempty"`
	SkipMissingVolumes  bool     `yaml:"skip_missing_volumes,omitempty"`
	MountMissingVolumes bool     `yaml:"mount_missing_volumes,omitempty"`
	RequiredVersion     string   `yaml:"required_version,omitempty"`
}

// NewPackageFromImage creates a package from a given image name,
// inspecting the image to fetch the package configuration
func NewPackageFromImage(image string, imageInspect types.ImageInspect) (*Package, error) {
	name := image
	splittedName := strings.Split(name, "/")
	name = splittedName[len(splittedName)-1]
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

			if requiredVersion, ok := labels["io.whalebrew.required_version"]; ok {
				if err := version.CheckCompatible(requiredVersion); err != nil {
					return nil, err
				}
				pkg.RequiredVersion = requiredVersion
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

			if v, ok := labels["io.whalebrew.config.keep_container_user"]; ok {
				if err := yaml.Unmarshal([]byte(v), &pkg.KeepContainerUser); err != nil {
					return pkg, err
				}
			}

			if v, ok := labels["io.whalebrew.config.missing_volumes"]; ok {
				missingVolumes := "error"
				if err := yaml.Unmarshal([]byte(v), &missingVolumes); err != nil {
					return pkg, err
				}
				switch missingVolumes {
				case "error", "":
				case "skip":
					pkg.SkipMissingVolumes = true
				case "mount":
					pkg.MountMissingVolumes = true
				default:
					return pkg, fmt.Errorf("unexpected io.whalebrew.config.missing_volumes value: %s expecting error, skip or mount", missingVolumes)
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
		WorkingDir: DefaultWorkingDir,
	}
	if err = yaml.Unmarshal(d, pkg); err != nil {
		return pkg, err
	}

	if pkg.Name == "" {
		pkg.Name = filepath.Base(path)
	}

	if pkg.RequiredVersion != "" {
		if err := version.CheckCompatible(pkg.RequiredVersion); err != nil {
			return pkg, err
		}
	}

	return pkg, nil
}

// ImageInspect inspects the image associated with this package
func (pkg *Package) ImageInspect(cli *client.Client) (*types.ImageInspect, error) {
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

func (pkg *Package) HasChanges(ctx context.Context, cli *client.Client) (bool, error) {
	imageInspect, err := cli.ImageInspect(ctx, pkg.Image)
	if err != nil {
		return false, err
	}

	newPkg, err := NewPackageFromImage(pkg.Image, *imageInspect)
	if err != nil {
		return false, err
	}

	if newPkg.WorkingDir == "" {
		newPkg.WorkingDir = DefaultWorkingDir
	}

	return cmp.Equal(newPkg, pkg), nil
}
