package packages

import (
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"strings"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/stretchr/testify/assert"
)

func TestPackageManagerInstall(t *testing.T) {
	installPath, err := ioutil.TempDir("", "whalebrewtest")
	assert.Nil(t, err)
	pm := NewPackageManager(installPath)

	pkg, err := NewPackageFromImage("whalebrew/whalesay", types.ImageInspect{})
	assert.Nil(t, err)
	err = pm.Install(pkg)
	assert.Nil(t, err)
	packagePath := path.Join(installPath, "whalesay")
	if runtime.GOOS == "windows" {
		packagePath += ".cmd"
	}
	contents, err := ioutil.ReadFile(packagePath)
	assert.Nil(t, err)
	var expectedContents string
	if runtime.GOOS == "windows" {
		expectedContents = "@echo off\r\nwhalebrew %~f0 %*\r\nexit\r\nimage: whalebrew/whalesay"
	} else {
		expectedContents = "#!/usr/bin/env whalebrew\nimage: whalebrew/whalesay"
	}
	assert.Equal(t, expectedContents, strings.TrimSpace(string(contents)))
	fi, err := os.Stat(packagePath)
	assert.Nil(t, err)
	var expectedFilemode int
	if runtime.GOOS == "windows" {
		expectedFilemode = 0666
	} else {
		expectedFilemode = 0755
	}
	assert.Equal(t, expectedFilemode, int(fi.Mode()))

	// custom install path
	pkg, err = NewPackageFromImage("whalebrew/whalesay", types.ImageInspect{})
	assert.Nil(t, err)
	pkg.Name = "whalesay2"
	err = pm.Install(pkg)
	assert.Nil(t, err)
	packagePath = path.Join(installPath, "whalesay2")
	if runtime.GOOS == "windows" {
		packagePath += ".cmd"
	}
	contents, err = ioutil.ReadFile(packagePath)
	assert.Nil(t, err)
	assert.Equal(t, expectedContents, strings.TrimSpace(string(contents)))

	// file already exists
	fileAlreadyExistsPath := path.Join(installPath, "alreadyexists")
	if runtime.GOOS == "windows" {
		fileAlreadyExistsPath += ".cmd"
	}
	err = ioutil.WriteFile(fileAlreadyExistsPath, []byte("not a whalebrew package"), 0755)
	assert.Nil(t, err)
	pkg, err = NewPackageFromImage("whalebrew/whalesay", types.ImageInspect{})
	assert.Nil(t, err)
	pkg.Name = "alreadyexists"
	err = pm.Install(pkg)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "already exists")

}

func TestPackageManagerList(t *testing.T) {
	installPath, err := ioutil.TempDir("", "whalebrewtest")
	assert.Nil(t, err)
	_, err = ioutil.TempDir(installPath, "sample-folder")
	assert.Nil(t, err)

	// file which isn't a package
	err = ioutil.WriteFile(path.Join(installPath, "notapackage"), []byte("not a whalebrew package"), 0755)
	assert.Nil(t, err)

	// no permissions to read file
	err = ioutil.WriteFile(path.Join(installPath, "nopermissions"), []byte("blah blah blah"), 0000)
	assert.Nil(t, err)

	// dead symlink
	if runtime.GOOS != "windows" {
		err = os.Symlink("/doesnotexist", path.Join(installPath, "deadsymlink"))
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
	fileName := "whalesay"
	if runtime.GOOS == "windows" {
		fileName += ".cmd"
	}
	assert.Equal(t, "whalebrew/whalesay", packages[fileName].Image)
}

func TestPackageManagerUninstall(t *testing.T) {
	installPath, err := ioutil.TempDir("", "whalebrewtest")
	assert.Nil(t, err)
	pm := NewPackageManager(installPath)

	pkg, err := NewPackageFromImage("whalebrew/whalesay", types.ImageInspect{})
	assert.Nil(t, err)
	err = pm.Install(pkg)
	assert.Nil(t, err)
	fileName := "whalesay"
	if runtime.GOOS == "windows" {
		fileName += ".cmd"
	}
	_, err = os.Stat(path.Join(installPath, fileName))
	assert.Nil(t, err)
	err = pm.Uninstall("whalesay")
	assert.Nil(t, err)

	notAPackage := "notapackage"
	if runtime.GOOS == "windows" {
		notAPackage += ".cmd"
	}
	err = ioutil.WriteFile(path.Join(installPath, notAPackage), []byte("not a whalebrew package"), 0755)
	assert.Nil(t, err)
	err = pm.Uninstall("notapackage")
	assert.Contains(t, err.Error(), notAPackage+" is not a Whalebrew package")
}

func TestIsPackage(t *testing.T) {
	dir, err := ioutil.TempDir("", "whalebrewtest")
	assert.Nil(t, err)

	err = ioutil.WriteFile(path.Join(dir, "onebyte"), []byte("!"), 0755)
	assert.Nil(t, err)
	isPackage, err := IsPackage(path.Join(dir, "onebyte"))
	assert.Nil(t, err)
	assert.False(t, isPackage)

	err = ioutil.WriteFile(path.Join(dir, "notpackage"), []byte("not a package"), 0755)
	assert.Nil(t, err)
	isPackage, err = IsPackage(path.Join(dir, "notpackage"))
	assert.Nil(t, err)
	assert.False(t, isPackage)

	err = ioutil.WriteFile(path.Join(dir, "workingpackage"), []byte("#!/usr/bin/env whalebrew\nimage: something"), 0755)
	assert.Nil(t, err)
	isPackage, err = IsPackage(path.Join(dir, "workingpackage"))
	assert.Nil(t, err)
	assert.True(t, isPackage)
}
