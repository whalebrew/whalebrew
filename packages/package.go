package packages

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/google/go-cmp/cmp"
	"github.com/whalebrew/whalebrew/client"
	"github.com/whalebrew/whalebrew/version"

	"github.com/docker/docker/api/types"
	"gopkg.in/yaml.v2"
)

const DefaultWorkingDir = "/workdir"

var (
	DefaultLoader = YamlLoader{}
)

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
	PathArguments       []string `yaml:"path_arguments,omitempty"`
	CustomGid string `yaml:"customgid,omitempty"`
}

// Loader loads a package from a given path
type Loader interface {
	LoadPackageFromPath(path string) (*Package, error)
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

			if pathArgs, ok := labels["io.whalebrew.config.volumes_from_args"]; ok {
				args := []string{}
				if err := yaml.Unmarshal([]byte(pathArgs), &args); err != nil {
					return pkg, err
				}
				for _, arg := range args {
					pkg.PathArguments = append(pkg.PathArguments, strings.TrimLeft(arg, "-"))
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

			if customgid, ok := labels["io.whalebrew.config.customgid"]; ok {
				if err := yaml.Unmarshal([]byte(customgid), &pkg.CustomGid); err != nil {
					return pkg, err
				}
			}
		}
	}

	return pkg, nil
}

// YamlLoader reads a package stored as yaml
type YamlLoader struct{}

// LoadPackageFromPath reads a package from the given path
func (y YamlLoader) LoadPackageFromPath(path string) (*Package, error) {
	d, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	pkg := &Package{
		WorkingDir: DefaultWorkingDir,
		Name:       filepath.Base(path),
	}

	if err = yaml.Unmarshal(d, pkg); err != nil {
		return pkg, err
	}

	if pkg.RequiredVersion != "" {
		if err := version.CheckCompatible(pkg.RequiredVersion); err != nil {
			return pkg, err
		}
	}

	return pkg, nil
}

// LoadPackageFromPath reads a package from the given path
func LoadPackageFromPath(path string) (*Package, error) {
	return DefaultLoader.LoadPackageFromPath(path)
}

// PreinstallMessage returns the preinstall message for the package
func (pkg *Package) PreinstallMessage(prevInstall *Package) string {
	var permissionReporter *PermissionChangeReporter
	if prevInstall == nil {
		permissionReporter = NewPermissionChangeReporter(true)
		cmp.Equal(&Package{}, pkg, cmp.Reporter(permissionReporter))
	} else {
		permissionReporter = NewPermissionChangeReporter(false)
		cmp.Equal(prevInstall, pkg, cmp.Reporter(permissionReporter))
	}

	return permissionReporter.String()
}

func (pkg *Package) HasChanges(ctx context.Context, cli *client.Client) (bool, string, error) {
	imageInspect, err := cli.ImageInspect(ctx, pkg.Image)
	if err != nil {
		return false, "", err
	}

	newPkg, err := NewPackageFromImage(pkg.Image, *imageInspect)
	if err != nil {
		return false, "", err
	}

	if newPkg.WorkingDir == "" {
		newPkg.WorkingDir = DefaultWorkingDir
	}

	reporter := NewDiffReporter()

	equal := cmp.Equal(newPkg, pkg, cmp.Reporter(reporter))

	return !equal, reporter.String(), nil
}
