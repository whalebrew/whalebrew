package packages

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"gopkg.in/yaml.v2"
)

// PackageManager manages packages at a given path
type PackageManager struct {
	InstallPath string
}

// Package represents a Whalebrew package
type Package struct {
	Name        string   `yaml:"-"`
	Environment []string `yaml:"environment,omitempty"`
	Image       string   `yaml:"image"`
	Volumes     []string `yaml:"volumes,omitempty"`
	Ports       []string `yaml:"ports,omitempty"`
}

// NewPackageFromImageName creates a package from a given image name,
// inspecting the image to fetch the package configuration
func NewPackageFromImageName(image string, imageInspect types.ImageInspect) (*Package, error) {
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

// ImageInspect inspects the image associated with this package
func (pkg *Package) ImageInspect() (*types.ImageInspect, error) {
	cli, err := client.NewEnvClient()
	if err != nil {
		return nil, err
	}
	img, _, err := cli.ImageInspectWithRaw(context.Background(), pkg.Image)
	return &img, err
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

// NewPackageManager creates a new PackageManager
func NewPackageManager(path string) *PackageManager {
	return &PackageManager{InstallPath: path}
}

// Install installs a package
func (pm *PackageManager) Install(pkg *Package) error {
	d, err := yaml.Marshal(&pkg)
	if err != nil {
		return err
	}

	packagePath := path.Join(pm.InstallPath, pkg.Name)

	if _, err := os.Stat(packagePath); err == nil {
		return fmt.Errorf("'%s' already exists", packagePath)
	}

	d = append([]byte("#!/usr/bin/env whalebrew\n"), d...)
	return ioutil.WriteFile(packagePath, d, 0755)
}

// List lists installed packages
func (pm *PackageManager) List() (map[string]*Package, error) {
	packages := make(map[string]*Package)
	files, err := ioutil.ReadDir(pm.InstallPath)
	if err != nil {
		return packages, err
	}
	for _, file := range files {
		isPackage, err := IsPackage(path.Join(pm.InstallPath, file.Name()))
		if err != nil {
			return packages, err
		}
		if isPackage {
			pkg, err := pm.Load(file.Name())
			if err != nil {
				return packages, err
			}
			packages[file.Name()] = pkg
		}
	}
	return packages, nil
}

// Load returns an installed package given its package name
func (pm *PackageManager) Load(name string) (*Package, error) {
	return LoadPackageFromPath(path.Join(pm.InstallPath, name))
}

// Uninstall uninstalls a package
func (pm *PackageManager) Uninstall(packageName string) error {
	p := path.Join(pm.InstallPath, packageName)
	isPackage, err := IsPackage(p)
	if err != nil {
		return err
	}
	if !isPackage {
		return fmt.Errorf("%s is not a Whalebrew package", p)
	}
	return os.Remove(p)
}

// IsPackage returns true if the given path is a whalebrew package
func IsPackage(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		// dead symlink
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	defer f.Close()

	info, err := f.Stat()

	if err != nil {
		return false, err
	}

	if info.IsDir() {
		return false, nil
	}

	reader := bufio.NewReader(f)
	firstTwoBytes := make([]byte, 2)
	_, err = reader.Read(firstTwoBytes)

	if err == io.EOF {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	if string(firstTwoBytes) != "#!" {
		return false, nil
	}

	line, _, err := reader.ReadLine()

	if err == io.EOF {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if strings.HasPrefix(string(line), "/usr/bin/env whalebrew") {
		return true, nil
	}

	return false, nil
}
