package cmd_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/whalebrew/whalebrew/cmd"
)

func TestIsShellbang(t *testing.T) {
	t.Run("when called without enough arguments, command is not detected as shellbang", func(t *testing.T) {
		assert.False(t, cmd.IsShellbang([]string{}))
	})

	t.Run("when called with a subcommand, command is not detected as shellbang", func(t *testing.T) {
		assert.False(t, cmd.IsShellbang([]string{"whalebrew", "list"}))
	})

	t.Run("when called with an absolute path, command is detected as shellbang", func(t *testing.T) {
		assert.True(t, cmd.IsShellbang([]string{"whalebrew", "/usr/local/bin/curl"}))
	})
}
