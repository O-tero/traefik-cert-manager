package main

import (
	"log"
	"pkg/config"
	"pkg/certs"
)

func main() {
	log.Println("Starting Certificate Manager...")

	// Load domains from configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Periodically check and renew certificates
	certs.RenewAndApplyCertificates(cfg.Domains)

	log.Println("Certificate Manager completed.")
}
