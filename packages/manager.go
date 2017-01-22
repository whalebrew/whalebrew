package packages

import (
	"bufio"
	"fmt"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
)

// PackageManager manages packages at a given path
type PackageManager struct {
	InstallPath string
}

type Package struct {
	Image string `yaml:"image"`
}

// NewPackageManager creates a new PackageManager
func NewPackageManager(path string) *PackageManager {
	return &PackageManager{InstallPath: path}
}

// Install installs a package
func (pm *PackageManager) Install(imageName, packageName string) error {
	if !strings.Contains(imageName, "/") {
		imageName = "whalebrew/" + imageName
	}
	packagePath := path.Join(pm.InstallPath, packageName)

	pkg := &Package{Image: imageName}
	d, err := yaml.Marshal(&pkg)
	if err != nil {
		return err
	}

	d = append([]byte("#!/usr/bin/env whalebrew run\n"), d...)
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
	d, err := ioutil.ReadFile(path.Join(pm.InstallPath, name))
	if err != nil {
		return nil, err
	}
	pkg := &Package{}
	if err = yaml.Unmarshal(d, pkg); err != nil {
		return pkg, err
	}
	return pkg, nil
}

// Uninstall uninstalls a package
func (pm *PackageManager) Uninstall(packageName string) error {
	path := path.Join(pm.InstallPath, packageName)
	isPackage, err := IsPackage(path)
	if err != nil {
		return err
	}
	if !isPackage {
		return fmt.Errorf("%s is not a Whalebrew package", path)
	}
	return os.Remove(path)
}

// IsPackage returns true if the given path is a whalebrew package
func IsPackage(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()
	reader := bufio.NewReader(f)
	firstTwoBytes := make([]byte, 2)
	_, err = reader.Read(firstTwoBytes)
	if err == io.EOF {
		return false, nil
	} else if err != nil {
		return false, err
	}
	if string(firstTwoBytes) == "#!" {
		line, _, err := reader.ReadLine()
		if err == io.EOF {
			return false, nil
		} else if err != nil {
			return false, err
		}
		if strings.HasPrefix(string(line), "/usr/bin/env whalebrew run") {
			return true, nil
		}
	}
	return false, nil
}
