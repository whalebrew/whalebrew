package packages

import (
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/stretchr/testify/assert"
)

func TestNewPackageFromImage(t *testing.T) {
	// with tag
	pkg, err := NewPackageFromImage("whalebrew/foo:bar", types.ImageInspect{})
	assert.Nil(t, err)
	assert.Equal(t, pkg.Name, "foo")
	assert.Equal(t, pkg.Image, "whalebrew/foo:bar")

	// test labels
	pkg, err = NewPackageFromImage("whalebrew/whalesay", types.ImageInspect{
		ContainerConfig: &container.Config{
			Labels: map[string]string{
				"io.whalebrew.name":                "ws",
				"io.whalebrew.config.environment":  "[\"SOME_CONFIG_OPTION\"]",
				"io.whalebrew.config.volumes":      "[\"/somesource:/somedest\"]",
				"io.whalebrew.config.ports":        "[\"8100:8100\"]",
				"io.whalebrew.config.networks":     "[\"host\"]",
				"io.whalebrew.post-install-message":"Some message to display after install",
			},
		},
	})
	assert.Nil(t, err)
	assert.Equal(t, "ws", pkg.Name)
	assert.Equal(t, "whalebrew/whalesay", pkg.Image)
	assert.Equal(t, []string{"SOME_CONFIG_OPTION"}, pkg.Environment)
	assert.Equal(t, []string{"/somesource:/somedest"}, pkg.Volumes)
	assert.Equal(t, []string{"8100:8100"}, pkg.Ports )
	assert.Equal(t, pkg.Networks, []string{"host"})
	assert.Equal(t, "Some message to display after install", pkg.PostInstallMessage)

}

func TestPreinstallMessage(t *testing.T) {
	pkg := &Package{}
	assert.Equal(t, pkg.PreinstallMessage(), "")

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
	assert.Equal(t, pkg.PreinstallMessage(),
		"This package needs additional access to your system. It wants to:\n"+
			"\n"+
			"* Read the environment variable AWS_ACCESS_KEY\n"+
			"* Listen on TCP port 80\n"+
			"* Listen on UDP port 81\n"+
			"* Read and write to the file or directory \"/etc/passwd\"\n"+
			"* Read the file or directory \"/etc/readonly\"\n",
	)
}
