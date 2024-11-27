// pkg/services/expiration_notifications.go
package services

import (
	"fmt"
	"log"
	"time"

	"github.com/O-tero/pkg/certs"
	"github.com/O-tero/pkg/notify"
)

// SendExpirationNotifications sends notifications for certificates nearing expiration
func SendExpirationNotifications() error {
	certStatuses, err := certs.CheckCertificatesStatus()
	if err != nil {
		return fmt.Errorf("failed to check certificate statuses: %v", err)
	}

	for _, cert := range certStatuses {
		if cert.Status == "Expiring Soon" { // Example condition
			subject := fmt.Sprintf("Certificate Expiry Alert for %s", cert.Domain)
			body := fmt.Sprintf("The certificate for domain %s will expire on %s. Please renew it.",
				cert.Domain, parseExpiry(cert.Expiry).Format("2006-01-02"))

			// Replace with actual recipient email
			email := "admin@example.com"

			err := notify.SendEmailNotification(email, subject, body)
			if err != nil {
				log.Printf("Failed to send notification for domain %s: %v", cert.Domain, err)
			}
		}
	}

	log.Println("Notifications sent for all expiring certificates.")
	return nil
}

func parseExpiry(expiry string) time.Time {
	parsedTime, err := time.Parse("2006-01-02", expiry)
	if err != nil {
		log.Printf("Failed to parse expiry date %s: %v", expiry, err)
		return time.Time{}
	}
	return parsedTime
}
