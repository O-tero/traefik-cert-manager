package certs

import (
	"fmt"
	"time"
	"crypto"
	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
	"github.com/O-tero/certs"
)

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

type CertificateStatus struct {
	Domain string
	Expiry string
	Status string
}

// CheckCertificatesStatus checks the status of certificates
func CheckCertificatesStatus() []CertificateStatus {
	// Mock implementation
	return []CertificateStatus{
		{Domain: "example.com", Expiry: "2024-12-31", Status: "Valid"},
		{Domain: "expired.com", Expiry: "2023-01-01", Status: "Expired"},
	}
}


func RequestCertificate(domain string) error {
	// Initialize a Lego ACME client with required config
	config := lego.NewConfig(&User{}) // User implements lego.User interface
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
	request := lego.CertificateRequest{
		Domains:    []string{domain},
		Bundle:     true,
	}
	certificates, err := client.Certificate.Obtain(request)
	if err != nil {
		return fmt.Errorf("certificate request failed for domain %s: %v", domain, err)
	}

	// Store certificates securely
	return StoreCertificate(certificates, domain)
}


func CheckAndRenewCertificates() {
	certificates, err := LoadCertificates()
	if err != nil {
		fmt.Printf("Failed to load certificates: %v\n", err)
		return
	}

	for domain, cert := range certificates {
		if IsCertificateExpiring(cert) {
			fmt.Printf("Renewing certificate for domain: %s\n", domain)
			err := certificates.RequestCertificate(domain)
			if err != nil {
				fmt.Printf("Failed to renew certificate for %s: %v\n", domain, err)
			} else {
				fmt.Printf("Certificate renewed successfully for domain: %s\n", domain)
			}
		}
	}
}

func StartCertificateManager(cfg Config) {
    // Scheduler for periodic renewal checks
    ticker := time.NewTicker(24 * time.Hour)
    go func() {
        for range ticker.C {
            CheckAndRenewCertificates()
        }
    }()
}

