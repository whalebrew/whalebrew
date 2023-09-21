package packages

import (
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"

	imagev1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
)

func TestPackageManagerInstall(t *testing.T) {
	installPath, err := ioutil.TempDir("", "whalebrewtest")
	assert.Nil(t, err)
	pm := NewPackageManager(installPath)

	pkg, err := NewPackageFromImage("whalebrew/whalesay", &imagev1.Image{})
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
	pkg, err = NewPackageFromImage("whalebrew/whalesay", &imagev1.Image{})
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
	pkg, err = NewPackageFromImage("whalebrew/whalesay", &imagev1.Image{})
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

	pkg, err := NewPackageFromImage("whalebrew/whalesay", &imagev1.Image{})
	assert.Nil(t, err)
	err = pm.ForceInstall(pkg)
	assert.Nil(t, err)
	packagePath := path.Join(installPath, "whalesay")
	contents, err := ioutil.ReadFile(packagePath)
	assert.Nil(t, err)
	assert.Equal(t, strings.TrimSpace(string(contents)), "#!/usr/bin/env whalebrew\nimage: whalebrew/whalesay")
	fi, err := os.Stat(packagePath)
	assert.Nil(t, err)
	assert.Equal(t, int(fi.Mode()), 0755)

	// custom install path
	pkg, err = NewPackageFromImage("whalebrew/whalesay", &imagev1.Image{})
	assert.Nil(t, err)
	pkg.Name = "whalesay2"
	err = pm.ForceInstall(pkg)
	assert.Nil(t, err)
	packagePath = path.Join(installPath, "whalesay2")
	contents, err = ioutil.ReadFile(packagePath)
	assert.Nil(t, err)
	assert.Equal(t, strings.TrimSpace(string(contents)), "#!/usr/bin/env whalebrew\nimage: whalebrew/whalesay")

	// file already exists
	err = ioutil.WriteFile(path.Join(installPath, "alreadyexists"), []byte("not a whalebrew package"), 0755)
	assert.Nil(t, err)
	pkg, err = NewPackageFromImage("whalebrew/whalesay", &imagev1.Image{})
	assert.Nil(t, err)
	pkg.Name = "alreadyexists"
	err = pm.ForceInstall(pkg)
	assert.Nil(t, err)
	packagePath = path.Join(installPath, "alreadyexists")
	contents, err = ioutil.ReadFile(packagePath)
	assert.Nil(t, err)
	assert.Equal(t, strings.TrimSpace(string(contents)), "#!/usr/bin/env whalebrew\nimage: whalebrew/whalesay")
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
	err = os.Symlink("/doesnotexist", path.Join(installPath, "deadsymlink"))
	assert.Nil(t, err)

	pm := NewPackageManager(installPath)
	pkg, err := NewPackageFromImage("whalebrew/whalesay", &imagev1.Image{})
	assert.Nil(t, err)
	err = pm.Install(pkg)
	assert.Nil(t, err)
	packages, err := pm.List()
	assert.Nil(t, err)
	assert.Equal(t, len(packages), 1)
	assert.Equal(t, packages["whalesay"].Image, "whalebrew/whalesay")
}

func TestPackageManagerFindByNameOrImage(t *testing.T) {
	installPath, err := ioutil.TempDir("", "whalebrewtest")
	assert.Nil(t, err)

	pm := NewPackageManager(installPath)
	pkg, err := NewPackageFromImage("whalebrew/whalesay", &imagev1.Image{})
	assert.Nil(t, err)
	pkg.Name = "some-whalesay"
	err = pm.Install(pkg)
	assert.Nil(t, err)

	pkg, err = NewPackageFromImage("whalebrew/whalesay", &imagev1.Image{})
	assert.Nil(t, err)
	pkg.Name = "some-other-whalesay"
	err = pm.Install(pkg)
	assert.Nil(t, err)

	t.Run("when no package matches", func(t *testing.T) {
		candidates, err := pm.FindByNameOrImage("whalesay")
		assert.Nil(t, err)
		assert.Empty(t, candidates)
	})

	t.Run("when searching with matching image name", func(t *testing.T) {
		candidates, err := pm.FindByNameOrImage("whalebrew/whalesay")
		assert.Nil(t, err)
		assert.Len(t, candidates, 2)
		assert.Contains(t, candidates, MatchingPackage{
			Package: Package{
				Image:      "whalebrew/whalesay",
				WorkingDir: DefaultWorkingDir,
				Name:       "some-whalesay",
			},
			Reason: MatchReasonPackageImageMatches,
		})
		assert.Contains(t, candidates, MatchingPackage{
			Package: Package{
				Image:      "whalebrew/whalesay",
				WorkingDir: DefaultWorkingDir,
				Name:       "some-other-whalesay",
			},
			Reason: MatchReasonPackageImageMatches,
		})
	})

	t.Run("when searching with matching package name", func(t *testing.T) {
		candidates, err := pm.FindByNameOrImage("some-whalesay")
		assert.Nil(t, err)
		assert.Len(t, candidates, 1)
		assert.Contains(t, candidates, MatchingPackage{
			Package: Package{
				Image:      "whalebrew/whalesay",
				WorkingDir: DefaultWorkingDir,
				Name:       "some-whalesay",
			},
			Reason: MatchReasonPackageNameMatches,
		})
	})
}

func TestPackageManagerUninstall(t *testing.T) {
	installPath, err := ioutil.TempDir("", "whalebrewtest")
	assert.Nil(t, err)
	pm := NewPackageManager(installPath)

	pkg, err := NewPackageFromImage("whalebrew/whalesay", &imagev1.Image{})
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
