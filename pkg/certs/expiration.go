package certs

import (
	"fmt"
	"time"
	"pkg/config"
	"pkg/notify"
)

const expirationThreshold = 7 * 24 * time.Hour // Notify 7 days before expiry

func GetCertificateExpiry(cert []byte) (time.Time, error) {
    parsedCert, err := x509.ParseCertificate(cert)
    if err != nil {
        return time.Time{}, err
    }
    return parsedCert.NotAfter, nil
}


func CheckAndNotifyExpiringCertificates() {
	certificates := ListCertificates() // Fetch list of certificates from storage or config

	emailConfig := config.LoadEmailConfig() // Load email settings from configuration

	for _, cert := range certificates {
		expiryDate := GetCertificateExpiry(cert) // Assume this retrieves the certificate's expiration date
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
