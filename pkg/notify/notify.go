// pkg/notify/notify.go
package notify

import (
    "fmt"
    "log"
    "net/smtp"
    "github.com/O-tero/pkg"
)

func SendEmailNotification(email, subject, body string) error {
    from := "youremail@example.com"
    password := "yourpassword"

    smtpHost := "smtp.example.com"
    smtpPort := "587"

    auth := smtp.PlainAuth("", from, password, smtpHost)

    msg := []byte("Subject: " + subject + "\n\n" + body)
    err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, []string{email}, msg)
    if err != nil {
        log.Printf("Failed to send email: %v", err)
        return err
    }

    log.Println("Email notification sent successfully")
    return nil
}

// SendExpirationNotifications sends notifications for certificates nearing expiration
func SendExpirationNotifications() error {
	certStatuses := certs.CheckCertificatesStatus()

	for _, cert := range certStatuses {
		if cert.Status == "Expiring Soon" { // Example condition
			// Replace with actual notification logic (email, SMS, etc.)
			log.Printf("Sending notification for domain %s expiring on %s\n", cert.Domain, cert.Expiry)
		}
	}

	fmt.Println("Notifications sent for all expiring certificates.")
	return nil
}
