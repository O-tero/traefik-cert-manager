package certs

import (
	"crypto"
	"fmt"
	"time"

	"github.com/go-acme/lego/v4/certificate" 
	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
)

// User represents an ACME user.
type User struct {
	Email        string
	Registration *registration.Resource
	Key          crypto.PrivateKey
}

func (u *User) GetEmail() string {
	return u.Email
}

func (u *User) GetRegistration() *registration.Resource {
	return u.Registration
}

func (u *User) GetPrivateKey() crypto.PrivateKey {
	return u.Key
}

type Config struct {
	CADirURL  string
	KeyType   certcrypto.KeyType
	Email     string
	TLSConfig interface{}
}

// CertificateStatus represents the status of a certificate.
type CertificateStatus struct {
	Domain string
	Expiry string
	Status string
}

func LoadCertificates() (map[string]CertificateStatus, error) {
	return map[string]CertificateStatus{
		"example.com": {Domain: "example.com", Expiry: "2024-12-31", Status: "Valid"},
		"expired.com": {Domain: "expired.com", Expiry: "2023-01-01", Status: "Expired"},
	}, nil
}

// RequestCertificate requests a new certificate for a domain.
func RequestCertificate(domain string) error {
	// Initialize a Lego ACME client with required config
	config := lego.NewConfig(&User{})
	config.CADirURL = lego.LEDirectoryProduction
	config.Certificate.KeyType = certcrypto.RSA2048

	client, err := lego.NewClient(config)
	if err != nil {
		return fmt.Errorf("failed to create ACME client: %v", err)
	}

	// Register with the ACME server
	_, err = client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
	if err != nil {
		return fmt.Errorf("registration failed: %v", err)
	}

	// Request a certificate for the given domain
	request := certificate.ObtainRequest{ 
		Domains: []string{domain},
		Bundle:  true,
	}
	certificates, err := client.Certificate.Obtain(request)
	if err != nil {
		return fmt.Errorf("certificate request failed for domain %s: %v", domain, err)
	}

	// Store certificates securely
	return StoreCertificate(certificates, domain)
}


// IsCertificateExpiring checks if a certificate is nearing expiration.
func IsCertificateExpiring(cert CertificateStatus) bool {
	expiryDate, err := time.Parse("2006-01-02", cert.Expiry)
	if err != nil {
		return false
	}
	return time.Now().After(expiryDate.AddDate(0, 0, -30)) 
}

// CheckAndRenewCertificates checks for expiring certificates and renews them.
func CheckAndRenewCertificates() {
	certificates, err := LoadCertificates()
	if err != nil {
		fmt.Printf("Failed to load certificates: %v\n", err)
		return
	}

	for domain, cert := range certificates {
		if IsCertificateExpiring(cert) {
			fmt.Printf("Renewing certificate for domain: %s\n", domain)
			err := RequestCertificate(domain)
			if err != nil {
				fmt.Printf("Failed to renew certificate for %s: %v\n", domain, err)
			} else {
				fmt.Printf("Certificate renewed successfully for domain: %s\n", domain)
			}
		}
	}
}

// StartCertificateManager starts the certificate manager with periodic renewal checks.
func StartCertificateManager(cfg Config) {
	// Scheduler for periodic renewal checks
	ticker := time.NewTicker(24 * time.Hour)
	go func() {
		for range ticker.C {
			CheckAndRenewCertificates()
		}
	}()
}

// StartScheduler starts a scheduler with a custom interval.
func StartScheduler(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		<-ticker.C
		CheckAndRenewCertificates()
	}
}