package packages

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/google/go-cmp/cmp"
	"github.com/whalebrew/whalebrew/client"
	"github.com/whalebrew/whalebrew/version"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"gopkg.in/yaml.v2"
)

const (
	DefaultWorkingDir = "/workdir"
	labelPrefix       = "io.whalebrew."
)

var (
	DefaultLoader = YamlLoader{}
)

// Package represents a Whalebrew package
type Package struct {
	Name                string   `yaml:"-" labels:"name"`
	Entrypoint          []string `yaml:"entrypoint,omitempty"`
	Environment         []string `yaml:"environment,omitempty" labels:"config.environment"`
	Image               string   `yaml:"image"`
	Volumes             []string `yaml:"volumes,omitempty" labels:"config.volumes"`
	Ports               []string `yaml:"ports,omitempty" labels:"config.ports"`
	Networks            []string `yaml:"networks,omitempty" labels:"config.networks"`
	WorkingDir          string   `yaml:"working_dir,omitempty" labels:"config.working_dir"`
	KeepContainerUser   bool     `yaml:"keep_container_user,omitempty" labels:"config.keep_container_user"`
	SkipMissingVolumes  bool     `yaml:"skip_missing_volumes,omitempty"`
	MountMissingVolumes bool     `yaml:"mount_missing_volumes,omitempty"`
	RequiredVersion     string   `yaml:"required_version,omitempty" labels:"required_version"`
	PathArguments       []string `yaml:"path_arguments,omitempty" labels:"config.volumes_from_args"`
}

type StrictError interface {
	Strict() bool
}

type LabelError struct {
	Err   error
	Label string
}

func (e LabelError) Error() string {
	return fmt.Sprintf("decoding label %s: %v", e.Label, e.Err)
}

func (e LabelError) Strict() bool {
	if strictError, ok := e.Err.(StrictError); ok {
		return strictError.Strict()
	}
	return true
}

type DecodeLabelError struct {
	Err   error
	Value string
}

func (e DecodeLabelError) Error() string {
	return fmt.Sprintf("failed to decode value %v: %v", e.Value, e.Err)
}

func (e DecodeLabelError) Strict() bool {
	return true
}

type UnknownLabelError struct {
	Label string
}

func (e UnknownLabelError) Error() string {
	return fmt.Sprintf("unknwon label %s", e.Label)
}

func (e UnknownLabelError) Strict() bool {
	return false
}

type NoEntrypointError struct {
}

func (e NoEntrypointError) Error() string {
	return `missing entrypoint in docker image. consider re-building using ENTRYPOINT ["/path/to/your/binary"]`
}

func (e NoEntrypointError) Strict() bool {
	return true
}

// Loader loads a package from a given path
type Loader interface {
	LoadPackageFromPath(path string) (*Package, error)
}

func decodeLabel(value string, dest interface{}) error {
	err := yaml.Unmarshal([]byte(value), dest)
	if err != nil {
		switch dest.(type) {
		// this is used when decoding plain strings that may not be valid yaml.
		// Specially required versions may be interpreted in yaml as an object
		// Whereas we expect it to be a plain string
		case *string:
			d := dest.(*string)
			*d = value
			err = nil
			return nil
		}
	}
	if err != nil {
		return DecodeLabelError{
			Err:   err,
			Value: value,
		}
	}
	return nil
}

func loadImageLabel(imageInspect types.ImageInspect, label string, dest interface{}) error {
	label = labelPrefix + label
	// In the previous behaviour we were reading from ContainerConfig only.
	// When building images with buildkit, it seems that those fields are not set any longer.
	// Make the transition smoother by using a fallback to the Config field when not found in ContainerConfig.
	// We should make a deeper analysis of the meaning of those 3 fields, the consequences to go for only one,
	// eventually notice about the deprecation and finally rmove it.
	for _, config := range []*container.Config{imageInspect.ContainerConfig, imageInspect.Config} {
		if config != nil && config.Labels != nil {
			if val, ok := config.Labels[label]; ok {
				return decodeLabel(val, dest)
			}
		}
	}
	return nil
}

func LintImage(imageInspect types.ImageInspect, reportError func(error)) {
	if imageInspect.Config == nil || len(imageInspect.Config.Entrypoint) == 0 {
		reportError(NoEntrypointError{})
	}
	for _, config := range []*container.Config{imageInspect.ContainerConfig, imageInspect.Config} {
		if config != nil && config.Labels != nil {
			for originalLabel, value := range config.Labels {
				if strings.HasPrefix(originalLabel, labelPrefix) {
					label := strings.TrimPrefix(originalLabel, labelPrefix)
					pkg := reflect.ValueOf(&Package{}).Elem()
					found := false
					for i := 0; i < pkg.NumField(); i++ {
						if pkg.Type().Field(i).Tag.Get("labels") == label {
							found = true
							err := decodeLabel(value, pkg.Field(i).Addr().Interface())
							if err != nil {
								reportError(LabelError{Err: err, Label: originalLabel})
							}
						}
					}
					if !found && label == "config.missing_volumes" {
						found = true
						switch value {
						case "", "error", "skip", "mount":
							return
						default:
							reportError(LabelError{Err: errors.New("missing volumes strategy must be one of error, skip or mount"), Label: originalLabel})
						}
					}
					if !found {
						reportError(UnknownLabelError{originalLabel})
					}
				}
			}
		}
	}
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

	v := reflect.ValueOf(pkg).Elem()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		label := v.Type().Field(i).Tag.Get("labels")
		if label != "" {
			if err := loadImageLabel(imageInspect, label, field.Addr().Interface()); err != nil {
				return nil, err
			}
		}
	}
	missingVolumes := ""
	if err := loadImageLabel(imageInspect, "config.missing_volumes", &missingVolumes); err != nil {
		return nil, err
	}

	if err := version.CheckCompatible(pkg.RequiredVersion); pkg.RequiredVersion != "" && err != nil {
		return nil, err
	}
	for i, arg := range pkg.PathArguments {
		pkg.PathArguments[i] = strings.TrimLeft(arg, "-")
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
