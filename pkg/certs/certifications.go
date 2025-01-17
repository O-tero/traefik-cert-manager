package certs

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"time"

	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/challenge/http01"
	"github.com/go-acme/lego/v4/challenge/tlsalpn01"
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

func generateUserPrivateKey() (crypto.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, 2048)
}

func NewUser(email string) (*User, error) {
	user := &User{
		Email: email,
	}
	key, err := generateUserPrivateKey()
	if err != nil {
		return nil, fmt.Errorf("error generating private key: %v", err)
	}
	user.Key = key
	return user, nil
}

func LoadCertificates() (map[string]CertificateStatus, error) {
	return map[string]CertificateStatus{
		"my-test-domain1.duckdns.org": {Domain: "my-test-domain1.duckdns.org", Expiry: "2024-12-31", Status: "Valid"},
		"my-test-domain2.duckdns.org": {Domain: "my-test-domain2.duckdns.org", Expiry: "2023-01-01", Status: "Expired"},
	}, nil
}

// RequestCertificate requests a new certificate for a domain.
func RequestCertificate(domain string) error {
	user := &User{
		Email: "zzv70525@msssg.com",
	}

	// Generate private key if not already set
	if user.Key == nil {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return fmt.Errorf("failed to generate private key: %v", err)
		}
		user.Key = key
	}

	config := lego.NewConfig(user)
	config.CADirURL = lego.LEDirectoryStaging 
	config.Certificate.KeyType = certcrypto.RSA2048

	client, err := lego.NewClient(config)
	if err != nil {
		return fmt.Errorf("failed to create ACME client: %v", err)
	}

	// Configure HTTP-01 solver
	httpProvider := http01.NewProviderServer("", "80")
	err = client.Challenge.SetHTTP01Provider(httpProvider)
	if err != nil {
		return fmt.Errorf("failed to set HTTP-01 provider: %v", err)
	}

	// Configure TLS-ALPN-01 solver
	tlsProvider := tlsalpn01.NewProviderServer("", "443")
	err = client.Challenge.SetTLSALPN01Provider(tlsProvider)
	if err != nil {
		return fmt.Errorf("failed to set TLS-ALPN-01 provider: %v", err)
	}

	// Register user with ACME
	_, err = client.Registration.Register(registration.RegisterOptions{
		TermsOfServiceAgreed: true,
	})
	if err != nil {
		return fmt.Errorf("registration failed: %v", err)
	}

	request := certificate.ObtainRequest{
		Domains: []string{domain},
		Bundle:  true,
	}
	cert, err := client.Certificate.Obtain(request)
	if err != nil {
		return fmt.Errorf("certificate request failed for domain %s: %v", domain, err)
	}

	return StoreCertificate(cert, domain)
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

// CheckCertificatesStatus fetches the status of all stored certificates.
func CheckCertificatesStatus() ([]CertificateStatus, error) {
	certificates, err := ListCertificates()
	if err != nil {
		return nil, fmt.Errorf("failed to list certificates: %w", err)
	}

	var statuses []CertificateStatus

	for _, cert := range certificates {
		expiryDate, err := GetCertificateExpiry(cert.Cert)
		if err != nil {
			fmt.Printf("Failed to parse certificate for domain %s: %v\n", cert.Domain, err)
			continue
		}

		timeLeft := time.Until(expiryDate)
		status := "Valid"

		if timeLeft <= 0 {
			status = "Expired"
		} else if timeLeft <= expirationThreshold {
			status = "Expiring Soon"
		}

		statuses = append(statuses, CertificateStatus{
			Domain: cert.Domain,
			Expiry: expiryDate.Format("2006-01-02"),
			Status: status,
		})
	}

	return statuses, nil
}
