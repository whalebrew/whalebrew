package hooks

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

type testRunner struct {
	t    *testing.T
	err  error
	name string
	args []string
}

func (tr testRunner) Run(name string, args ...string) error {
	assert.Equal(tr.t, tr.name, name)
	assert.EqualValues(tr.t, tr.args, args)
	return tr.err
}

type testStater struct {
	t        *testing.T
	fileInfo os.FileInfo
	err      error
	name     string
}

func (ts testStater) Stat(name string) (os.FileInfo, error) {
	assert.Equal(ts.t, ts.name, name)
	return ts.fileInfo, ts.err
}

type testFileInfo struct {
	mode  os.FileMode
	isDir bool
}

func (testFileInfo) Name() string {
	return ""
}

func (testFileInfo) Size() int64 {
	return 0
}

func (tfi testFileInfo) Mode() os.FileMode {
	return tfi.mode
}

func (testFileInfo) ModTime() time.Time {
	return time.Now()
}

func (tfi testFileInfo) IsDir() bool {
	return tfi.isDir
}

func (testFileInfo) Sys() interface{} {
	return nil
}

type testDirChanger struct {
	t         *testing.T
	cwd       string
	dirs      []string
	getcwdErr error
	chdirErr  error
}

func (tdc *testDirChanger) Getwd() (string, error) {
	return tdc.cwd, tdc.getcwdErr
}

func (tdc *testDirChanger) Chdir(path string) error {
	d := tdc.dirs[0]
	tdc.dirs = tdc.dirs[1:]
	assert.Equal(tdc.t, d, path)
	return tdc.chdirErr
}

func TestRun(t *testing.T) {
	t.Run("When the hook exists", func(t *testing.T) {
		assert.NoError(
			t,
			run(
				testStater{t, testFileInfo{os.FileMode(0700), false}, nil, "/tmp/.whalebrew/hooks/post-install"},
				testRunner{t, nil, "/tmp/.whalebrew/hooks/post-install", nil},
				osDirGetChanger{},
				"/tmp",
				"post-install"),
		)
		assert.NoError(
			t,
			run(
				testStater{t, testFileInfo{os.FileMode(0700), false}, nil, "/tmp/.whalebrew/hooks/post-install"},
				testRunner{t, nil, "/tmp/.whalebrew/hooks/post-install", []string{"an-argument"}},
				&testDirChanger{t, "some/path", []string{"/tmp", "some/path"}, nil, nil},
				"/tmp",
				"post-install",
				"an-argument"),
		)
	})
	t.Run("When failing to get current directory", func(t *testing.T) {
		assert.Error(
			t,
			run(
				testStater{t, testFileInfo{os.FileMode(0600), false}, nil, "/tmp/.whalebrew/hooks/post-install"},
				testRunner{t, nil, "/tmp/.whalebrew/hooks/post-install", nil},
				&testDirChanger{t, "", nil, fmt.Errorf("testError"), nil},
				"/tmp",
				"post-install",
				"an-argument"),
		)
	})
	t.Run("When failing to change directory", func(t *testing.T) {
		assert.Error(
			t,
			run(
				testStater{t, testFileInfo{os.FileMode(0600), false}, nil, "/tmp/.whalebrew/hooks/post-install"},
				testRunner{t, nil, "/tmp/.whalebrew/hooks/post-install", nil},
				&testDirChanger{t, "some/path", []string{"/tmp/whalebrew"}, nil, fmt.Errorf("testError")},
				"/tmp/whalebrew",
				"post-install",
				"an-argument"),
		)
	})
	t.Run("When webhook is not executable", func(t *testing.T) {
		assert.Error(
			t,
			run(
				testStater{t, testFileInfo{os.FileMode(0600), false}, nil, "/tmp/.whalebrew/hooks/post-install"},
				testRunner{t, nil, "/tmp/.whalebrew/hooks/post-install", nil},
				osDirGetChanger{},
				"/tmp",
				"post-install",
				"an-argument"),
		)
	})
	t.Run("When webhook is a directory", func(t *testing.T) {
		assert.Error(
			t,
			run(
				testStater{t, testFileInfo{os.FileMode(0700), true}, nil, "/tmp/.whalebrew/hooks/post-install"},
				testRunner{t, nil, "/tmp/.whalebrew/hooks/post-install", nil},
				osDirGetChanger{},
				"/tmp",
				"post-install",
				"an-argument"),
		)
	})

	t.Run("When command fails exists", func(t *testing.T) {
		assert.Error(
			t,
			run(
				testStater{t, testFileInfo{os.FileMode(0700), false}, nil, "/tmp/.whalebrew/hooks/post-install"},
				testRunner{t, fmt.Errorf("test-error"), "/tmp/.whalebrew/hooks/post-install", []string{"an-argument"}},
				osDirGetChanger{},
				"/tmp",
				"post-install",
				"an-argument"),
		)
	})
	t.Run("When the webhook does not exist", func(t *testing.T) {
		viper.Set("install_path", ".")
		fmt.Println(Run(
			"post-install",
			"an-argument"))
		assert.NoError(
			t,
			Run(
				"post-install",
				"an-argument"),
		)
	})
}

func TestExecRunner(t *testing.T) {
	assert.NoError(t, execRunner{}.Run("ls", "-al"))
	assert.Error(t, execRunner{}.Run("false"))
}
