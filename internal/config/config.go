package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v2"
)

// application configuration
type Config struct {
	TraefikAPI   string       `yaml:"traefik_api"`
	Email        string       `yaml:"email"`
	Notification Notification `yaml:"notification"`
	Domains      []Domain     `yaml:"domains"`
	ACME         ACME         `yaml:"acme"`
	Certificates Certificates `yaml:"certificates"`
	App          App          `yaml:"app"`
}

type Notification struct {
	SMTPHost string `yaml:"smtp_host"`
	SMTPPort int    `yaml:"smtp_port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	From     string `yaml:"from"`
}

type Domain struct {
	Service string   `yaml:"service"`
	Domain  string   `yaml:"domain"`
	Aliases []string `yaml:"aliases"`
}

// ACME client configuration
type ACME struct {
	CADirURL string `yaml:"ca_dir_url"`
	KeyType  string `yaml:"key_type"`
	Email    string `yaml:"email"`
}

// Certificate management settings
type Certificates struct {
	RenewalDays int    `yaml:"renewal_days"`
	StoragePath string `yaml:"storage_path"`
}

// App holds application-level settings
type App struct {
	LogLevel      string `yaml:"log_level"`
	CheckInterval string `yaml:"check_interval"`
	Timeout       string `yaml:"timeout"`
}

// configuration from a YAML file
func LoadConfig(configPath string) (*Config, error) {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", configPath)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	config.setDefaults()

	return &config, nil
}

// validate ensures the configuration is valid
func (c *Config) validate() error {
	if c.TraefikAPI == "" {
		return fmt.Errorf("traefik_api is required")
	}

	if c.Email == "" {
		return fmt.Errorf("email is required")
	}

	if c.Notification.SMTPHost == "" {
		return fmt.Errorf("notification.smtp_host is required")
	}

	if c.Notification.SMTPPort == 0 {
		return fmt.Errorf("notification.smtp_port is required")
	}

	if len(c.Domains) == 0 {
		return fmt.Errorf("at least one domain configuration is required")
	}

	// Validate each domain
	for i, domain := range c.Domains {
		if domain.Service == "" {
			return fmt.Errorf("domain[%d].service is required", i)
		}
		if domain.Domain == "" {
			return fmt.Errorf("domain[%d].domain is required", i)
		}
	}

	return nil
}

// setDefaults sets default values for optional fields
func (c *Config) setDefaults() {
	if c.ACME.CADirURL == "" {
		c.ACME.CADirURL = "https://acme-v02.api.letsencrypt.org/directory"
	}
	if c.ACME.KeyType == "" {
		c.ACME.KeyType = "RSA2048"
	}
	if c.ACME.Email == "" {
		c.ACME.Email = c.Email
	}

	if c.Certificates.RenewalDays == 0 {
		c.Certificates.RenewalDays = 30
	}
	if c.Certificates.StoragePath == "" {
		c.Certificates.StoragePath = "./certs"
	}

	if c.App.LogLevel == "" {
		c.App.LogLevel = "info"
	}
	if c.App.CheckInterval == "" {
		c.App.CheckInterval = "24h"
	}
	if c.App.Timeout == "" {
		c.App.Timeout = "30s"
	}

	if c.Notification.From == "" {
		c.Notification.From = "noreply@example.com"
	}
}

func (c *Config) GetCheckInterval() (time.Duration, error) {
	return time.ParseDuration(c.App.CheckInterval)
}

func (c *Config) GetTimeout() (time.Duration, error) {
	return time.ParseDuration(c.App.Timeout)
}

func (c *Config) GetCertPath(domain string) string {
	return filepath.Join(c.Certificates.StoragePath, domain+".crt")
}

func (c *Config) GetKeyPath(domain string) string {
	return filepath.Join(c.Certificates.StoragePath, domain+".key")
}

// GetAllDomains returns all configured domains including aliases
func (c *Config) GetAllDomains() []string {
	var domains []string
	for _, domainConfig := range c.Domains {
		domains = append(domains, domainConfig.Domain)
		domains = append(domains, domainConfig.Aliases...)
	}
	return domains
}

func (c *Config) GetDomainForService(serviceName string) (string, bool) {
	for _, domainConfig := range c.Domains {
		if domainConfig.Service == serviceName {
			return domainConfig.Domain, true
		}
	}
	return "", false
}