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
				"io.whalebrew.name":               "ws",
				"io.whalebrew.config.environment": "[\"SOME_CONFIG_OPTION\"]",
				"io.whalebrew.config.volumes":     "[\"/somesource:/somedest\"]",
				"io.whalebrew.config.ports":       "[\"8100:8100\"]",
			},
		},
	})
	assert.Nil(t, err)
	assert.Equal(t, pkg.Name, "ws")
	assert.Equal(t, pkg.Image, "whalebrew/whalesay")
	assert.Equal(t, pkg.Environment, []string{"SOME_CONFIG_OPTION"})
	assert.Equal(t, pkg.Volumes, []string{"/somesource:/somedest"})
	assert.Equal(t, pkg.Ports, []string{"8100:8100"})

}
