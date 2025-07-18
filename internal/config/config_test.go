package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	configContent := `
traefik_api: "http://localhost:8080/api"
email: "test@example.com"
notification:
  smtp_host: "smtp.test.com"
  smtp_port: 587
  username: "user"
  password: "pass"
  from: "noreply@test.com"
domains:
  - service: "web"
    domain: "example.com"
    aliases: ["www.example.com"]
  - service: "api"
    domain: "api.example.com"
acme:
  ca_dir_url: "https://acme-staging-v02.api.letsencrypt.org/directory"
  key_type: "RSA2048"
  email: "acme@example.com"
certificates:
  renewal_days: 14
  storage_path: "/tmp/certs"
app:
  log_level: "debug"
  check_interval: "12h"
  timeout: "60s"
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if config.TraefikAPI != "http://localhost:8080/api" {
		t.Errorf("Expected TraefikAPI to be 'http://localhost:8080/api', got '%s'", config.TraefikAPI)
	}

	if config.Email != "test@example.com" {
		t.Errorf("Expected Email to be 'test@example.com', got '%s'", config.Email)
	}

	if config.Notification.SMTPHost != "smtp.test.com" {
		t.Errorf("Expected SMTPHost to be 'smtp.test.com', got '%s'", config.Notification.SMTPHost)
	}

	if config.Notification.SMTPPort != 587 {
		t.Errorf("Expected SMTPPort to be 587, got %d", config.Notification.SMTPPort)
	}

	if len(config.Domains) != 2 {
		t.Errorf("Expected 2 domains, got %d", len(config.Domains))
	}

	if config.Domains[0].Service != "web" {
		t.Errorf("Expected first domain service to be 'web', got '%s'", config.Domains[0].Service)
	}

	if config.Domains[0].Domain != "example.com" {
		t.Errorf("Expected first domain to be 'example.com', got '%s'", config.Domains[0].Domain)
	}

	if len(config.Domains[0].Aliases) != 1 || config.Domains[0].Aliases[0] != "www.example.com" {
		t.Errorf("Expected first domain aliases to be ['www.example.com'], got %v", config.Domains[0].Aliases)
	}

	if config.ACME.CADirURL != "https://acme-staging-v02.api.letsencrypt.org/directory" {
		t.Errorf("Expected ACME CADirURL to be staging URL, got '%s'", config.ACME.CADirURL)
	}

	if config.ACME.Email != "acme@example.com" {
		t.Errorf("Expected ACME Email to be 'acme@example.com', got '%s'", config.ACME.Email)
	}

	if config.Certificates.RenewalDays != 14 {
		t.Errorf("Expected RenewalDays to be 14, got %d", config.Certificates.RenewalDays)
	}

	if config.Certificates.StoragePath != "/tmp/certs" {
		t.Errorf("Expected StoragePath to be '/tmp/certs', got '%s'", config.Certificates.StoragePath)
	}

	if config.App.LogLevel != "debug" {
		t.Errorf("Expected LogLevel to be 'debug', got '%s'", config.App.LogLevel)
	}

	if config.App.CheckInterval != "12h" {
		t.Errorf("Expected CheckInterval to be '12h', got '%s'", config.App.CheckInterval)
	}
}

func TestLoadConfigWithDefaults(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	// Minimal config to test defaults
	configContent := `
traefik_api: "http://localhost:8080/api"
email: "test@example.com"
notification:
  smtp_host: "smtp.test.com"
  smtp_port: 587
domains:
  - service: "web"
    domain: "example.com"
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Test defaults
	if config.ACME.CADirURL != "https://acme-v02.api.letsencrypt.org/directory" {
		t.Errorf("Expected default ACME CADirURL, got '%s'", config.ACME.CADirURL)
	}

	if config.ACME.KeyType != "RSA2048" {
		t.Errorf("Expected default KeyType to be 'RSA2048', got '%s'", config.ACME.KeyType)
	}

	if config.ACME.Email != "test@example.com" {
		t.Errorf("Expected ACME Email to default to main email, got '%s'", config.ACME.Email)
	}

	if config.Certificates.RenewalDays != 30 {
		t.Errorf("Expected default RenewalDays to be 30, got %d", config.Certificates.RenewalDays)
	}

	if config.Certificates.StoragePath != "./certs" {
		t.Errorf("Expected default StoragePath to be './certs', got '%s'", config.Certificates.StoragePath)
	}

	if config.App.LogLevel != "info" {
		t.Errorf("Expected default LogLevel to be 'info', got '%s'", config.App.LogLevel)
	}

	if config.App.CheckInterval != "24h" {
		t.Errorf("Expected default CheckInterval to be '24h', got '%s'", config.App.CheckInterval)
	}

	if config.Notification.From != "noreply@example.com" {
		t.Errorf("Expected default From to be 'noreply@example.com', got '%s'", config.Notification.From)
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name           string
		config         Config
		expectedError  string
	}{
		{
			name: "missing traefik_api",
			config: Config{
				Email: "test@example.com",
				Notification: Notification{SMTPHost: "smtp.test.com", SMTPPort: 587},
				Domains: []Domain{{Service: "web", Domain: "example.com"}},
			},
			expectedError: "traefik_api is required",
		},
		{
			name: "missing email",
			config: Config{
				TraefikAPI: "http://localhost:8080/api",
				Notification: Notification{SMTPHost: "smtp.test.com", SMTPPort: 587},
				Domains: []Domain{{Service: "web", Domain: "example.com"}},
			},
			expectedError: "email is required",
		},
		{
			name: "missing smtp_host",
			config: Config{
				TraefikAPI: "http://localhost:8080/api",
				Email: "test@example.com",
				Notification: Notification{SMTPPort: 587},
				Domains: []Domain{{Service: "web", Domain: "example.com"}},
			},
			expectedError: "notification.smtp_host is required",
		},
		{
			name: "missing smtp_port",
			config: Config{
				TraefikAPI: "http://localhost:8080/api",
				Email: "test@example.com",
				Notification: Notification{SMTPHost: "smtp.test.com"},
				Domains: []Domain{{Service: "web", Domain: "example.com"}},
			},
			expectedError: "notification.smtp_port is required",
		},
		{
			name: "no domains",
			config: Config{
				TraefikAPI: "http://localhost:8080/api",
				Email: "test@example.com",
				Notification: Notification{SMTPHost: "smtp.test.com", SMTPPort: 587},
				Domains: []Domain{},
			},
			expectedError: "at least one domain configuration is required",
		},
		{
			name: "domain missing service",
			config: Config{
				TraefikAPI: "http://localhost:8080/api",
				Email: "test@example.com",
				Notification: Notification{SMTPHost: "smtp.test.com", SMTPPort: 587},
				Domains: []Domain{{Domain: "example.com"}},
			},
			expectedError: "domain[0].service is required",
		},
		{
			name: "domain missing domain",
			config: Config{
				TraefikAPI: "http://localhost:8080/api",
				Email: "test@example.com",
				Notification: Notification{SMTPHost: "smtp.test.com", SMTPPort: 587},
				Domains: []Domain{{Service: "web"}},
			},
			expectedError: "domain[0].domain is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.validate()
			if err == nil {
				t.Errorf("Expected validation error, got nil")
			} else if err.Error() != tt.expectedError {
				t.Errorf("Expected error '%s', got '%s'", tt.expectedError, err.Error())
			}
		})
	}
}

func TestConfigHelperMethods(t *testing.T) {
	config := &Config{
		Certificates: Certificates{
			StoragePath: "/tmp/certs",
		},
		Domains: []Domain{
			{Service: "web", Domain: "example.com", Aliases: []string{"www.example.com"}},
			{Service: "api", Domain: "api.example.com"},
		},
		App: App{
			CheckInterval: "12h",
			Timeout: "30s",
		},
	}

	certPath := config.GetCertPath("example.com")
	expected := "/tmp/certs/example.com.crt"
	if certPath != expected {
		t.Errorf("Expected cert path '%s', got '%s'", expected, certPath)
	}

	keyPath := config.GetKeyPath("example.com")
	expected = "/tmp/certs/example.com.key"
	if keyPath != expected {
		t.Errorf("Expected key path '%s', got '%s'", expected, keyPath)
	}

	domains := config.GetAllDomains()
	expectedDomains := []string{"example.com", "www.example.com", "api.example.com"}
	if len(domains) != len(expectedDomains) {
		t.Errorf("Expected %d domains, got %d", len(expectedDomains), len(domains))
	}

	domain, found := config.GetDomainForService("web")
	if !found {
		t.Error("Expected to find domain for service 'web'")
	}
	if domain != "example.com" {
		t.Errorf("Expected domain 'example.com', got '%s'", domain)
	}

	_, found = config.GetDomainForService("nonexistent")
	if found {
		t.Error("Expected not to find domain for nonexistent service")
	}

	interval, err := config.GetCheckInterval()
	if err != nil {
		t.Errorf("Failed to parse check interval: %v", err)
	}
	if interval != 12*time.Hour {
		t.Errorf("Expected check interval 12h, got %v", interval)
	}

	timeout, err := config.GetTimeout()
	if err != nil {
		t.Errorf("Failed to parse timeout: %v", err)
	}
	if timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", timeout)
	}
}

func TestLoadConfigFileNotFound(t *testing.T) {
	_, err := LoadConfig("/nonexistent/config.yaml")
	if err == nil {
		t.Error("Expected error for non-existent config file")
	}
	if !strings.Contains(err.Error(), "config file not found") {
		t.Errorf("Expected 'config file not found' error, got: %v", err)
	}
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	// Invalid YAML content
	invalidYAML := `
traefik_api: "http://localhost:8080/api"
email: "test@example.com"
invalid_yaml: [
`

	err := os.WriteFile(configPath, []byte(invalidYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	_, err = LoadConfig(configPath)
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
	if !strings.Contains(err.Error(), "failed to parse config file") {
		t.Errorf("Expected 'failed to parse config file' error, got: %v", err)
	}
}