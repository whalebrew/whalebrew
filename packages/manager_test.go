package packages

import (
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"
)

func TestPackageManagerInstall(t *testing.T) {
	installPath, err := ioutil.TempDir("", "whalebrewtest")
	assert.Nil(t, err)
	pm := NewPackageManager(installPath)

	pkg, err := NewPackageFromImageName("whalebrew/whalesay", types.ImageInspect{})
	assert.Nil(t, err)
	err = pm.Install(pkg)
	assert.Nil(t, err)
	packagePath := path.Join(installPath, "whalesay")
	contents, err := ioutil.ReadFile(packagePath)
	assert.Nil(t, err)
	assert.Equal(t, strings.TrimSpace(string(contents)), "#!/usr/bin/env whalebrew\nimage: whalebrew/whalesay")
	fi, err := os.Stat(packagePath)
	assert.Nil(t, err)
	assert.Equal(t, int(fi.Mode()), 0755)

	// custom install path
	pkg, err = NewPackageFromImageName("whalebrew/whalesay", types.ImageInspect{})
	assert.Nil(t, err)
	pkg.Name = "whalesay2"
	err = pm.Install(pkg)
	assert.Nil(t, err)
	packagePath = path.Join(installPath, "whalesay2")
	contents, err = ioutil.ReadFile(packagePath)
	assert.Nil(t, err)
	assert.Equal(t, strings.TrimSpace(string(contents)), "#!/usr/bin/env whalebrew\nimage: whalebrew/whalesay")

	// file already exists
	err = ioutil.WriteFile(path.Join(installPath, "alreadyexists"), []byte("not a whalebrew package"), 0755)
	assert.Nil(t, err)
	pkg, err = NewPackageFromImageName("whalebrew/whalesay", types.ImageInspect{})
	assert.Nil(t, err)
	pkg.Name = "alreadyexists"
	err = pm.Install(pkg)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "already exists")

	// with tag
	pkg, err = NewPackageFromImageName("whalebrew/foo:bar", types.ImageInspect{})
	assert.Nil(t, err)
	assert.Equal(t, pkg.Name, "foo")
	assert.Equal(t, pkg.Image, "whalebrew/foo:bar")

	// test labels
	pkg, err = NewPackageFromImageName("whalebrew/whalesay", types.ImageInspect{
		ContainerConfig: &container.Config{
			Labels: map[string]string{
				"io.whalebrew.name":               "ws",
				"io.whalebrew.config.environment": "[\"SOME_CONFIG_OPTION\"]",
				"io.whalebrew.config.volumes":     "[\"/somesource:/somedest\"]",
			},
		},
	})
	assert.Nil(t, err)
	assert.Equal(t, pkg.Name, "ws")
	assert.Equal(t, pkg.Image, "whalebrew/whalesay")
	assert.Equal(t, pkg.Environment, []string{"SOME_CONFIG_OPTION"})
	assert.Equal(t, pkg.Volumes, []string{"/somesource:/somedest"})

}

func TestPackageManagerList(t *testing.T) {
	installPath, err := ioutil.TempDir("", "whalebrewtest")
	assert.Nil(t, err)
	err = ioutil.WriteFile(path.Join(installPath, "notapackage"), []byte("not a whalebrew package"), 0755)
	assert.Nil(t, err)
	pm := NewPackageManager(installPath)
	pkg, err := NewPackageFromImageName("whalebrew/whalesay", types.ImageInspect{})
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

	pkg, err := NewPackageFromImageName("whalebrew/whalesay", types.ImageInspect{})
	assert.Nil(t, err)
	err = pm.Install(pkg)
	assert.Nil(t, err)
	_, err = os.Stat(path.Join(installPath, "whalesay"))
	assert.Nil(t, err)
	err = pm.Uninstall("whalesay")
	assert.Nil(t, err)

	err = ioutil.WriteFile(path.Join(installPath, "notapackage"), []byte("not a whalebrew package"), 0755)
	assert.Nil(t, err)
	err = pm.Uninstall("notapackage")
	assert.Contains(t, err.Error(), "/notapackage is not a Whalebrew package")
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
