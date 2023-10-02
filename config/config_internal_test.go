package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestXDGPaths(t *testing.T) {
	t.Run("With no specific environment variable", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "")
		t.Setenv("XDG_CONFIG_DIRS", "")
		t.Setenv("HOME", "home")
		t.Setenv("USERPROFILE", "home")
		assert.Equal(t, []string{"home/.config/test", "/etc/xdg/test"}, xdgConfigDirs("test"))
	})
	t.Run("With specific environment variables", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "my-home/.config")
		t.Setenv("XDG_CONFIG_DIRS", "/config:/local/config")
		assert.Equal(t, []string{"my-home/.config/test", "/config/test", "/local/config/test"}, xdgConfigDirs("test"))
	})
}
