package certmanager

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
	"fmt"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/O-tero/traefik-cert-manager/internal/config"
)

// MockACMEClient implements a mock ACME client for testing
type MockACMEClient struct {
	mock.Mock
	storagePath string
	logger      *log.Logger
}

func NewMockACMEClient(storagePath string, logger *log.Logger) *MockACMEClient {
	return &MockACMEClient{
		storagePath: storagePath,
		logger:      logger,
	}
}

func (m *MockACMEClient) RequestCertificate(domain string) (*Certificate, error) {
	args := m.Called(domain)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Certificate), args.Error(1)
}

func (m *MockACMEClient) RenewCertificate(cert *Certificate) (*Certificate, error) {
	args := m.Called(cert)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Certificate), args.Error(1)
}

func (m *MockACMEClient) LoadCertificate(domain string) (*Certificate, error) {
	args := m.Called(domain)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Certificate), args.Error(1)
}

// Test helper functions
func createTestCertificate(domain string, validDays int) *Certificate {
	// Generate a private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: domain,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Duration(validDays) * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{domain},
	}

	// Create certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		panic(err)
	}

	// Encode certificate
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	// Encode private key
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	cert := &Certificate{
		Domain:      domain,
		Certificate: certPEM,
		PrivateKey:  keyPEM,
		IssuedAt:    time.Now(),
		ExpiresAt:   time.Now().Add(time.Duration(validDays) * 24 * time.Hour),
	}

	return cert
}

func createTestConfig() *config.Config {
	return &config.Config{
		TraefikAPI: "http://localhost:8080",
		Email:      "test@example.com",
		Domains: []config.Domain{
			{Service: "test-service", Domain: "example.com"},
			{Service: "api-service", Domain: "api.example.com"},
		},
		ACME: config.ACME{
			CADirURL: "https://acme-staging-v02.api.letsencrypt.org/directory",
			KeyType:  "RSA2048",
			Email:    "test@example.com",
		},
		Certificates: config.Certificates{
			RenewalDays: 30,
			StoragePath: "./test-certs",
		},
		App: config.App{
			LogLevel:      "info",
			CheckInterval: "24h",
			Timeout:       "30s",
		},
	}
}

func setupTestDir(t *testing.T) string {
	tempDir, err := os.MkdirTemp("", "cert-manager-test-*")
	require.NoError(t, err)
	
	t.Cleanup(func() {
		os.RemoveAll(tempDir)
	})
	
	return tempDir
}

// Test Certificate struct methods
func TestCertificate_IsExpired(t *testing.T) {
	tests := []struct {
		name     string
		validDays int
		expected bool
	}{
		{
			name:     "valid certificate",
			validDays: 30,
			expected: false,
		},
		{
			name:     "expired certificate",
			validDays: -1,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cert := createTestCertificate("example.com", tt.validDays)
			assert.Equal(t, tt.expected, cert.IsExpired())
		})
	}
}

func TestCertificate_NeedsRenewal(t *testing.T) {
	tests := []struct {
		name        string
		validDays   int
		renewalDays int
		expected    bool
	}{
		{
			name:        "needs renewal",
			validDays:   15,
			renewalDays: 30,
			expected:    true,
		},
		{
			name:        "does not need renewal",
			validDays:   60,
			renewalDays: 30,
			expected:    false,
		},
		{
			name:        "exactly at renewal threshold",
			validDays:   30,
			renewalDays: 30,
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cert := createTestCertificate("example.com", tt.validDays)
			assert.Equal(t, tt.expected, cert.NeedsRenewal(tt.renewalDays))
		})
	}
}

func TestCertificate_DaysUntilExpiry(t *testing.T) {
	cert := createTestCertificate("example.com", 30)
	days := cert.DaysUntilExpiry()
	
	// Should be approximately 30 days (allowing for test execution time)
	assert.Greater(t, days, 29)
	assert.Less(t, days, 31)
}

func TestCertificate_ParseCertificate(t *testing.T) {
	cert := createTestCertificate("example.com", 30)
	
	// Clear the ExpiresAt field to test parsing
	cert.ExpiresAt = time.Time{}
	
	err := cert.parseCertificate()
	require.NoError(t, err)
	
	assert.False(t, cert.ExpiresAt.IsZero())
	assert.True(t, cert.ExpiresAt.After(time.Now()))
}

func TestCertificate_ParseCertificate_InvalidPEM(t *testing.T) {
	cert := &Certificate{
		Domain:      "example.com",
		Certificate: []byte("invalid pem data"),
	}
	
	err := cert.parseCertificate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse certificate PEM")
}

// Test CertificateManager
func TestNewCertificateManager(t *testing.T) {
	testDir := setupTestDir(t)
	cfg := createTestConfig()
	cfg.Certificates.StoragePath = testDir
	
	// Note: This would normally create a real ACME client
	// In a real test environment, we'd need to mock the ACME client creation
	t.Skip("Skipping test that requires real ACME client - would need dependency injection")
}

func TestCertificateManager_RequestCertificate_WithMock(t *testing.T) {
	testDir := setupTestDir(t)
	cfg := createTestConfig()
	cfg.Certificates.StoragePath = testDir
	
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	
	// Create mock ACME client
	mockClient := NewMockACMEClient(testDir, logger)
	
	// Create certificate manager with mock client
	cm := &CertificateManager{
		config:     cfg,
		acmeClient: mockClient,
		logger:     logger,
		certs:      make(map[string]*Certificate),
	}
	
	// Setup mock expectations
	testCert := createTestCertificate("example.com", 90)
	mockClient.On("RequestCertificate", "example.com").Return(testCert, nil)
	
	// Test certificate request
	err := cm.RequestCertificate("example.com")
	require.NoError(t, err)
	
	// Verify certificate was stored
	cert, err := cm.GetCertificate("example.com")
	require.NoError(t, err)
	assert.Equal(t, "example.com", cert.Domain)
	
	// Verify mock was called
	mockClient.AssertExpectations(t)
}

func TestCertificateManager_RequestCertificate_SkipValid(t *testing.T) {
	testDir := setupTestDir(t)
	cfg := createTestConfig()
	cfg.Certificates.StoragePath = testDir
	
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	
	// Create mock ACME client
	mockClient := NewMockACMEClient(testDir, logger)
	
	// Create certificate manager with mock client
	cm := &CertificateManager{
		config:     cfg,
		acmeClient: mockClient,
		logger:     logger,
		certs:      make(map[string]*Certificate),
	}
	
	// Add a valid certificate
	validCert := createTestCertificate("example.com", 60)
	cm.certs["example.com"] = validCert
	
	// Test certificate request (should skip)
	err := cm.RequestCertificate("example.com")
	require.NoError(t, err)
	
	// Verify mock was not called (since certificate is valid)
	mockClient.AssertNotCalled(t, "RequestCertificate")
}

func TestCertificateManager_RenewCertificate(t *testing.T) {
	testDir := setupTestDir(t)
	cfg := createTestConfig()
	cfg.Certificates.StoragePath = testDir
	
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	
	// Create mock ACME client
	mockClient := NewMockACMEClient(testDir, logger)
	
	// Create certificate manager with mock client
	cm := &CertificateManager{
		config:     cfg,
		acmeClient: mockClient,
		logger:     logger,
		certs:      make(map[string]*Certificate),
	}
	
	// Add an expiring certificate
	oldCert := createTestCertificate("example.com", 15)
	cm.certs["example.com"] = oldCert
	
	// Setup mock expectations
	newCert := createTestCertificate("example.com", 90)
	mockClient.On("RenewCertificate", oldCert).Return(newCert, nil)
	
	// Test certificate renewal
	err := cm.RenewCertificate("example.com")
	require.NoError(t, err)
	
	// Verify certificate was updated
	cert, err := cm.GetCertificate("example.com")
	require.NoError(t, err)
	assert.Equal(t, newCert.ExpiresAt, cert.ExpiresAt)
	
	// Verify mock was called
	mockClient.AssertExpectations(t)
}

func TestCertificateManager_CheckCertificateHealth(t *testing.T) {
	testDir := setupTestDir(t)
	cfg := createTestConfig()
	cfg.Certificates.StoragePath = testDir
	
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	
	// Create certificate manager
	cm := &CertificateManager{
		config:     cfg,
		acmeClient: nil, // Not needed for this test
		logger:     logger,
		certs:      make(map[string]*Certificate),
	}
	
	// Add certificates with different statuses
	validCert := createTestCertificate("valid.com", 60)
	renewalCert := createTestCertificate("renewal.com", 15)
	expiredCert := createTestCertificate("expired.com", -5)
	
	cm.certs["valid.com"] = validCert
	cm.certs["renewal.com"] = renewalCert
	cm.certs["expired.com"] = expiredCert
	
	// Check health
	health := cm.CheckCertificateHealth()
	
	// Verify health statuses
	assert.Equal(t, "valid", health["valid.com"].Status)
	assert.Equal(t, "needs_renewal", health["renewal.com"].Status)
	assert.Equal(t, "expired", health["expired.com"].Status)
	
	// Verify boolean flags
	assert.False(t, health["valid.com"].IsExpired)
	assert.False(t, health["valid.com"].NeedsRenewal)
	
	assert.False(t, health["renewal.com"].IsExpired)
	assert.True(t, health["renewal.com"].NeedsRenewal)
	
	assert.True(t, health["expired.com"].IsExpired)
	assert.True(t, health["expired.com"].NeedsRenewal)
}

func TestCertificateManager_ListCertificates(t *testing.T) {
	testDir := setupTestDir(t)
	cfg := createTestConfig()
	cfg.Certificates.StoragePath = testDir
	
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	
	// Create certificate manager
	cm := &CertificateManager{
		config:     cfg,
		acmeClient: nil,
		logger:     logger,
		certs:      make(map[string]*Certificate),
	}
	
	// Add test certificates
	cert1 := createTestCertificate("example.com", 60)
	cert2 := createTestCertificate("api.example.com", 30)
	
	cm.certs["example.com"] = cert1
	cm.certs["api.example.com"] = cert2
	
	// List certificates
	certs := cm.ListCertificates()
	
	// Verify results
	assert.Len(t, certs, 2)
	assert.Contains(t, certs, "example.com")
	assert.Contains(t, certs, "api.example.com")
	
	// Verify it's a copy (modifying returned map shouldn't affect original)
	delete(certs, "example.com")
	assert.Len(t, cm.certs, 2) // Original should still have 2 certificates
}

func TestCertificateManager_GetCertificatePaths(t *testing.T) {
	testDir := setupTestDir(t)
	cfg := createTestConfig()
	cfg.Certificates.StoragePath = testDir
	
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	
	// Create certificate manager
	cm := &CertificateManager{
		config:     cfg,
		acmeClient: nil,
		logger:     logger,
		certs:      make(map[string]*Certificate),
	}
	
	// Get certificate paths
	certPath, keyPath := cm.GetCertificatePaths("example.com")
	
	// Verify paths
	expectedCertPath := filepath.Join(testDir, "example.com.crt")
	expectedKeyPath := filepath.Join(testDir, "example.com.key")
	
	assert.Equal(t, expectedCertPath, certPath)
	assert.Equal(t, expectedKeyPath, keyPath)
}

func TestCertificateManager_Cleanup(t *testing.T) {
	testDir := setupTestDir(t)
	cfg := createTestConfig()
	cfg.Certificates.StoragePath = testDir
	
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	
	// Create certificate manager
	cm := &CertificateManager{
		config:     cfg,
		acmeClient: nil,
		logger:     logger,
		certs:      make(map[string]*Certificate),
	}
	
	// Add certificates
	validCert := createTestCertificate("valid.com", 60)
	recentlyExpiredCert := createTestCertificate("recent.com", -5)
	oldExpiredCert := createTestCertificate("old.com", -40)
	
	cm.certs["valid.com"] = validCert
	cm.certs["recent.com"] = recentlyExpiredCert
	cm.certs["old.com"] = oldExpiredCert
	
	// Run cleanup
	err := cm.Cleanup()
	require.NoError(t, err)
	
	// Verify cleanup results
	assert.Len(t, cm.certs, 2) // Should keep valid and recently expired
	assert.Contains(t, cm.certs, "valid.com")
	assert.Contains(t, cm.certs, "recent.com")
	assert.NotContains(t, cm.certs, "old.com")
}

// Benchmark tests
func BenchmarkCertificate_IsExpired(b *testing.B) {
	cert := createTestCertificate("example.com", 30)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cert.IsExpired()
	}
}

func BenchmarkCertificate_NeedsRenewal(b *testing.B) {
	cert := createTestCertificate("example.com", 30)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cert.NeedsRenewal(30)
	}
}

func BenchmarkCertificateManager_CheckCertificateHealth(b *testing.B) {
	testDir := setupTestDir(&testing.T{})
	defer os.RemoveAll(testDir)
	
	cfg := createTestConfig()
	cfg.Certificates.StoragePath = testDir
	
	logger := log.New(os.Stdout, "[BENCH] ", log.LstdFlags)
	
	// Create certificate manager
	cm := &CertificateManager{
		config:     cfg,
		acmeClient: nil,
		logger:     logger,
		certs:      make(map[string]*Certificate),
	}
	
	// Add many certificates
	for i := 0; i < 100; i++ {
		domain := fmt.Sprintf("example%d.com", i)
		cert := createTestCertificate(domain, 30+i)
		cm.certs[domain] = cert
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cm.CheckCertificateHealth()
	}
}