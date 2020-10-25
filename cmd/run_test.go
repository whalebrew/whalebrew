package cmd_test

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/whalebrew/whalebrew/cmd"
	"github.com/whalebrew/whalebrew/packages"
	"github.com/whalebrew/whalebrew/run"
)

type testRunner func(p *packages.Package, e *run.Execution) error

func (tr testRunner) Run(p *packages.Package, e *run.Execution) error {
	return tr(p, e)
}

type testLoader func(path string) (*packages.Package, error)

func (tl testLoader) LoadPackageFromPath(path string) (*packages.Package, error) {
	return tl(path)
}

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

func TestRun(t *testing.T) {
	f := func(p *packages.Package, e *run.Execution) error {
		return errors.New("test error")
	}
	assert.Error(t, cmd.Run(packages.DefaultLoader, testRunner(f), []string{"whalebrew", "../packages/resources/aws"}))
	os.Setenv("TEST_ENVIRONMENT_VARIABLE", "SOME-VALUE")
	f = func(p *packages.Package, e *run.Execution) error {
		assert.Contains(t, e.Environment, "TEST_ENV=SOME-VALUE")
		assert.Equal(t, 2, len(e.Volumes))
		return nil
	}
	assert.NoError(t, cmd.Run(packages.DefaultLoader, testRunner(f), []string{"whalebrew", "../packages/resources/aws"}))
	f = func(p *packages.Package, e *run.Execution) error {
		return nil
	}
	assert.Error(t, cmd.Run(packages.DefaultLoader, testRunner(f), []string{"whalebrew", "./this-package-does-not-exist", "arg1"}))
}

func TestRunWorkdirIsExpanded(t *testing.T) {
	os.Setenv("HOME", "/homes/test-user")
	assert.NoError(
		t,
		cmd.Run(
			testLoader(func(string) (*packages.Package, error) {
				return &packages.Package{WorkingDir: "$HOME"}, nil
			}),
			testRunner(func(p *packages.Package, e *run.Execution) error {
				assert.Equal(t, "/homes/test-user", e.WorkingDir)
				return nil
			}),
			[]string{"whalebrew", "/usr/local/bin/pkg"},
		),
	)
}

func TestRunVolumesIsExpanded(t *testing.T) {
	os.Setenv("HOME", "./resources")
	assert.NoError(
		t,
		cmd.Run(
			testLoader(func(string) (*packages.Package, error) {
				return &packages.Package{
					Volumes:    []string{"$HOME:/resources"},
					WorkingDir: "/workdir",
				}, nil
			}),
			testRunner(func(p *packages.Package, e *run.Execution) error {
				assert.Equal(t, []string{"./resources:/resources"}, e.Volumes[:len(e.Volumes)-1])
				v := strings.SplitN(e.Volumes[len(e.Volumes)-1], ":", 2)
				assert.Equal(t, "/workdir", v[1])
				return nil
			}),
			[]string{"whalebrew", "/usr/local/bin/pkg"},
		),
	)
}

func TestRunCLIVolumesIsConsidered(t *testing.T) {
	os.Setenv("HOME", "./resources")
	wd, err := os.Getwd()
	assert.NoError(t, err)
	assert.NoError(
		t,
		cmd.Run(
			testLoader(func(string) (*packages.Package, error) {
				return &packages.Package{
					PathArguments: []string{"c", "change-dir"},
				}, nil
			}),
			testRunner(func(p *packages.Package, e *run.Execution) error {
				assert.Equal(t, []string{"/bla:/bla", wd + "/hello-world:" + wd + "/hello-world"}, e.Volumes[1:])
				return nil
			}),
			[]string{"whalebrew", "/usr/local/bin/pkg", "-c", "/bla", "--change-dir", "hello-world"},
		),
	)
}

func TestRunEnvironmentIsExpanded(t *testing.T) {
	os.Setenv("HOME", "./resources")
	assert.NoError(
		t,
		cmd.Run(
			testLoader(func(string) (*packages.Package, error) {
				return &packages.Package{
					Environment: []string{"HOME=${HOME}"},
				}, nil
			}),
			testRunner(func(p *packages.Package, e *run.Execution) error {
				assert.Equal(t, []string{"HOME=./resources"}, e.Environment)
				return nil
			}),
			[]string{"whalebrew", "/usr/local/bin/pkg"},
		),
	)
}
