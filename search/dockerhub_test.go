package search

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDockerHub(t *testing.T) {
	d := DockerHub{Owner: "whalebrew"}
	count := 0
	for img := range d.Search("jq", nil) {
		assert.Equal(t, "whalebrew/jq", img)
		count++
	}
	assert.Equal(t, 1, count)
	d = DockerHub{Owner: "bitnami"}
	count = 0
	for img := range d.Search("kubectl", nil) {
		assert.Equal(t, "bitnami/kubectl", img)
		count++
	}
	assert.Equal(t, 1, count)
}
