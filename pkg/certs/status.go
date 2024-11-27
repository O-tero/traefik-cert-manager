package certs

import (
	"crypto/x509"
	"fmt"
	"time"
)

type CertificateStatus struct {
	Domain string
	Expiry time.Time
	Status string // e.g., "Valid", "Expiring Soon", "Expired"
}

// CheckCertificatesStatus fetches the status of all stored certificates.
func CheckCertificatesStatus() ([]CertificateStatus, error) {
	certificates, err := ListCertificates() // Fetch certificates from storage
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
			Expiry: expiryDate,
			Status: status,
		})
	}

	return statuses, nil
}
