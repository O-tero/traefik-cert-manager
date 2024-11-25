// pkg/notify/notify.go
package notify

import (
    "log"
    "net/smtp"
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
