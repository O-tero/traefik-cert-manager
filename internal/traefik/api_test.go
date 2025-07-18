package traefik

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewAPIClient(t *testing.T) {
	client := NewAPIClient("http://localhost:8080/api", 30*time.Second)
	
	if client.baseURL != "http://localhost:8080/api" {
		t.Errorf("Expected baseURL to be 'http://localhost:8080/api', got '%s'", client.baseURL)
	}
	
	if client.httpClient.Timeout != 30*time.Second {
		t.Errorf("Expected timeout to be 30s, got %v", client.httpClient.Timeout)
	}
}

func TestAPIClient_GetServices(t *testing.T) {
	mockServices := []Service{
		{
			Name:   "service1@docker",
			Type:   "loadbalancer",
			Status: "enabled",
			Health: "healthy",
		},
		{
			Name:   "service2@docker",
			Type:   "loadbalancer",
			Status: "enabled",
			Health: "healthy",
		},
	}

	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/http/services" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockServices)
		} else {
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, 30*time.Second)
	ctx := context.Background()

	services, err := client.GetServices(ctx)
	if err != nil {
		t.Fatalf("Failed to get services: %v", err)
	}

	expectedServices := []string{"service1@docker", "service2@docker"}
	if len(services) != len(expectedServices) {
		t.Errorf("Expected %d services, got %d", len(expectedServices), len(services))
	}

	for i, service := range services {
		if service != expectedServices[i] {
			t.Errorf("Expected service '%s', got '%s'", expectedServices[i], service)
		}
	}
}

func TestAPIClient_GetRouters(t *testing.T) {
	mockRouters := []Router{
		{
			Name:        "router1@docker",
			Status:      "enabled",
			Rule:        "Host(`example.com`)",
			Priority:    1,
			EntryPoints: []string{"web"},
			Service:     "service1@docker",
		},
		{
			Name:        "router2@docker",
			Status:      "enabled",
			Rule:        "Host(`api.example.com`)",
			Priority:    1,
			EntryPoints: []string{"web"},
			Service:     "service2@docker",
		},
	}

	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/http/routers" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockRouters)
		} else {
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, 30*time.Second)
	ctx := context.Background()

	routers, err := client.GetRouters(ctx)
	if err != nil {
		t.Fatalf("Failed to get routers: %v", err)
	}

	if len(routers) != 2 {
		t.Errorf("Expected 2 routers, got %d", len(routers))
	}

	if routers[0].Name != "router1@docker" {
		t.Errorf("Expected first router name 'router1@docker', got '%s'", routers[0].Name)
	}

	if routers[0].Rule != "Host(`example.com`)" {
		t.Errorf("Expected first router rule 'Host(`example.com`)', got '%s'", routers[0].Rule)
	}

	if routers[0].Service != "service1@docker" {
		t.Errorf("Expected first router service 'service1@docker', got '%s'", routers[0].Service)
	}
}

func TestAPIClient_GetServicesByDomain(t *testing.T) {
	// Mock routers response
	mockRouters := []Router{
		{
			Name:        "router1@docker",
			Status:      "enabled",
			Rule:        "Host(`example.com`)",
			Service:     "service1@docker",
		},
		{
			Name:        "router2@docker",
			Status:      "enabled",
			Rule:        "Host(`api.example.com`)",
			Service:     "service2@docker",
		},
		{
			Name:        "router3@docker",
			Status:      "enabled",
			Rule:        "Host(`example.com`) && PathPrefix(`/api`)",
			Service:     "service3@docker",
		},
	}

	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/http/routers" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockRouters)
		} else {
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, 30*time.Second)
	ctx := context.Background()

	domains := []string{"example.com", "api.example.com"}
	servicesByDomain, err := client.GetServicesByDomain(ctx, domains)
	if err != nil {
		t.Fatalf("Failed to get services by domain: %v", err)
	}

	// Check example.com services
	exampleServices := servicesByDomain["example.com"]
	if len(exampleServices) != 2 {
		t.Errorf("Expected 2 services for example.com, got %d", len(exampleServices))
	}

	expectedServices := []string{"service1@docker", "service3@docker"}
	for _, expected := range expectedServices {
		found := false
		for _, actual := range exampleServices {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected service '%s' for example.com", expected)
		}
	}

	// Check api.example.com services
	apiServices := servicesByDomain["api.example.com"]
	if len(apiServices) != 1 {
		t.Errorf("Expected 1 service for api.example.com, got %d", len(apiServices))
	}

	if apiServices[0] != "service2@docker" {
		t.Errorf("Expected service 'service2@docker' for api.example.com, got '%s'", apiServices[0])
	}
}

func TestAPIClient_IsHealthy(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ping" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		} else {
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, 30*time.Second)
	ctx := context.Background()

	err := client.IsHealthy(ctx)
	if err != nil {
		t.Errorf("Expected healthy check to pass, got error: %v", err)
	}
}

func TestAPIClient_IsHealthy_Unhealthy(t *testing.T) {
	// Create mock server that returns 500
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ping" {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal Server Error"))
		} else {
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, 30*time.Second)
	ctx := context.Background()

	err := client.IsHealthy(ctx)
	if err == nil {
		t.Error("Expected health check to fail")
	}

	if !strings.Contains(err.Error(), "health check failed with status 500") {
		t.Errorf("Expected health check error with status 500, got: %v", err)
	}
}

func TestAPIClient_GetServiceHealth(t *testing.T) {
	// Mock services response
	mockServices := []Service{
		{
			Name:   "service1@docker",
			Type:   "loadbalancer",
			Status: "enabled",
			Health: "healthy",
		},
		{
			Name:   "service2@docker",
			Type:   "loadbalancer",
			Status: "enabled",
			Health: "unhealthy",
		},
	}

	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/http/services" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockServices)
		} else {
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, 30*time.Second)
	ctx := context.Background()

	// Test existing service
	health, err := client.GetServiceHealth(ctx, "service1@docker")
	if err != nil {
		t.Fatalf("Failed to get service health: %v", err)
	}

	if health != "healthy" {
		t.Errorf("Expected health 'healthy', got '%s'", health)
	}

	// Test non-existing service
	_, err = client.GetServiceHealth(ctx, "nonexistent@docker")
	if err == nil {
		t.Error("Expected error for non-existent service")
	}

	if !strings.Contains(err.Error(), "service nonexistent@docker not found") {
		t.Errorf("Expected 'service not found' error, got: %v", err)
	}
}

func TestRouterMatchesDomain(t *testing.T) {
	client := &APIClient{}

	tests := []struct {
		name     string
		router   Router
		domain   string
		expected bool
	}{
		{
			name:     "exact host match",
			router:   Router{Rule: "Host(`example.com`)"},
			domain:   "example.com",
			expected: true,
		},
		{
			name:     "host with path prefix",
			router:   Router{Rule: "Host(`example.com`) && PathPrefix(`/api`)"},
			domain:   "example.com",
			expected: true,
		},
		{
			name:     "hostregexp match",
			router:   Router{Rule: "HostRegexp(`example.com`)"},
			domain:   "example.com",
			expected: true,
		},
		{
			name:     "no match",
			router:   Router{Rule: "Host(`other.com`)"},
			domain:   "example.com",
			expected: false,
		},
		{
			name:     "case insensitive match",
			router:   Router{Rule: "Host(`EXAMPLE.COM`)"},
			domain:   "example.com",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.routerMatchesDomain(tt.router, tt.domain)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for router rule '%s' and domain '%s'", 
					tt.expected, result, tt.router.Rule, tt.domain)
			}
		})
	}
}

func TestAPIClient_ErrorHandling(t *testing.T) {
	// Test with non-existent server
	client := NewAPIClient("http://nonexistent:8080/api", 1*time.Second)
	ctx := context.Background()

	_, err := client.GetServices(ctx)
	if err == nil {
		t.Error("Expected error for non-existent server")
	}

	// Test with server returning 404
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer server.Close()

	client = NewAPIClient(server.URL, 30*time.Second)

	_, err = client.GetServices(ctx)
	if err == nil {
		t.Error("Expected error for 404 response")
	}

	if !strings.Contains(err.Error(), "API returned status 404") {
		t.Errorf("Expected 404 error, got: %v", err)
	}
}

func TestAPIClient_Timeout(t *testing.T) {
	// Create slow server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]Service{})
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, 50*time.Millisecond)
	ctx := context.Background()

	_, err := client.GetServices(ctx)
	if err == nil {
		t.Error("Expected timeout error")
	}

	if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}

func TestGetServices_LegacyFunction(t *testing.T) {
	// Mock services response
	mockServices := []Service{
		{Name: "service1@docker"},
		{Name: "service2@docker"},
	}

	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/http/services" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockServices)
		} else {
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	services, err := GetServices(server.URL)
	if err != nil {
		t.Fatalf("Failed to get services: %v", err)
	}

	expectedServices := []string{"service1@docker", "service2@docker"}
	if len(services) != len(expectedServices) {
		t.Errorf("Expected %d services, got %d", len(expectedServices), len(services))
	}

	for i, service := range services {
		if service != expectedServices[i] {
			t.Errorf("Expected service '%s', got '%s'", expectedServices[i], service)
		}
	}
}