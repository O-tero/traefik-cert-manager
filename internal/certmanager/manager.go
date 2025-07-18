package certmanager

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/O-tero/traefik-cert-manager/internal/config"
)

// ACMEClientInterface defines the interface for ACME client methods used by CertificateManager
type ACMEClientInterface interface {
	RequestCertificate(domain string) (*Certificate, error)
	RenewCertificate(cert *Certificate) (*Certificate, error)
	LoadCertificate(domain string) (*Certificate, error)
}

type CertificateManager struct {
	config     *config.Config
	acmeClient ACMEClientInterface
	logger     *log.Logger
	mu         sync.RWMutex
	certs      map[string]*Certificate
}

func NewCertificateManager(cfg *config.Config, logger *log.Logger) (*CertificateManager, error) {
	if logger == nil {
		logger = log.New(os.Stdout, "[CertManager] ", log.LstdFlags)
	}

	acmeConfig := ACMEConfig{
		CADirURL:    cfg.ACME.CADirURL,
		Email:       cfg.ACME.Email,
		KeyType:     cfg.ACME.KeyType,
		StoragePath: cfg.Certificates.StoragePath,
		Logger:      logger,
	}

	acmeClient, err := NewACMEClient(acmeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create ACME client: %w", err)
	}

	cm := &CertificateManager{
		config:     cfg,
		acmeClient: acmeClient,
		logger:     logger,
		certs:      make(map[string]*Certificate),
	}

	if err := cm.loadExistingCertificates(); err != nil {
		logger.Printf("Warning: failed to load existing certificates: %v", err)
	}

	return cm, nil
}

func (cm *CertificateManager) RequestCertificate(domain string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.logger.Printf("Requesting certificate for domain: %s", domain)

	if cert, exists := cm.certs[domain]; exists {
		if !cert.IsExpired() && !cert.NeedsRenewal(cm.config.Certificates.RenewalDays) {
			cm.logger.Printf("Certificate for %s is still valid, skipping request", domain)
			return nil
		}
		cm.logger.Printf("Certificate for %s needs renewal", domain)
	}

	cert, err := cm.acmeClient.RequestCertificate(domain)
	if err != nil {
		cm.logger.Printf("Failed to request certificate for %s: %v", domain, err)
		return fmt.Errorf("failed to request certificate for %s: %w", domain, err)
	}

	cm.certs[domain] = cert

	cm.logger.Printf("Successfully requested certificate for %s (expires: %s)", 
		domain, cert.ExpiresAt.Format(time.RFC3339))

	return nil
}

func (cm *CertificateManager) RenewCertificate(domain string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.logger.Printf("Renewing certificate for domain: %s", domain)

	cert, exists := cm.certs[domain]
	if !exists {
		loadedCert, err := cm.acmeClient.LoadCertificate(domain)
		if err != nil {
			return fmt.Errorf("certificate not found for domain %s: %w", domain, err)
		}
		cert = loadedCert
		cm.certs[domain] = cert
	}

	renewedCert, err := cm.acmeClient.RenewCertificate(cert)
	if err != nil {
		cm.logger.Printf("Failed to renew certificate for %s: %v", domain, err)
		return fmt.Errorf("failed to renew certificate for %s: %w", domain, err)
	}

	cm.certs[domain] = renewedCert

	cm.logger.Printf("Successfully renewed certificate for %s (expires: %s)", 
		domain, renewedCert.ExpiresAt.Format(time.RFC3339))

	return nil
}

func (cm *CertificateManager) GetCertificate(domain string) (*Certificate, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	cert, exists := cm.certs[domain]
	if !exists {
		return nil, fmt.Errorf("certificate not found for domain: %s", domain)
	}

	return cert, nil
}

func (cm *CertificateManager) ListCertificates() map[string]*Certificate {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// Return a copy to prevent external modification
	result := make(map[string]*Certificate)
	for domain, cert := range cm.certs {
		result[domain] = cert
	}

	return result
}

func (cm *CertificateManager) CheckCertificateHealth() map[string]CertificateHealth {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	health := make(map[string]CertificateHealth)

	for domain, cert := range cm.certs {
		status := CertificateHealth{
			Domain:    domain,
			IssuedAt:  cert.IssuedAt,
			ExpiresAt: cert.ExpiresAt,
			IsExpired: cert.IsExpired(),
			DaysUntilExpiry: cert.DaysUntilExpiry(),
		}

		status.NeedsRenewal = cert.NeedsRenewal(cm.config.Certificates.RenewalDays)

		if status.IsExpired {
			status.Status = "expired"
		} else if status.NeedsRenewal {
			status.Status = "needs_renewal"
		} else {
			status.Status = "valid"
		}

		health[domain] = status
	}

	return health
}

func (cm *CertificateManager) ProcessAllDomains(ctx context.Context) error {
	domains := cm.config.GetAllDomains()
	
	cm.logger.Printf("Processing %d domains", len(domains))

	var errs []error
	for _, domain := range domains {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := cm.RequestCertificate(domain); err != nil {
				errs = append(errs, fmt.Errorf("failed to process domain %s: %w", domain, err))
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to process %d domains: %v", len(errs), errs)
	}

	return nil
}

func (cm *CertificateManager) RenewExpiredCertificates(ctx context.Context) error {
	health := cm.CheckCertificateHealth()
	
	var errs []error
	for domain, status := range health {
		if status.NeedsRenewal {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				cm.logger.Printf("Certificate for %s needs renewal (expires in %d days)", 
					domain, status.DaysUntilExpiry)
				
				if err := cm.RenewCertificate(domain); err != nil {
					errs = append(errs, fmt.Errorf("failed to renew certificate for %s: %w", domain, err))
				}
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to renew %d certificates: %v", len(errs), errs)
	}

	return nil
}

func (cm *CertificateManager) loadExistingCertificates() error {
	storagePath := cm.config.Certificates.StoragePath

	// Check if storage directory exists
	if _, err := os.Stat(storagePath); os.IsNotExist(err) {
		cm.logger.Printf("Storage directory %s does not exist, creating it", storagePath)
		if err := os.MkdirAll(storagePath, 0755); err != nil {
			return fmt.Errorf("failed to create storage directory: %w", err)
		}
		return nil
	}

	entries, err := os.ReadDir(storagePath)
	if err != nil {
		return fmt.Errorf("failed to read storage directory: %w", err)
	}

	// Find certificate files
	certFiles := make(map[string]bool)
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".crt" {
			domain := entry.Name()[:len(entry.Name())-4] // Remove .crt extension
			if domain != "" && domain != "issuer" {
				certFiles[domain] = true
			}
		}
	}

	// Load certificates
	for domain := range certFiles {
		cert, err := cm.acmeClient.LoadCertificate(domain)
		if err != nil {
			cm.logger.Printf("Failed to load certificate for %s: %v", domain, err)
			continue
		}

		cm.certs[domain] = cert
		cm.logger.Printf("Loaded certificate for %s (expires: %s)", 
			domain, cert.ExpiresAt.Format(time.RFC3339))
	}

	cm.logger.Printf("Loaded %d certificates from disk", len(cm.certs))
	return nil
}

type CertificateHealth struct {
	Domain          string    `json:"domain"`
	Status          string    `json:"status"` // valid, needs_renewal, expired
	IssuedAt        time.Time `json:"issued_at"`
	ExpiresAt       time.Time `json:"expires_at"`
	IsExpired       bool      `json:"is_expired"`
	NeedsRenewal    bool      `json:"needs_renewal"`
	DaysUntilExpiry int       `json:"days_until_expiry"`
}

func (cm *CertificateManager) GetCertificatePaths(domain string) (certPath, keyPath string) {
	certPath = filepath.Join(cm.config.Certificates.StoragePath, domain+".crt")
	keyPath = filepath.Join(cm.config.Certificates.StoragePath, domain+".key")
	return certPath, keyPath
}

func (cm *CertificateManager) Cleanup() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	var cleaned []string
	for domain, cert := range cm.certs {
		// Remove certificates that have been expired for more than 30 days
		if cert.IsExpired() && time.Since(cert.ExpiresAt) > 30*24*time.Hour {
			delete(cm.certs, domain)
			cleaned = append(cleaned, domain)
		}
	}

	if len(cleaned) > 0 {
		cm.logger.Printf("Cleaned up %d expired certificates: %v", len(cleaned), cleaned)
	}

	return nil
}