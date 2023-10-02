package config_test

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/whalebrew/whalebrew/config"
)

func createConfigFile(t *testing.T, dir string, content ...io.Reader) {
	t.Helper()
	require.NoError(t, os.MkdirAll(dir, 0777))
	fd, err := os.Create(filepath.Join(dir, "config.yaml"))
	require.NoError(t, err)
	defer func() {
		require.NoError(t, fd.Close())
	}()
	for _, reader := range content {
		_, err := io.Copy(fd, reader)
		require.NoError(t, err)
	}
}

func TestGetConfigDir(t *testing.T) {
	t.Cleanup(func() {
		os.RemoveAll(".test-resources")
	})
	t.Run("without specific whalebrew config dir", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "/do-not-exist")
		t.Setenv("XDG_CONFIG_DIRS", "/do-not-exist")
		t.Setenv("WHALEBREW_CONFIG_DIR", "")
		t.Setenv("HOME", "home")
		t.Setenv("USERPROFILE", "home")
		assert.Equal(t, "home/.whalebrew", config.ConfigDir())
	})

	t.Run("with specific whalebrew config dir", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "/do-not-exist")
		t.Setenv("XDG_CONFIG_DIRS", "/do-not-exist")
		t.Setenv("WHALEBREW_CONFIG_DIR", ".whalebrew")
		t.Setenv("HOME", "home")
		t.Setenv("USERPROFILE", "home")
		assert.Equal(t, ".whalebrew", config.ConfigDir())
	})

	t.Run("when detecting whalebrew config directory", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", ".test-resources/xdg.home")
		t.Setenv("XDG_CONFIG_DIRS", ".test-resources/xdg.global")
		t.Setenv("WHALEBREW_CONFIG_DIR", "")
		t.Setenv("HOME", ".test-resources/home")
		t.Setenv("USERPROFILE", ".test-resources/home")
		t.Run("when only global sdg path exists", func(t *testing.T) {
			require.NoError(t, os.RemoveAll(".test-resources"))
			createConfigFile(t, ".test-resources/xdg.global/whalebrew")

			assert.Equal(t, ".test-resources/xdg.global/whalebrew", config.ConfigDir())
		})
		t.Run("when global and local xdg paths exists", func(t *testing.T) {
			require.NoError(t, os.RemoveAll(".test-resources"))
			createConfigFile(t, ".test-resources/xdg.global/whalebrew")
			createConfigFile(t, ".test-resources/xdg.home/whalebrew")

			assert.Equal(t, ".test-resources/xdg.home/whalebrew", config.ConfigDir())
		})
		t.Run("when whalebrew config path and global and local xdg paths exists", func(t *testing.T) {
			require.NoError(t, os.RemoveAll(".test-resources"))
			createConfigFile(t, ".test-resources/xdg.global/whalebrew")
			createConfigFile(t, ".test-resources/xdg.local/whalebrew")
			createConfigFile(t, ".test-resources/home/.whalebrew")

			assert.Equal(t, ".test-resources/home/.whalebrew", config.ConfigDir())
		})
	})
}

func TestGetConfig(t *testing.T) {
	t.Cleanup(func() {
		config.Reset()
		os.RemoveAll(".test-resources")
	})
	t.Run("When install path is not provided as an environment variable", func(t *testing.T) {
		t.Setenv("WHALEBREW_CONFIG_DIR", ".test-resources/whalebrew")
		t.Setenv("WHALEBREW_INSTALL_PATH", "")

		t.Run("When the config file does not exist", func(t *testing.T) {
			config.Reset()
			require.NoError(t, os.RemoveAll(".test-resources"))
			assert.Equal(t, "/usr/local/bin", config.GetConfig().InstallPath)
		})

		t.Run("When the config file exists", func(t *testing.T) {
			config.Reset()
			createConfigFile(t, ".test-resources/whalebrew", strings.NewReader(`install_path: my-path`))

			assert.Equal(t, "my-path", config.GetConfig().InstallPath)
		})
	})
	t.Run("When install path is provided as an environment variable", func(t *testing.T) {
		t.Setenv("WHALEBREW_CONFIG_DIR", ".test-resources/whalebrew")
		t.Setenv("WHALEBREW_INSTALL_PATH", "env-path")

		t.Run("When the config file does not exist", func(t *testing.T) {
			config.Reset()
			require.NoError(t, os.RemoveAll(".test-resources"))
			assert.Equal(t, "env-path", config.GetConfig().InstallPath)
		})

		t.Run("When the config file exists", func(t *testing.T) {
			config.Reset()
			createConfigFile(t, ".test-resources/whalebrew", strings.NewReader(`install_path: my-path`))

			assert.Equal(t, "env-path", config.GetConfig().InstallPath)
		})
	})
}
