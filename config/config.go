// pkg/config/config.go
package config

import (
    "log"
    "os"

    "gopkg.in/yaml.v2"
)

type Config struct {
    Domains       []string `yaml:"domains"`
    NotifyEmail   string   `yaml:"notifyEmail"`
    TraefikAPIURL string   `yaml:"traefikAPIUrl"`
}

func LoadConfig(filePath string) Config {
    file, err := os.ReadFile(filePath)
    if err != nil {
        log.Fatalf("Failed to load config file: %v", err)
    }

    var cfg Config
    err = yaml.Unmarshal(file, &cfg)
    if err != nil {
        log.Fatalf("Failed to parse config file: %v", err)
    }

    return cfg
}
