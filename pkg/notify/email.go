package notify

import (
	"fmt"
	"net/smtp"

	"github.com/O-tero/pkg/config"
)

// SendEmail sends an email notification using the provided email configuration.
func SendEmail(emailConfig config.EmailConfig, subject, body string) error {
	// Set up authentication
	auth := smtp.PlainAuth("", emailConfig.SenderEmail, emailConfig.Password, emailConfig.SMTPServer)

	// Build the email
	msg := []byte(fmt.Sprintf("Subject: %s\r\n\r\n%s", subject, body))

	// Send the email
	addr := fmt.Sprintf("%s:%s", emailConfig.SMTPServer, emailConfig.SMTPPort)
	err := smtp.SendMail(addr, auth, emailConfig.SenderEmail, emailConfig.RecipientEmails, msg)
	if err != nil {
		return fmt.Errorf("failed to send email: %v", err)
	}

	fmt.Println("Email sent successfully.")
	return nil
}
