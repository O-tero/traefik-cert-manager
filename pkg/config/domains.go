package config

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

type DomainConfig struct {
	ServiceName string `json:"service_name"`
	Domain      string `json:"domain"`
}

var (
	domainConfigFile = "config/domains.json"
	configMutex      sync.Mutex
)

func LoadDomainConfigs() ([]DomainConfig, error) {
	configMutex.Lock()
	defer configMutex.Unlock()

	file, err := os.Open(domainConfigFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open domain config file: %v", err)
	}
	defer file.Close()

	var configs []DomainConfig
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&configs); err != nil {
		return nil, fmt.Errorf("failed to parse domain config: %v", err)
	}

	return configs, nil
}

func SaveDomainConfigs(configs []DomainConfig) error {
	configMutex.Lock()
	defer configMutex.Unlock()

	file, err := os.Create(domainConfigFile)
	if err != nil {
		return fmt.Errorf("failed to create domain config file: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(configs); err != nil {
		return fmt.Errorf("failed to write domain config: %v", err)
	}

	return nil
}
