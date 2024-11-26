package notify

import (
	"fmt"
	"net/smtp"
	"os"
	"strings"
)

type EmailConfig struct {
	SMTPServer   string
	SMTPPort     string
	SenderEmail  string
	Password     string
	RecipientEmails []string
}

func LoadEmailConfig() EmailConfig {
	return EmailConfig{
		SMTPServer:      os.Getenv("SMTP_SERVER"),
		SMTPPort:        os.Getenv("SMTP_PORT"),
		SenderEmail:     os.Getenv("SMTP_SENDER"),
		Password:        os.Getenv("SMTP_PASSWORD"),
		RecipientEmails: strings.Split(os.Getenv("SMTP_RECIPIENTS"), ","),
	}
}

func SendEmail(config EmailConfig, subject, body string) error {
	// Set up authentication
	auth := smtp.PlainAuth("", config.SenderEmail, config.Password, config.SMTPServer)

	// Build the email
	to := strings.Join(config.RecipientEmails, ",")
	msg := []byte(fmt.Sprintf("Subject: %s\r\n\r\n%s", subject, body))

	// Send the email
	addr := fmt.Sprintf("%s:%s", config.SMTPServer, config.SMTPPort)
	err := smtp.SendMail(addr, auth, config.SenderEmail, config.RecipientEmails, msg)
	if err != nil {
		return fmt.Errorf("failed to send email: %v", err)
	}

	fmt.Println("Email sent successfully.")
	return nil
}
