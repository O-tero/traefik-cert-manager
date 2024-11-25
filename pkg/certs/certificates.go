package certs

import (
	"fmt"
	"log"
	"time"
	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
)

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
    log.Println("Checking for expiring certificates")
    // Logic to check and renew certificates
}

func StoreCertificate(certData []byte, domain string) error {
    log.Printf("Storing certificate for domain: %s", domain)
    // Save encrypted certificate data to disk
    return nil
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
