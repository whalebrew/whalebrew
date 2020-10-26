package search

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/whalebrew/whalebrew/dockerregistry"
)

type fakeCatalog struct{}

func (fakeCatalog) Catalog() (dockerregistry.Catalog, error) {
	return dockerregistry.Catalog{
		Repositories: []string{
			"some-folder/some-image",
			"some-folder/other-image",
			"some-folder-with-suffix/some-image",
			"other-folder/other-image",
		},
	}, nil
}

func (fakeCatalog) ImageName(path string) string {
	return "my-registry/" + path
}

func TestDockerRegistry(t *testing.T) {
	dr := DockerRegistry{
		Owner:    "some-folder",
		Registry: fakeCatalog{},
	}
	count := 0
	for img := range dr.Search("some", nil) {
		assert.Equal(t, "my-registry/some-folder/some-image", img)
		count++
	}
	assert.Equal(t, 1, count)
}
