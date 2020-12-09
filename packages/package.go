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
	"github.com/docker/docker/api/types/container"
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
}

// Loader loads a package from a given path
type Loader interface {
	LoadPackageFromPath(path string) (*Package, error)
}

func loadImageLabel(imageInspect types.ImageInspect, label string, dest interface{}) error {
	label = "io.whalebrew." + label
	// In the previous behaviour we were reading from ContainerConfig only.
	// When building images with buildkit, it seems that those fields are not set any longer.
	// Make the transition smoother by using a fallback to the Config field when not found in ContainerConfig.
	// We should make a deeper analysis of the meaning of those 3 fields, the consequences to go for only one,
	// eventually notice about the deprecation and finally rmove it.
	for _, config := range []*container.Config{imageInspect.ContainerConfig, imageInspect.Config} {
		if config != nil && config.Labels != nil {
			if val, ok := config.Labels[label]; ok {
				err := yaml.Unmarshal([]byte(val), dest)
				if err != nil {
					switch dest.(type) {
					// this is used when decoding plain strings that may not be valid yaml.
					// Specially required versions may be interpreted in yaml as an object
					// Whereas we expect it to be a plain string
					case *string:
						d := dest.(*string)
						*d = val
						err = nil
						return nil
					}
				}
				return err
			}
		}
	}
	return nil
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
	missingVolumes := "error"
	args := []string{}
	if err := loadImageLabel(imageInspect, "name", &pkg.Name); err != nil {
		return nil, err
	}
	if err := loadImageLabel(imageInspect, "required_version", &pkg.RequiredVersion); err != nil {
		return nil, err
	}
	if err := loadImageLabel(imageInspect, "config.working_dir", &pkg.WorkingDir); err != nil {
		return nil, err
	}
	if err := loadImageLabel(imageInspect, "config.environment", &pkg.Environment); err != nil {
		return nil, err
	}
	if err := loadImageLabel(imageInspect, "config.volumes", &pkg.Volumes); err != nil {
		return nil, err
	}
	if err := loadImageLabel(imageInspect, "config.volumes_from_args", &args); err != nil {
		return nil, err
	}
	if err := loadImageLabel(imageInspect, "config.ports", &pkg.Ports); err != nil {
		return nil, err
	}
	if err := loadImageLabel(imageInspect, "config.networks", &pkg.Networks); err != nil {
		return nil, err
	}
	if err := loadImageLabel(imageInspect, "config.keep_container_user", &pkg.KeepContainerUser); err != nil {
		return nil, err
	}
	if err := loadImageLabel(imageInspect, "config.missing_volumes", &missingVolumes); err != nil {
		return nil, err
	}

	if err := version.CheckCompatible(pkg.RequiredVersion); pkg.RequiredVersion != "" && err != nil {
		return nil, err
	}
	for _, arg := range args {
		pkg.PathArguments = append(pkg.PathArguments, strings.TrimLeft(arg, "-"))
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
