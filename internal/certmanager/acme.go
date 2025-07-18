package certmanager

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge/http01"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
)

type ACMEUser struct {
	Email        string
	Registration *registration.Resource
	key          crypto.PrivateKey
}

func (u *ACMEUser) GetEmail() string {
	return u.Email
}

func (u *ACMEUser) GetRegistration() *registration.Resource {
	return u.Registration
}

func (u *ACMEUser) GetPrivateKey() crypto.PrivateKey {
	return u.key
}

// ACMEClient handles ACME operations
type ACMEClient struct {
	client      *lego.Client
	user        *ACMEUser
	storagePath string
	logger      *log.Logger
}

// ACMEConfig holds configuration for ACME client
type ACMEConfig struct {
	CADirURL    string
	Email       string
	KeyType     string
	StoragePath string
	Logger      *log.Logger
}

func NewACMEClient(config ACMEConfig) (*ACMEClient, error) {
	if config.Logger == nil {
		config.Logger = log.New(os.Stdout, "[ACME] ", log.LstdFlags)
	}

	// Create user with private key
	privateKey, err := generatePrivateKey(config.KeyType)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	user := &ACMEUser{
		Email: config.Email,
		key:   privateKey,
	}

	// Create lego config
	legoConfig := lego.NewConfig(user)
	legoConfig.CADirURL = config.CADirURL
	legoConfig.Certificate.KeyType = getKeyType(config.KeyType)

	// Create client
	client, err := lego.NewClient(legoConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create lego client: %w", err)
	}

	// Set up HTTP challenge solver
	err = client.Challenge.SetHTTP01Provider(http01.NewProviderServer("", "5002"))
	if err != nil {
		return nil, fmt.Errorf("failed to set HTTP01 provider: %w", err)
	}

	acmeClient := &ACMEClient{
		client:      client,
		user:        user,
		storagePath: config.StoragePath,
		logger:      config.Logger,
	}

	if err := acmeClient.registerUser(); err != nil {
		return nil, fmt.Errorf("failed to register user: %w", err)
	}

	return acmeClient, nil
}

// registerUser registers the user with ACME server
func (c *ACMEClient) registerUser() error {
	reg, err := c.client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
	if err != nil {
		return fmt.Errorf("failed to register: %w", err)
	}

	c.user.Registration = reg
	c.logger.Printf("User registered successfully with URI: %s", reg.URI)

	return nil
}

func (c *ACMEClient) RequestCertificate(domain string) (*Certificate, error) {
	c.logger.Printf("Requesting certificate for domain: %s", domain)

	// Ensure storage directory exists
	if err := os.MkdirAll(c.storagePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	// Request certificate
	request := certificate.ObtainRequest{
		Domains: []string{domain},
		Bundle:  true,
	}

	certificates, err := c.client.Certificate.Obtain(request)
	if err != nil {
		c.logger.Printf("Failed to obtain certificate for %s: %v", domain, err)
		return nil, fmt.Errorf("failed to obtain certificate: %w", err)
	}

	c.logger.Printf("Successfully obtained certificate for %s", domain)

	// Create certificate object
	cert := &Certificate{
		Domain:      domain,
		Certificate: certificates.Certificate,
		PrivateKey:  certificates.PrivateKey,
		IssuerCert:  certificates.IssuerCertificate,
		URL:         certificates.CertURL,
		IssuedAt:    time.Now(),
	}

	// Parse certificate to get expiry
	if err := cert.parseCertificate(); err != nil {
		c.logger.Printf("Warning: failed to parse certificate: %v", err)
	}

	// Save certificate to disk
	if err := c.saveCertificate(cert); err != nil {
		return nil, fmt.Errorf("failed to save certificate: %w", err)
	}

	c.logger.Printf("Certificate saved successfully for %s", domain)
	return cert, nil
}

func (c *ACMEClient) RenewCertificate(cert *Certificate) (*Certificate, error) {
	c.logger.Printf("Renewing certificate for domain: %s", cert.Domain)

	certResource := &certificate.Resource{
		Domain:      cert.Domain,
		Certificate: cert.Certificate,
		PrivateKey:  cert.PrivateKey,
		CertURL:     cert.URL,
	}

	// Renew certificate
	renewedCert, err := c.client.Certificate.Renew(*certResource, true, false, "")
	if err != nil {
		c.logger.Printf("Failed to renew certificate for %s: %v", cert.Domain, err)
		return nil, fmt.Errorf("failed to renew certificate: %w", err)
	}

	c.logger.Printf("Successfully renewed certificate for %s", cert.Domain)

	// Create new certificate object
	newCert := &Certificate{
		Domain:      cert.Domain,
		Certificate: renewedCert.Certificate,
		PrivateKey:  renewedCert.PrivateKey,
		IssuerCert:  renewedCert.IssuerCertificate,
		URL:         renewedCert.CertURL,
		IssuedAt:    time.Now(),
	}

	if err := newCert.parseCertificate(); err != nil {
		c.logger.Printf("Warning: failed to parse renewed certificate: %v", err)
	}

	if err := c.saveCertificate(newCert); err != nil {
		return nil, fmt.Errorf("failed to save renewed certificate: %w", err)
	}

	c.logger.Printf("Renewed certificate saved successfully for %s", cert.Domain)
	return newCert, nil
}

func (c *ACMEClient) saveCertificate(cert *Certificate) error {
	// Save certificate
	certPath := filepath.Join(c.storagePath, cert.Domain+".crt")
	if err := os.WriteFile(certPath, cert.Certificate, 0644); err != nil {
		return fmt.Errorf("failed to save certificate file: %w", err)
	}

	// Save private key
	keyPath := filepath.Join(c.storagePath, cert.Domain+".key")
	if err := os.WriteFile(keyPath, cert.PrivateKey, 0600); err != nil {
		return fmt.Errorf("failed to save private key file: %w", err)
	}

	// Save issuer certificate if available
	if cert.IssuerCert != nil {
		issuerPath := filepath.Join(c.storagePath, cert.Domain+".issuer.crt")
		if err := os.WriteFile(issuerPath, cert.IssuerCert, 0644); err != nil {
			c.logger.Printf("Warning: failed to save issuer certificate: %v", err)
		}
	}

	return nil
}

func (c *ACMEClient) LoadCertificate(domain string) (*Certificate, error) {
	certPath := filepath.Join(c.storagePath, domain+".crt")
	keyPath := filepath.Join(c.storagePath, domain+".key")

	// Check if files exist
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("certificate file not found: %s", certPath)
	}
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("private key file not found: %s", keyPath)
	}

	// Read certificate
	certData, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate file: %w", err)
	}

	// Read private key
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file: %w", err)
	}

	// Load issuer certificate if available
	var issuerData []byte
	issuerPath := filepath.Join(c.storagePath, domain+".issuer.crt")
	if _, err := os.Stat(issuerPath); err == nil {
		issuerData, _ = os.ReadFile(issuerPath)
	}

	info, err := os.Stat(certPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get certificate file info: %w", err)
	}

	cert := &Certificate{
		Domain:      domain,
		Certificate: certData,
		PrivateKey:  keyData,
		IssuerCert:  issuerData,
		IssuedAt:    info.ModTime(),
	}

	// Parse certificate to get expiry
	if err := cert.parseCertificate(); err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	return cert, nil
}

// generatePrivateKey generates a private key based on the key type
func generatePrivateKey(keyType string) (crypto.PrivateKey, error) {
	switch keyType {
	case "RSA2048":
		return rsa.GenerateKey(rand.Reader, 2048)
	case "RSA4096":
		return rsa.GenerateKey(rand.Reader, 4096)
	default:
		return rsa.GenerateKey(rand.Reader, 2048)
	}
}

// getKeyType converts string key type to certcrypto.KeyType
func getKeyType(keyType string) certcrypto.KeyType {
	switch keyType {
	case "RSA2048":
		return certcrypto.RSA2048
	case "RSA4096":
		return certcrypto.RSA4096
	case "EC256":
		return certcrypto.EC256
	case "EC384":
		return certcrypto.EC384
	default:
		return certcrypto.RSA2048
	}
}

// Certificate represents an SSL/TLS certificate
type Certificate struct {
	Domain      string
	Certificate []byte
	PrivateKey  []byte
	IssuerCert  []byte
	URL         string
	IssuedAt    time.Time
	ExpiresAt   time.Time
}

// parseCertificate parses the certificate to extract expiry date
func (c *Certificate) parseCertificate() error {
	block, _ := pem.Decode(c.Certificate)
	if block == nil {
		return fmt.Errorf("failed to parse certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}

	c.ExpiresAt = cert.NotAfter
	return nil
}

func (c *Certificate) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}

func (c *Certificate) NeedsRenewal(renewalDays int) bool {
	renewalTime := c.ExpiresAt.AddDate(0, 0, -renewalDays)
	return time.Now().After(renewalTime)
}

func (c *Certificate) DaysUntilExpiry() int {
	duration := time.Until(c.ExpiresAt)
	return int(duration.Hours() / 24)
}

func (c *Certificate) GetCertPath(storagePath string) string {
	return filepath.Join(storagePath, c.Domain+".crt")
}

// GetKeyPath returns the path to the private key file
func (c *Certificate) GetKeyPath(storagePath string) string {
	return filepath.Join(storagePath, c.Domain+".key")
}