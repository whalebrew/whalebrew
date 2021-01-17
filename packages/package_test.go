package packages

import (
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/stretchr/testify/assert"
)

func newTestPkg(label, value string) (*Package, error) {
	return NewPackageFromImage("whalebrew/whalesay", types.ImageInspect{
		ContainerConfig: &container.Config{
			Labels: map[string]string{label: value},
		},
	})
}

func mustNewTestPkg(t *testing.T, label, value string) *Package {
	pkg, err := newTestPkg(label, value)
	assert.NoErrorf(t, err, "creating a package with label '%s' and value '%s' should not raise an error", label, value)
	return pkg
}

func mustNewTestPackageFromImage(t *testing.T, imageName string) *Package {
	pkg, err := NewPackageFromImage(imageName, types.ImageInspect{
		ContainerConfig: &container.Config{},
	})
	assert.NoErrorf(t, err, "creating a package for image '%s' should not raise an error", imageName)
	return pkg
}

func TestLoadImageLabelDecodesYamlList(t *testing.T) {
	value := []string{}
	assert.NoError(
		t,
		loadImageLabel(
			types.ImageInspect{
				ContainerConfig: &container.Config{
					Labels: map[string]string{"io.whalebrew.some.key": "- some\n- other"},
				},
			},
			"some.key",
			&value,
		),
	)
	assert.Equal(t, []string{"some", "other"}, value)
}

func TestLintImageErrorsWhenMissingVolumesStrategyIsUnknown(t *testing.T) {
	errored := false
	LintImage(
		types.ImageInspect{},
		func(e error) {
			errored = true
			assert.Equal(t, NoEntrypointError{}, e)
		},
	)
	assert.True(t, errored)
}

func TestLintImageErrorsWhenImageConfigIsNil(t *testing.T) {
	errored := false
	LintImage(
		types.ImageInspect{
			ContainerConfig: &container.Config{
				Labels: map[string]string{"io.whalebrew.config.missing_volumes": "other"},
			},
			Config: &container.Config{
				Entrypoint: []string{"/entrypoint"},
			},
		},
		func(e error) {
			errored = true
			assert.Contains(t, e.Error(), "one of error, skip or mount")
		},
	)
	assert.True(t, errored)
}

func TestLintImageSucceedsWithSupportedMissingVolumeStrategy(t *testing.T) {
	for _, strategy := range []string{"error", "skip", "mount"} {
		t.Run("with strategy "+strategy, func(t *testing.T) {
			errored := false
			LintImage(
				types.ImageInspect{
					ContainerConfig: &container.Config{
						Labels: map[string]string{"io.whalebrew.config.missing_volumes": strategy},
					},
					Config: &container.Config{
						Entrypoint: []string{"/entrypoint"},
					},
				},
				func(e error) {
					errored = true
					t.Errorf("no error should be raised")
				},
			)
			assert.False(t, errored)
		})
	}
}

func TestLintImageErrorsWhenEntrypointIsMissing(t *testing.T) {
	errored := false
	LintImage(
		types.ImageInspect{
			Config: &container.Config{},
		},
		func(e error) {
			errored = true
			assert.Equal(t, NoEntrypointError{}, e)
		},
	)
	assert.True(t, errored)
}

func TestLintImageErrorsWhenLabelDoesNotExitsFromContainerConfig(t *testing.T) {
	errored := false
	LintImage(
		types.ImageInspect{
			ContainerConfig: &container.Config{
				Labels: map[string]string{"io.whalebrew.some.key": "- some\n- other"},
			},
			Config: &container.Config{
				Entrypoint: []string{"/entrypoint"},
			},
		},
		func(e error) {
			errored = true
			assert.Equal(t, UnknownLabelError{"io.whalebrew.some.key"}, e)
		},
	)
	assert.True(t, errored)
}

func TestLintImageErrorsWhenLabelDoesNotExitsFromImageConfig(t *testing.T) {
	errored := false
	LintImage(
		types.ImageInspect{
			Config: &container.Config{
				Labels:     map[string]string{"io.whalebrew.some.key": "- some\n- other"},
				Entrypoint: []string{"/entrypoint"},
			},
		},
		func(e error) {
			errored = true
			assert.Equal(t, UnknownLabelError{"io.whalebrew.some.key"}, e)
		},
	)
	assert.True(t, errored)
}

func TestLintImageErrorsWhenLabelCantBeDecodedFromContainerConfig(t *testing.T) {
	errored := false
	LintImage(
		types.ImageInspect{
			ContainerConfig: &container.Config{
				Labels: map[string]string{"io.whalebrew.config.environment": "content of label"},
			},
			Config: &container.Config{
				Entrypoint: []string{"/entrypoint"},
			},
		},
		func(e error) {
			errored = true
			assert.IsType(t, LabelError{}, e)
			assert.Contains(t, e.Error(), "io.whalebrew.config.environment")
			assert.Contains(t, e.Error(), "cannot unmarshal")
			assert.Contains(t, e.Error(), "content of label")
		},
	)
	assert.True(t, errored)
}

func TestLintImageErrorsWhenLabelCantBeDecodedFromImageConfig(t *testing.T) {
	errored := false
	LintImage(
		types.ImageInspect{
			Config: &container.Config{
				Labels:     map[string]string{"io.whalebrew.config.environment": "content of label"},
				Entrypoint: []string{"/entrypoint"},
			},
		},
		func(e error) {
			errored = true
			assert.IsType(t, LabelError{}, e)
			assert.Contains(t, e.Error(), "io.whalebrew.config.environment")
			assert.Contains(t, e.Error(), "cannot unmarshal")
			assert.Contains(t, e.Error(), "content of label")
		},
	)
	assert.True(t, errored)
}

func TestLintImageSucceedsFromContainerConfig(t *testing.T) {
	LintImage(
		types.ImageInspect{
			ContainerConfig: &container.Config{
				Labels: map[string]string{"io.whalebrew.config.environment": "[NAME]"},
			},
			Config: &container.Config{
				Entrypoint: []string{"/entrypoint"},
			},
		},
		func(e error) {
			t.Errorf("when labels are correct, no error should be raised")
		},
	)
}

func TestLintImageSucceedsFromImageConfig(t *testing.T) {
	LintImage(
		types.ImageInspect{
			Config: &container.Config{
				Labels:     map[string]string{"io.whalebrew.config.environment": "[NAME]"},
				Entrypoint: []string{"/entrypoint"},
			},
		},
		func(e error) {
			t.Errorf("when labels are correct, no error should be raised")
		},
	)
}

func TestLoadImageLabelDecodesYamlString(t *testing.T) {
	value := ""
	assert.NoError(
		t,
		loadImageLabel(
			types.ImageInspect{
				ContainerConfig: &container.Config{
					Labels: map[string]string{"io.whalebrew.some.key": `"some value"`},
				},
			},
			"some.key",
			&value,
		),
	)
	assert.Equal(t, "some value", value)
}

func TestLoadImageLabelDecodesPlainString(t *testing.T) {
	value := ""
	assert.NoError(
		t,
		loadImageLabel(
			types.ImageInspect{
				ContainerConfig: &container.Config{
					Labels: map[string]string{"io.whalebrew.some.key": "some: value"},
				},
			},
			"some.key",
			&value,
		),
	)
	assert.Equal(t, "some: value", value)
}

func TestLoadImageLabelPrefersContainerConfig(t *testing.T) {
	value := []string{}
	assert.NoError(
		t,
		loadImageLabel(
			types.ImageInspect{
				ContainerConfig: &container.Config{
					Labels: map[string]string{"io.whalebrew.config.environment": `["SOME_CONFIG_OPTION"]`},
				},
				Config: &container.Config{
					Labels: map[string]string{"io.whalebrew.config.environment": `["OTHER_CONFIG_OPTION"]`},
				},
			},
			"config.environment",
			&value,
		),
	)
	assert.Equal(t, []string{"SOME_CONFIG_OPTION"}, value)
}

func TestLoadImageLabelFallsBackToConfig(t *testing.T) {
	value := []string{}
	assert.NoError(
		t,
		loadImageLabel(
			types.ImageInspect{
				Config: &container.Config{
					Labels: map[string]string{"io.whalebrew.config.environment": `["OTHER_CONFIG_OPTION"]`},
				},
			},
			"config.environment",
			&value,
		),
	)
	assert.Equal(t, []string{"OTHER_CONFIG_OPTION"}, value)
}

func TestNewPackageFromImage(t *testing.T) {
	// with tag
	pkg, err := NewPackageFromImage("whalebrew/foo:bar", types.ImageInspect{})
	assert.Nil(t, err)
	assert.Equal(t, pkg.Name, "foo")
	assert.Equal(t, pkg.Image, "whalebrew/foo:bar")

	assert.Equal(t, "ws", mustNewTestPkg(t, "io.whalebrew.name", "ws").Name)
	assert.Equal(t, "whalebrew/whalesay", mustNewTestPkg(t, "io.whalebrew.name", "ws").Image)
	assert.Equal(t, []string{"SOME_CONFIG_OPTION"}, mustNewTestPkg(t, "io.whalebrew.config.environment", `["SOME_CONFIG_OPTION"]`).Environment)
	assert.Equal(t, []string{"/somesource:/somedest"}, mustNewTestPkg(t, "io.whalebrew.config.volumes", `["/somesource:/somedest"]`).Volumes)
	assert.Equal(t, []string{"8100:8100"}, mustNewTestPkg(t, "io.whalebrew.config.ports", `["8100:8100"]`).Ports)
	assert.Equal(t, []string{"host"}, mustNewTestPkg(t, "io.whalebrew.config.networks", `["host"]`).Networks)
	assert.Equal(t, []string{"C", "exec-path"}, mustNewTestPkg(t, "io.whalebrew.config.volumes_from_args", `["-C", "--exec-path"]`).PathArguments)

	assert.True(t, mustNewTestPkg(t, "io.whalebrew.config.missing_volumes", "mount").MountMissingVolumes)
	assert.False(t, mustNewTestPkg(t, "io.whalebrew.config.missing_volumes", "mount").SkipMissingVolumes)

	assert.False(t, mustNewTestPkg(t, "io.whalebrew.config.missing_volumes", "skip").MountMissingVolumes)
	assert.True(t, mustNewTestPkg(t, "io.whalebrew.config.missing_volumes", "skip").SkipMissingVolumes)

	assert.False(t, mustNewTestPkg(t, "io.whalebrew.config.missing_volumes", "error").MountMissingVolumes)
	assert.False(t, mustNewTestPkg(t, "io.whalebrew.config.missing_volumes", "error").SkipMissingVolumes)

	assert.False(t, mustNewTestPkg(t, "any", "ws").MountMissingVolumes)
	assert.False(t, mustNewTestPkg(t, "any", "other").SkipMissingVolumes)

	assert.Equal(t, "example", mustNewTestPackageFromImage(t, "quay.io/some/registry/example").Name)
	assert.Equal(t, "quay.io/some/registry/example", mustNewTestPackageFromImage(t, "quay.io/some/registry/example").Image)

	_, err = newTestPkg("io.whalebrew.config.missing_volumes", "some-other")
	assert.Error(t, err)

	_, err = newTestPkg("io.whalebrew.required_version", "some-other")
	assert.Error(t, err)
	_, err = newTestPkg("io.whalebrew.required_version", ">0.0.1")
	assert.NoError(t, err)
}

func TestPreinstallMessage(t *testing.T) {
	pkg := &Package{}
	assert.Equal(t, "", pkg.PreinstallMessage(nil))

	pkg = &Package{
		Environment: []string{"AWS_ACCESS_KEY"},
		Ports: []string{
			"80:80",
			"81:81:udp",
		},
		Volumes: []string{
			"/etc/passwd:/passwdtosteal",
			"/etc/readonly:/readonly:ro",
		},
	}
	assert.Equal(t,
		"This package needs additional access to your system. It wants to:\n"+
			"\n"+
			"* Read the environment variable AWS_ACCESS_KEY\n"+
			"* Listen on TCP port 80\n"+
			"* Listen on UDP port 81\n"+
			"* Read and write to the file or directory \"/etc/passwd\"\n"+
			"* Read the file or directory \"/etc/readonly\"\n",
		pkg.PreinstallMessage(nil))
}

func TestLoadPackageFromFile(t *testing.T) {
	_, err := LoadPackageFromPath("resources/aws")
	assert.NoError(t, err)
	_, err = LoadPackageFromPath("resources/aws-incompatible")
	assert.Error(t, err)
	_, err = LoadPackageFromPath("resources/syntax-error")
	assert.Error(t, err)
	_, err = LoadPackageFromPath("resources/file-that-does-not-exist")
	assert.Error(t, err)
}
