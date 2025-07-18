package traefik

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Service represents a Traefik service
type Service struct {
	Name   string            `json:"name"`
	Type   string            `json:"type"`
	Status string            `json:"status"`
	Tags   []string          `json:"tags"`
	Health string            `json:"health"`
	Props  map[string]string `json:"props"`
}

type Router struct {
	Name        string   `json:"name"`
	Status      string   `json:"status"`
	Using       []string `json:"using"`
	Rule        string   `json:"rule"`
	Priority    int      `json:"priority"`
	EntryPoints []string `json:"entryPoints"`
	Service     string   `json:"service"`
	TLS         *TLS     `json:"tls,omitempty"`
}

type TLS struct {
	Passthrough bool `json:"passthrough"`
}

// APIClient handles communication with Traefik API
type APIClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewAPIClient creates a new Traefik API client
func NewAPIClient(baseURL string, timeout time.Duration) *APIClient {
	return &APIClient{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// GetServices retrieves all services from Traefik API
func (c *APIClient) GetServices(ctx context.Context) ([]string, error) {
	services, err := c.getServicesDetailed(ctx)
	if err != nil {
		return nil, err
	}

	var serviceNames []string
	for _, service := range services {
		serviceNames = append(serviceNames, service.Name)
	}

	return serviceNames, nil
}

// GetServicesDetailed retrieves detailed service information from Traefik API
func (c *APIClient) getServicesDetailed(ctx context.Context) ([]Service, error) {
	url := fmt.Sprintf("%s/http/services", c.baseURL)
	
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Traefik API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var services []Service
	if err := json.NewDecoder(resp.Body).Decode(&services); err != nil {
		return nil, fmt.Errorf("failed to decode services response: %w", err)
	}

	return services, nil
}

// GetRouters retrieves all routers from Traefik API
func (c *APIClient) GetRouters(ctx context.Context) ([]Router, error) {
	url := fmt.Sprintf("%s/http/routers", c.baseURL)
	
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Traefik API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var routers []Router
	if err := json.NewDecoder(resp.Body).Decode(&routers); err != nil {
		return nil, fmt.Errorf("failed to decode routers response: %w", err)
	}

	return routers, nil
}

// GetServicesByDomain returns services that handle specific domains
func (c *APIClient) GetServicesByDomain(ctx context.Context, domains []string) (map[string][]string, error) {
	routers, err := c.GetRouters(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get routers: %w", err)
	}

	domainToServices := make(map[string][]string)
	
	for _, router := range routers {
		for _, domain := range domains {
			if c.routerMatchesDomain(router, domain) {
				domainToServices[domain] = append(domainToServices[domain], router.Service)
			}
		}
	}

	return domainToServices, nil
}

func (c *APIClient) routerMatchesDomain(router Router, domain string) bool {
	//  Reminder: do more sophisticated rule parsing
	rule := strings.ToLower(router.Rule)
	domain = strings.ToLower(domain)
	
	if strings.Contains(rule, fmt.Sprintf("host(`%s`)", domain)) {
		return true
	}
	
	if strings.Contains(rule, fmt.Sprintf("hostregexp(`%s`)", domain)) {
		return true
	}
	
	// Check for domain in Host rule with multiple domains
	if strings.Contains(rule, "host(") && strings.Contains(rule, domain) {
		return true
	}
	
	return false
}

// IsHealthy checks if Traefik API is accessible
func (c *APIClient) IsHealthy(ctx context.Context) error {
	url := fmt.Sprintf("%s/ping", c.baseURL)
	
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to perform health check: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status %d", resp.StatusCode)
	}

	return nil
}

// GetServiceHealth returns health status of a specific service
func (c *APIClient) GetServiceHealth(ctx context.Context, serviceName string) (string, error) {
	services, err := c.getServicesDetailed(ctx)
	if err != nil {
		return "", err
	}

	for _, service := range services {
		if service.Name == serviceName {
			return service.Health, nil
		}
	}

	return "", fmt.Errorf("service %s not found", serviceName)
}

// Legacy function for backward compatibility
func GetServices(apiURL string) ([]string, error) {
	client := NewAPIClient(apiURL, 30*time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	return client.GetServices(ctx)
}