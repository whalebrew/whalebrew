package search

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/whalebrew/whalebrew/config"
)

func TestForRepositories(t *testing.T) {
	for searcher := range ForRegistries([]config.Registry{{}}, func(err error) bool {
		assert.Error(t, err)
		return true
	}) {
		t.Errorf("no searcher should be returned for empty config, got: %v", searcher)
	}
	count := 0
	for searcher := range ForRegistries([]config.Registry{}, func(err error) bool {
		t.Errorf("With a valid config, no error should be raised. Got: %v", err)
		return true
	}) {
		assert.IsType(t, &DockerHub{}, searcher)
		if dh, ok := searcher.(*DockerHub); ok {
			assert.Equal(t, "whalebrew", dh.Owner)
		}
		count++
	}
	assert.Equal(t, 1, count)
}
