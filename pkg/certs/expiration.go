package certs

import (
	"crypto/x509"
	"fmt"
	"github.com/O-tero/pkg/config"
	"time"
	"github.com/O-tero/pkg/notify"
)

type StoredCertificate struct {
	Domain string
	Cert   []byte
}

const expirationThreshold = 7 * 24 * time.Hour // Notify 7 days before expiry

func GetCertificateExpiry(cert []byte) (time.Time, error) {
    parsedCert, err := x509.ParseCertificate(cert)
    if err != nil {
        return time.Time{}, err
    }
    return parsedCert.NotAfter, nil
}



func CheckAndNotifyExpiringCertificates() {
	certificates, err := ListCertificates() // Fetch list of certificates from storage or config
	if err != nil {
		fmt.Printf("Failed to list certificates: %v\n", err)
		return
	}

	emailConfig := config.LoadEmailConfig() // Load email settings from configuration

	for _, cert := range certificates {
		expiryDate, err := GetCertificateExpiry(cert.Cert) // Assume this retrieves the certificate's expiration date
		if err != nil {
			fmt.Printf("Failed to parse certificate for %s: %v\n", cert.Domain, err)
			continue
		}
		timeLeft := time.Until(expiryDate)

		if timeLeft > 0 && timeLeft <= expirationThreshold {
			subject := fmt.Sprintf("Certificate Expiry Alert: %s", cert.Domain)
			body := fmt.Sprintf("The certificate for domain %s will expire on %s. Please renew it soon.",
				cert.Domain, expiryDate.Format("2006-01-02"))

			err := notify.SendEmail(emailConfig, subject, body)
			if err != nil {
				fmt.Printf("Failed to send notification for %s: %v\n", cert.Domain, err)
			}
		}
	}
}
