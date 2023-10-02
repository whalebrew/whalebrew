package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

const (
	configPath = "config.yaml"
)

var (
	config Config
	once   = sync.Once{}
)

type DockerHubRegistry struct {
	Owner string `yaml:"owner"`
}

type DockerRegistry struct {
	Owner   string `yaml:"owner"`
	Host    string `yaml:"host"`
	UseHTTP bool   `yaml:"useHTTP"`
}

type Registry struct {
	DockerHub      *DockerHubRegistry `yaml:"dockerHub"`
	DockerRegistry *DockerRegistry    `yaml:"dockerRegistry"`
}

type Config struct {
	InstallPath          string     `yaml:"install_path" env:"install_path" mapstructure:"install_path"`
	Registries           []Registry `yaml:"registries"`
	isDefaultInstallPath bool
}

func parseYaml(path string, out interface{}) error {
	fd, err := os.Open(path)
	if err != nil {
		return err
	}
	defer fd.Close()
	d := yaml.NewDecoder(fd)
	d.KnownFields(true)
	err = d.Decode(out)
	if err != nil {
		return err
	}
	return nil
}

func ConfigDir() string {
	if os.Getenv("WHALEBREW_CONFIG_DIR") != "" {
		return os.Getenv("WHALEBREW_CONFIG_DIR")
	}
	for _, candidate := range append(
		[]string{
			filepath.Join(Home(), ".whalebrew"),
		},
		xdgConfigDirs("whalebrew")...) {
		_, err := os.Stat(filepath.Join(candidate, configPath))
		if err == nil {
			return candidate
		}
	}
	return filepath.Join(Home(), ".whalebrew")
}

func defaultInstallDir() string {
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		return "/opt/whalebrew/bin"
	}
	return "/usr/local/bin"
}

func (c Config) IsDefaultInstallPath() bool {
	return c.isDefaultInstallPath
}

func ConfigPath() string {
	return filepath.Join(ConfigDir(), "config.yaml")
}

func GetConfig() Config {
	once.Do(func() {
		err := parseYaml(ConfigPath(), &config)
		if err != nil && !os.IsNotExist(err) {
			fmt.Printf("Invalid whalebrew configuration in %s: %v\n", filepath.Join(ConfigDir(), "config.yaml"), err)
			os.Exit(1)
		}
		if os.Getenv("WHALEBREW_INSTALL_PATH") != "" {
			config.InstallPath = os.Getenv("WHALEBREW_INSTALL_PATH")
		}
		if config.InstallPath == "" {
			config.InstallPath = defaultInstallDir()
			config.isDefaultInstallPath = true
		}
	})
	return config
}

func Reset() {
	once = sync.Once{}
}

func xdgConfigDirs(suffix string) []string {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		configHome = filepath.Join(Home(), ".config")
	}
	configHome = filepath.Join(configHome, suffix)
	configDirs := os.Getenv("XDG_CONFIG_DIRS")
	if configDirs == "" {
		configDirs = "/etc/xdg"
	}
	configDirsSlice := strings.Split(configDirs, string(os.PathListSeparator))
	for i, dir := range configDirsSlice {
		configDirsSlice[i] = filepath.Join(dir, suffix)
	}
	return append([]string{configHome}, configDirsSlice...)
}

func Home() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
