package certs

import (
	"fmt"
	"io/ioutil"
	"github.com/O-tero/pkg/api"
	"github.com/go-acme/lego/v4/certificate"
	"path/filepath"
)

// Define the storage path for certificates and keys
const CertsStoragePath = "/path/to/certificates"

// GetCertificate retrieves the certificate and private key for a given domain from secure storage.
func GetCertificate(domain string) (string, string, error) {
	// Define file paths for certificate and key
	certPath := filepath.Join(CertsStoragePath, fmt.Sprintf("%s.crt", domain))
	keyPath := filepath.Join(CertsStoragePath, fmt.Sprintf("%s.key", domain))

	// Read the certificate file
	certData, err := ioutil.ReadFile(certPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to read certificate for domain %s: %v", domain, err)
	}

	// Read the private key file
	keyData, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to read private key for domain %s: %v", domain, err)
	}

	return string(certData), string(keyData), nil
}

func RenewAndApplyCertificates(domains []string) {
	for _, domain := range domains {
		if IsCertificateExpiring(CertificateStatus{Domain: domain}) {
			// Request a new certificate
			err := RequestCertificate(domain)
			if err != nil {
				fmt.Printf("Failed to renew certificate for %s: %v\n", domain, err)
				continue
			}

			// Fetch the new certificate and key from secure storage
			cert, key, err := GetCertificate(domain)
			if err != nil {
				fmt.Printf("Failed to retrieve certificate for %s: %v\n", domain, err)
				continue
			}

			// Create a certificate.Resource
			certResource := &certificate.Resource{
				Domain:       domain,
				Certificate:  []byte(cert),
				PrivateKey:   []byte(key),
				IssuerCertificate: nil, // Set issuer certificate if available
			}

			// Store and use the certificate
			err = StoreCertificate(certResource, domain)
			if err != nil {
				fmt.Printf("Failed to store certificate for %s: %v\n", domain, err)
				continue
			}

			// Push the certificate to Traefik dynamically
			err = api.PushCertificateToTraefik(api.Certificate{
				Domains: struct {
					Main string   `json:"main"`
					SANs []string `json:"sans,omitempty"`
				}{
					Main: domain,
				},
				Certificate: cert,
				Key:         key,
			})
			if err != nil {
				fmt.Printf("Failed to push certificate for %s to Traefik: %v\n", domain, err)
			}
		}
	}
}