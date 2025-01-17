package config

import (
    "encoding/json"
    "fmt"
    "os"
)

type DomainConfig struct {
    Domain      string `json:"Domain"`
    NotifyEmail string `json:"NotifyEmail"`
}

func LoadDomainConfigs() ([]DomainConfig, error) {
        filePath := "config/domains.json"
        data, err := os.ReadFile(filePath)
        if err != nil {
            return nil, fmt.Errorf("failed to open domain config file: %v", err)
        }
    
        var domainConfigs []DomainConfig
        err = json.Unmarshal(data, &domainConfigs)
        if err != nil {
            return nil, fmt.Errorf("failed to parse domain config file: %v", err)
        }
    
        return domainConfigs, nil
    }
    
