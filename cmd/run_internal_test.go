package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/whalebrew/whalebrew/packages"
)

func TestShouldBind(t *testing.T) {
	wd, err := os.Getwd()
	assert.NoError(t, err)
	t.Run("with a file that exists", func(t *testing.T) {
		t.Run("when not skipping missing volumes", func(t *testing.T) {
			bind, err := shouldBind(filepath.Join(wd, "run.go"), &packages.Package{SkipMissingVolumes: false})
			assert.NoError(t, err)
			assert.True(t, bind)
		})
		t.Run("when skipping missing volumes", func(t *testing.T) {
			bind, err := shouldBind(filepath.Join(wd, "run.go"), &packages.Package{SkipMissingVolumes: true})
			assert.NoError(t, err)
			assert.True(t, bind)
		})
	})
	t.Run("with a file that does not exists", func(t *testing.T) {
		t.Run("when not skipping missing volumes", func(t *testing.T) {
			bind, err := shouldBind(filepath.Join(wd, "thisFileShouldNotExist.go"), &packages.Package{SkipMissingVolumes: false})
			assert.Error(t, err)
			assert.False(t, bind)
		})
		t.Run("when skipping missing volumes", func(t *testing.T) {
			bind, err := shouldBind(filepath.Join(wd, "thisFileShouldNotExist.go"), &packages.Package{SkipMissingVolumes: true})
			assert.NoError(t, err)
			assert.False(t, bind)
		})
		t.Run("when mounting missing volumes", func(t *testing.T) {
			bind, err := shouldBind(filepath.Join(wd, "thisFileShouldNotExist.go"), &packages.Package{MountMissingVolumes: true})
			assert.NoError(t, err)
			assert.True(t, bind)
		})
	})
}

func TestAppendVolumes(t *testing.T) {
	wd, err := os.Getwd()
	assert.NoError(t, err)
	volumes := []string{
		filepath.Join(wd, "thisFileShouldNotExist.go:/notExists"),
		filepath.Join(wd, "run.go:/exists"),
		"~/:/home/user",
	}
	t.Run("when non existing volumes should be skept", func(t *testing.T) {
		volumes, err := getVolumes(&packages.Package{SkipMissingVolumes: true, Volumes: volumes})
		assert.NoError(t, err)
		assert.NotNil(t, volumes)
		assert.Len(t, volumes, 3)
		t.Run("all existing volumes instructed to mount on command line", func(t *testing.T) {
			assert.Equal(t, []string{fmt.Sprintf("%s/run.go:/exists", wd)}, volumes[:1])
			if !strings.HasSuffix(volumes[1], "/:/home/user") {
				t.Errorf("home volume should be mounted")
			}
			if strings.HasPrefix(volumes[1], "~") {
				t.Errorf("~/ prefix should be replaced by current user home directory")
			}
		})
	})
	t.Run("when non existing volumes should be mounted", func(t *testing.T) {
		args, err := getVolumes(&packages.Package{MountMissingVolumes: true, Volumes: volumes})
		assert.NoError(t, err)
		assert.NotNil(t, args)
		assert.Len(t, args, 4)
		t.Run("all existing volumes instructed to mount on command line", func(t *testing.T) {
			assert.Equal(t, []string{fmt.Sprintf("%s/thisFileShouldNotExist.go:/notExists", wd), fmt.Sprintf("%s/run.go:/exists", wd)}, args[:2])
			if !strings.HasSuffix(args[2], "/:/home/user") {
				t.Errorf("home volume should be mounted")
			}
			if strings.HasPrefix(args[2], "~") {
				t.Errorf("~/ prefix should be replaced by current user home directory")
			}
		})
	})
	t.Run("when non existing volumes should not be skept", func(t *testing.T) {
		args, err := getVolumes(&packages.Package{SkipMissingVolumes: false, Volumes: volumes})
		assert.Error(t, err)
		assert.Nil(t, args)
	})
}

func TestParseRuntimeVolumes(t *testing.T) {
	pkg := &packages.Package{
		PathArguments: []string{"C", "exec-path", "X", "stream"},
	}

	wd, err := os.Getwd()
	assert.NoError(t, err)
	assert.Equal(
		t,
		[]string{"/tmp:/tmp", "/other/path:/other/path", "/some/path:/some/path", fmt.Sprintf("%s/local:%s/local", wd, wd), "/something:/something"},
		parseRuntimeVolumes([]string{"-C/tmp", "--other", "arg", "--exec-path", "/some/path", "-C", "/other/path", "--exec-path=local", "--stream", "-", "-X", "/dev/stdin", "-X/something"}, pkg),
	)
}
