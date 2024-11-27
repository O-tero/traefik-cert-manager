package config

import (
	"fmt"
	"os"
	"strings"
)

type EmailConfig struct {
	SMTPServer      string
	SMTPPort        string
	SenderEmail     string
	Password        string
	RecipientEmails []string
}

// LoadEmailConfig loads the email configuration from environment variables.
func LoadEmailConfig() (EmailConfig, error) {
	config := EmailConfig{
		SMTPServer:      os.Getenv("SMTP_SERVER"),
		SMTPPort:        os.Getenv("SMTP_PORT"),
		SenderEmail:     os.Getenv("SMTP_SENDER"),
		Password:        os.Getenv("SMTP_PASSWORD"),
		RecipientEmails: strings.Split(os.Getenv("SMTP_RECIPIENTS"), ","),
	}

	if config.SMTPServer == "" || config.SenderEmail == "" || config.Password == "" {
		return EmailConfig{}, fmt.Errorf("incomplete email configuration")
	}

	return config, nil
}
