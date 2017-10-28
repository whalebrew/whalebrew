package packages

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/stretchr/testify/assert"
)

var shebang = "#!/usr/bin/env whalebrew\n"

func init() {
	// for Windows
	if runtime.GOOS == "windows" {
		shebang = ":: |\r\n  @echo off\r\n  whalebrew run %~f0 %*\r\n  exit /b %errorlevel%\r\n"
	}
}

func TestMakePackagePath(t *testing.T) {
	installPath, err := ioutil.TempDir("", "whalebrewtest")
	assert.Nil(t, err)
	pkgName := "testpkg"
	truePkgPath := filepath.Join(installPath, pkgName)
	if runtime.GOOS == "windows" {
		// if on Windows, file is batch file
		truePkgPath = truePkgPath + ".bat"
	}

	assert.Equal(t, MakePackagePath(installPath, pkgName), truePkgPath)
	pm := NewPackageManager(installPath)
	assert.Equal(t, pm.MakePackagePath(pkgName), truePkgPath)
}

func TestPackageManagerInstall(t *testing.T) {
	installPath, err := ioutil.TempDir("", "whalebrewtest")
	assert.Nil(t, err)
	pm := NewPackageManager(installPath)

	pkg, err := NewPackageFromImage("whalebrew/whalesay", types.ImageInspect{})
	assert.Nil(t, err)
	err = pm.Install(pkg)
	assert.Nil(t, err)
	packagePath := MakePackagePath(installPath, "whalesay")
	contents, err := ioutil.ReadFile(packagePath)
	assert.Nil(t, err)
	assert.Equal(t, strings.TrimSpace(string(contents)), shebang+"image: whalebrew/whalesay")
	fi, err := os.Stat(packagePath)
	assert.Nil(t, err)
	// if not on Windows, check permission
	if runtime.GOOS != "windows" {
		assert.Equal(t, int(fi.Mode()), 0755)
	}

	// custom install path
	pkg, err = NewPackageFromImage("whalebrew/whalesay", types.ImageInspect{})
	assert.Nil(t, err)
	pkg.Name = "whalesay2"
	err = pm.Install(pkg)
	assert.Nil(t, err)
	packagePath = MakePackagePath(installPath, "whalesay2")
	contents, err = ioutil.ReadFile(packagePath)
	assert.Nil(t, err)
	assert.Equal(t, strings.TrimSpace(string(contents)), shebang+"image: whalebrew/whalesay")

	// file already exists
	packagePath = MakePackagePath(installPath, "alreadyexists")
	err = ioutil.WriteFile(packagePath, []byte("not a whalebrew package"), 0755)
	assert.Nil(t, err)
	pkg, err = NewPackageFromImage("whalebrew/whalesay", types.ImageInspect{})
	assert.Nil(t, err)
	pkg.Name = "alreadyexists"
	err = pm.Install(pkg)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestPackageManagerForceInstall(t *testing.T) {
	installPath, err := ioutil.TempDir("", "whalebrewtest")
	assert.Nil(t, err)
	pm := NewPackageManager(installPath)

	pkg, err := NewPackageFromImage("whalebrew/whalesay", types.ImageInspect{})
	assert.Nil(t, err)
	err = pm.ForceInstall(pkg)
	assert.Nil(t, err)
	packagePath := MakePackagePath(installPath, "whalesay")
	contents, err := ioutil.ReadFile(packagePath)
	assert.Nil(t, err)
	assert.Equal(t, strings.TrimSpace(string(contents)), shebang+"image: whalebrew/whalesay")
	fi, err := os.Stat(packagePath)
	assert.Nil(t, err)
	// if not on Windows, check permission
	if runtime.GOOS != "windows" {
		assert.Equal(t, int(fi.Mode()), 0755)
	}

	// custom install path
	pkg, err = NewPackageFromImage("whalebrew/whalesay", types.ImageInspect{})
	assert.Nil(t, err)
	pkg.Name = "whalesay2"
	err = pm.ForceInstall(pkg)
	assert.Nil(t, err)
	packagePath = MakePackagePath(installPath, "whalesay2")
	contents, err = ioutil.ReadFile(packagePath)
	assert.Nil(t, err)
	assert.Equal(t, strings.TrimSpace(string(contents)), shebang+"image: whalebrew/whalesay")

	// file already exists
	err = ioutil.WriteFile(MakePackagePath(installPath, "alreadyexists"), []byte("not a whalebrew package"), 0755)
	assert.Nil(t, err)
	pkg, err = NewPackageFromImage("whalebrew/whalesay", types.ImageInspect{})
	assert.Nil(t, err)
	pkg.Name = "alreadyexists"
	err = pm.ForceInstall(pkg)
	assert.Nil(t, err)
	packagePath = MakePackagePath(installPath, "alreadyexists")
	contents, err = ioutil.ReadFile(packagePath)
	assert.Nil(t, err)
	assert.Equal(t, strings.TrimSpace(string(contents)), shebang+"image: whalebrew/whalesay")
}

func TestPackageManagerList(t *testing.T) {
	installPath, err := ioutil.TempDir("", "whalebrewtest")
	assert.Nil(t, err)
	_, err = ioutil.TempDir(installPath, "sample-folder")
	assert.Nil(t, err)

	// file which isn't a package
	err = ioutil.WriteFile(MakePackagePath(installPath, "notapackage"), []byte("not a whalebrew package"), 0755)
	assert.Nil(t, err)

	// no permissions to read file
	err = ioutil.WriteFile(MakePackagePath(installPath, "nopermissions"), []byte("blah blah blah"), 0000)
	assert.Nil(t, err)

	// dead symlink
	if runtime.GOOS != "windows" {
		err = os.Symlink("/doesnotexist", MakePackagePath(installPath, "deadsymlink"))
		assert.Nil(t, err)
	}

	pm := NewPackageManager(installPath)
	pkg, err := NewPackageFromImage("whalebrew/whalesay", types.ImageInspect{})
	assert.Nil(t, err)
	err = pm.Install(pkg)
	assert.Nil(t, err)
	packages, err := pm.List()
	assert.Nil(t, err)
	assert.Equal(t, len(packages), 1)
	assert.Equal(t, packages["whalesay"].Image, "whalebrew/whalesay")
}

func TestPackageManagerUninstall(t *testing.T) {
	installPath, err := ioutil.TempDir("", "whalebrewtest")
	assert.Nil(t, err)
	pm := NewPackageManager(installPath)

	pkg, err := NewPackageFromImage("whalebrew/whalesay", types.ImageInspect{})
	assert.Nil(t, err)
	err = pm.Install(pkg)
	assert.Nil(t, err)
	packagePath := MakePackagePath(installPath, "whalesay")
	_, err = os.Stat(packagePath)
	assert.Nil(t, err)
	err = pm.Uninstall("whalesay")
	assert.Nil(t, err)

	packagePath = MakePackagePath(installPath, "notapackage")
	err = ioutil.WriteFile(packagePath, []byte("not a whalebrew package"), 0755)
	assert.Nil(t, err)
	err = pm.Uninstall("notapackage")
	assert.Contains(t, err.Error(), packagePath+" is not a Whalebrew package")
}

func TestIsPackage(t *testing.T) {
	dir, err := ioutil.TempDir("", "whalebrewtest")
	assert.Nil(t, err)

	packagePath := MakePackagePath(dir, "onebyte")
	err = ioutil.WriteFile(packagePath, []byte("!"), 0755)
	assert.Nil(t, err)
	isPackage, err := IsPackage(packagePath)
	assert.Nil(t, err)
	assert.False(t, isPackage)

	packagePath = MakePackagePath(dir, "notpackage")
	err = ioutil.WriteFile(packagePath, []byte("not a package"), 0755)
	assert.Nil(t, err)
	isPackage, err = IsPackage(packagePath)
	assert.Nil(t, err)
	assert.False(t, isPackage)

	packagePath = MakePackagePath(dir, "workingpackage")
	err = ioutil.WriteFile(packagePath, []byte(shebang+"image: something"), 0755)
	assert.Nil(t, err)
	isPackage, err = IsPackage(packagePath)
	assert.Nil(t, err)
	assert.True(t, isPackage)
}
