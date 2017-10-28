package packages

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"gopkg.in/yaml.v2"
)

// PackageManager manages packages at a given path
type PackageManager struct {
	InstallPath string
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

	packagePath := pm.MakePackagePath(pkg.Name)

	if _, err := os.Stat(packagePath); err == nil {
		return fmt.Errorf("'%s' already exists", packagePath)
	}

	if runtime.GOOS == "windows" {
		d = append([]byte(":: |\r\n  @echo off\r\n  whalebrew run %~f0 %*\r\n  exit /b %errorlevel%\r\n"), d...)
		d = bytes.Replace(d, []byte("\r\n"), []byte("\n"), -1)
		d = bytes.Replace(d, []byte("\n"), []byte("\r\n"), -1)
	} else {
		d = append([]byte("#!/usr/bin/env whalebrew\n"), d...)
	}
	return ioutil.WriteFile(packagePath, d, 0755)
}

// ForceInstall installs a package
func (pm *PackageManager) ForceInstall(pkg *Package) error {
	d, err := yaml.Marshal(&pkg)
	if err != nil {
		return err
	}

	packagePath := pm.MakePackagePath(pkg.Name)

	if runtime.GOOS == "windows" {
		d = append([]byte(":: |\r\n  @echo off\r\n  whalebrew run %~f0 %*\r\n  exit /b %errorlevel%\r\n"), d...)
		d = bytes.Replace(d, []byte("\r\n"), []byte("\n"), -1)
		d = bytes.Replace(d, []byte("\n"), []byte("\r\n"), -1)
	} else {
		d = append([]byte("#!/usr/bin/env whalebrew\n"), d...)
	}
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
		isPackage, err := IsPackage(filepath.Join(pm.InstallPath, file.Name()))
		if err != nil {
			// Check for various file errors here rather than in IsPackage so it
			// does not swallow errors when checking individual files.

			// permission denied
			if os.IsPermission(err) {
				continue
			}
			// dead symlink
			if os.IsNotExist(err) {
				continue
			}

			return packages, err
		}
		if isPackage {
			pkgName := file.Name()
			if runtime.GOOS == "windows" {
				re := regexp.MustCompile("\\.bat$")
				pkgName = re.ReplaceAllLiteralString(pkgName, "")
			}

			pkg, err := pm.Load(pkgName)
			if err != nil {
				return packages, err
			}
			packages[pkgName] = pkg
		}
	}
	return packages, nil
}

// Load returns an installed package given its package name
func (pm *PackageManager) Load(name string) (*Package, error) {
	return LoadPackageFromPath(pm.MakePackagePath(name))
}

// Uninstall uninstalls a package
func (pm *PackageManager) Uninstall(packageName string) error {
	p := pm.MakePackagePath(packageName)
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
	if runtime.GOOS == "windows" && !strings.HasSuffix(path, ".bat") {
		return false, nil
	}

	f, err := os.Open(path)
	if err != nil {
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

	if runtime.GOOS == "windows" {
		line, _, err := reader.ReadLine()

		if err == io.EOF {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		if strings.HasPrefix(string(line), ":: |") {
			return true, nil
		}
		return false, nil
	}

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

// MakePackagePath returns package path
func MakePackagePath(dir, name string) string {
	// if on Windows, file is batch file.
	pkgPath := filepath.Join(dir, name)
	if runtime.GOOS == "windows" {
		pkgPath = pkgPath + ".bat"
	}
	return pkgPath
}

// MakePackagePath returns package path
func (pm *PackageManager) MakePackagePath(name string) string {
	return MakePackagePath(pm.InstallPath, name)
}
