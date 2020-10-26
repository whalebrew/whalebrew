package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
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
	Registries []Registry `yaml:"registries"`
}

func GetConfig() Config {
	configPath := filepath.Join(viper.GetString("config_dir"), "config.yaml")
	fd, err := os.Open(configPath)
	c := Config{}
	if err == nil {
		defer fd.Close()
		yaml.NewDecoder(fd).Decode(&c)
	}
	if len(c.Registries) == 0 {
		c.Registries = []Registry{{DockerHub: &DockerHubRegistry{Owner: "whalebrew"}}}
	}
	return c
}
