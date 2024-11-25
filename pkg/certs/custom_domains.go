package certs

import (
	"fmt"
	"pkg/config"
)

func RequestCertificatesForCustomDomains() {
	domainConfigs, err := config.LoadDomainConfigs()
	if err != nil {
		fmt.Printf("Error loading domain configurations: %v\n", err)
		return
	}

	for _, domainConfig := range domainConfigs {
		err := RequestCertificate(domainConfig.Domain)
		if err != nil {
			fmt.Printf("Failed to request certificate for %s: %v\n", domainConfig.Domain, err)
		} else {
			fmt.Printf("Successfully requested certificate for %s\n", domainConfig.Domain)
		}
	}
}
