package hooks

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/viper"
)

type stater interface {
	Stat(string) (os.FileInfo, error)
}

type runner interface {
	Run(string, ...string) error
}

type dirGetChanger interface {
	Chdir(string) error
	Getwd() (string, error)
}

type osStater struct{}

func (osStater) Stat(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

type execRunner struct{}

func (execRunner) Run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

type osDirGetChanger struct{}

func (osDirGetChanger) Getwd() (string, error) {
	return os.Getwd()
}
func (osDirGetChanger) Chdir(path string) error {
	return os.Chdir(path)
}

func run(s stater, r runner, wdChanger dirGetChanger, configDir, installPath string, hook string, args ...string) error {
	hookPath := filepath.Join(configDir, "hooks", hook)
	wd, err := wdChanger.Getwd()
	if err != nil {
		return fmt.Errorf("unable to get current directory: %s", err.Error())
	}
	err = wdChanger.Chdir(installPath)
	if err != nil {
		return fmt.Errorf("unable to change directory to %s: %s", installPath, err.Error())
	}
	defer func() {
		wdChanger.Chdir(wd)
	}()
	if stat, err := s.Stat(hookPath); err == nil {
		if stat.IsDir() || stat.Mode().Perm()&0100 != 0100 {
			return fmt.Errorf("%s: file is not executable", hookPath)
		}
		if err := r.Run(hookPath, args...); err != nil {
			return fmt.Errorf("%s: %s", hookPath, err.Error())
		}
	}
	return nil
}

func Run(hook string, args ...string) error {
	return run(osStater{}, execRunner{}, osDirGetChanger{}, viper.GetString("config_dir"), viper.GetString("install_path"), hook, args...)
}
