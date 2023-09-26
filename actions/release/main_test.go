package main

import (
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/actions-go/toolkit/core"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func copyToFS(t *testing.T, fs afero.Fs, path string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("../../", path))
	require.NoError(t, err)

	fd, err := fs.Create(path)
	require.NoError(t, err)
	defer fd.Close()
	written, err := fd.Write(data)
	require.NoError(t, err)
	assert.Equal(t, len(data), written)
}

func assertFileContains(t *testing.T, fs afero.Fs, path string, contains string) {
	t.Helper()
	fd, err := fs.Open(path)
	require.NoError(t, err)
	defer fd.Close()

	data, err := io.ReadAll(fd)
	require.NoError(t, err)

	assert.Contains(t, string(data), contains)
}

func assertFileNotContains(t *testing.T, fs afero.Fs, path string, contains string) {
	t.Helper()
	fd, err := fs.Open(path)
	require.NoError(t, err)
	defer fd.Close()

	data, err := io.ReadAll(fd)
	require.NoError(t, err)

	assert.NotContains(t, string(data), contains)
}

func testVersionBump(t *testing.T, testName, version string, check func(t *testing.T, fs afero.Fs)) {
	t.Helper()
	if os.Getenv("TEST_RELEASE_ACTION_INTEGRATION") == "false" {
		t.Skip("disabled release action integration test")
		return
	}
	t.Run(testName, func(t *testing.T) {
		t.Cleanup(func() {
			os.RemoveAll(".fs")
		})
		require.NoError(t, os.MkdirAll(".fs/version", 0777))
		fs = afero.NewBasePathFs(afero.NewOsFs(), ".fs")
		gitWorktree = ".fs"

		require.NoError(t, git("init"))

		copyToFS(t, fs, "README.md")
		copyToFS(t, fs, "CHANGELOG.md")
		copyToFS(t, fs, "version/version.go")
		require.NoError(t, git("add", "README.md", "CHANGELOG.md", "version/version.go"))
		require.NoError(t, git("commit", "--allow-empty", "-m", "Initial commit"))

		t.Setenv("INPUT_VERSION", version)
		defer func() {
			recover()
		}()
		exit = func(code int) {
			t.Helper()
			assert.Failf(t, "Unexpected call to exit", "with code %d", code)
			panic(nil)
		}

		main()

		check(t, fs)
	})
}

func TestReleaseAction(t *testing.T) {
	t.Cleanup(func() {
		now = time.Now
		envHandler = newHandler()
		core.StartCommands("TestReleaseAction-stop-command")
	})
	core.StopCommands("TestReleaseAction-stop-command")
	t.Setenv("GITHUB_ACTIONS", "true")
	envHandler = newHandler()
	now = func() time.Time {
		return time.Date(2023, 9, 25, 0, 0, 0, 0, time.UTC)
	}
	t.Run("with no  version number", func(t *testing.T) {
		os.Unsetenv("INPUT_VERSION")
		var exitCode *int
		defer func() {
			recover()
		}()
		exit = func(code int) {
			exitCode = &code
			panic(nil)
		}
		main()
		require.NotNil(t, exitCode)
		assert.Equal(t, 1, *exitCode)
	})
	for _, invalid := range []string{"This.is.not-a-version-number", "v1.2.3.4.5", "v1.2.3-!build", "v1.2.3+build", "1.2.3+build"} {
		t.Run("with an invalid version number "+invalid, func(t *testing.T) {
			t.Setenv("INPUT_VERSION", invalid)
			var exitCode *int
			defer func() {
				recover()
			}()
			exit = func(code int) {
				exitCode = &code
				panic(nil)
			}
			main()
			require.NotNil(t, exitCode)
			assert.Equal(t, 1, *exitCode)
		})
	}

	testVersionBump(t, "Without the v prefix and no pre-release", "1.2.3", func(t *testing.T, fs afero.Fs) {
		assertFileContains(t, fs, "version/version.go", "1.2.4-dev")
		assertFileContains(t, fs, "README.md", "1.2.3")
		assertFileContains(t, fs, "CHANGELOG.md", "1.2.3")

		require.NoError(t, git("checkout", "HEAD^"))
		assertFileContains(t, fs, "version/version.go", "1.2.3+from-sources")
		assertFileContains(t, fs, "README.md", "1.2.3")
		assertFileContains(t, fs, "CHANGELOG.md", "1.2.3")

		// require.NoError(t, git("checkout", "1.2.3"))
		// assertFileContains(t, fs, "version/version.go", "1.2.3+from-sources")
		// assertFileContains(t, fs, "README.md", "1.2.3")
		// assertFileContains(t, fs, "CHANGELOG.md", "1.2.3")
	})
	testVersionBump(t, "With the v prefix and no pre-release", "v1.2.3", func(t *testing.T, fs afero.Fs) {
		assertFileContains(t, fs, "version/version.go", "1.2.4-dev")
		assertFileContains(t, fs, "README.md", "1.2.3")
		assertFileContains(t, fs, "CHANGELOG.md", "1.2.3 - 2023-09-25")

		require.NoError(t, git("checkout", "HEAD^"))
		assertFileContains(t, fs, "version/version.go", "1.2.3+from-sources")
		assertFileContains(t, fs, "README.md", "1.2.3")
		assertFileContains(t, fs, "CHANGELOG.md", "1.2.3 - 2023-09-25")

		// require.NoError(t, git("checkout", "1.2.3"))
		// assertFileContains(t, fs, "version/version.go", "1.2.3+from-sources")
		// assertFileContains(t, fs, "README.md", "1.2.3")
		// assertFileContains(t, fs, "CHANGELOG.md", "1.2.3 - 2023-09-25")
	})

	testVersionBump(t, "Without the v prefix and a pre-release", "1.2.3-alpha", func(t *testing.T, fs afero.Fs) {
		assertFileContains(t, fs, "version/version.go", "1.2.3-dev")
		assertFileNotContains(t, fs, "README.md", "1.2.3-alpha")
		assertFileContains(t, fs, "CHANGELOG.md", "1.2.3-alpha - 2023-09-25")

		require.NoError(t, git("checkout", "HEAD^"))
		assertFileContains(t, fs, "version/version.go", "1.2.3-alpha+from-sources")
		assertFileNotContains(t, fs, "README.md", "1.2.3-alpha")
		assertFileContains(t, fs, "CHANGELOG.md", "1.2.3-alpha - 2023-09-25")

		// require.NoError(t, git("checkout", "1.2.3-alpha"))
		// assertFileContains(t, fs, "version/version.go", "1.2.3-alpha+from-sources")
		// assertFileNotContains(t, fs, "README.md", "1.2.3-alpha")
		// assertFileContains(t, fs, "CHANGELOG.md", "1.2.3-alpha - 2023-09-25")
	})
	testVersionBump(t, "With the v prefix and a pre-release", "v1.2.3-alpha", func(t *testing.T, fs afero.Fs) {
		assertFileContains(t, fs, "version/version.go", "1.2.3-dev")
		assertFileNotContains(t, fs, "README.md", "1.2.3-alpha")
		assertFileContains(t, fs, "CHANGELOG.md", "1.2.3-alpha - 2023-09-25")

		require.NoError(t, git("checkout", "HEAD^"))
		assertFileContains(t, fs, "version/version.go", "1.2.3-alpha+from-sources")
		assertFileNotContains(t, fs, "README.md", "1.2.3-alpha")
		assertFileContains(t, fs, "CHANGELOG.md", "1.2.3-alpha")

		// require.NoError(t, git("checkout", "1.2.3-alpha"))
		// assertFileContains(t, fs, "version/version.go", "1.2.3-alpha+from-sources")
		// assertFileNotContains(t, fs, "README.md", "1.2.3-alpha")
		// assertFileContains(t, fs, "CHANGELOG.md", "1.2.3-alpha - 2023-09-25")
	})
}
