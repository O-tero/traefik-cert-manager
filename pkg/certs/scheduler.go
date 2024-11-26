package certs

import (
	"fmt"
	"time"
	"certs"
)

func CheckAndRenewCertificates() {
	certificates, err := LoadCertificates()
	if err != nil {
		fmt.Printf("Failed to load certificates: %v\n", err)
		return
	}

	for domain, cert := range certificates {
		if IsCertificateExpiring(cert) {
			fmt.Printf("Renewing certificate for domain: %s\n", domain)
			err := certificates.RequestCertificate(domain)
			if err != nil {
				fmt.Printf("Failed to renew certificate for %s: %v\n", domain, err)
			} else {
				fmt.Printf("Certificate renewed successfully for domain: %s\n", domain)
			}
		}
	}
}

func StartScheduler(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		<-ticker.C
		CheckAndRenewCertificates()
	}
}
