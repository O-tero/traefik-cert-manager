package certs

import (
	"fmt"
	"github.com/O-tero/pkg/api"
)

func RenewAndApplyCertificates(domains []string) {
	for _, domain := range domains {
		if IsCertificateExpiring(domain) {
			// Request a new certificate
			err := RequestCertificate(domain)
			if err != nil {
				fmt.Printf("Failed to renew certificate for %s: %v\n", domain, err)
				continue
			}

			// Fetch the new certificate and key
			cert, key := GetCertificate(domain) // Assume this retrieves cert/key for the domain
			err = StoreCertificate(domain, cert, key)
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
