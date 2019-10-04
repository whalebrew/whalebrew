package cmd_test

import (
	"errors"
	"os"
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
	assert.Error(t, cmd.Run(testRunner(f), []string{"whalebrew", "../packages/resources/aws"}))
	os.Setenv("TEST_ENVIRONMENT_VARIABLE", "SOME-VALUE")
	f = func(p *packages.Package, e *run.Execution) error {
		assert.Contains(t, e.Environment, "TEST_ENV=SOME-VALUE")
		assert.Equal(t, 2, len(e.Volumes))
		return nil
	}
	assert.NoError(t, cmd.Run(testRunner(f), []string{"whalebrew", "../packages/resources/aws"}))
	f = func(p *packages.Package, e *run.Execution) error {
		return nil
	}
	assert.Error(t, cmd.Run(testRunner(f), []string{"whalebrew", "./this-package-does-not-exist", "arg1"}))
}
