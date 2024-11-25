package main

import (
	"log"
	"net/http"
	"pkg/api"
	"pkg/certs"
	"time"
)

func main() {
	log.Println("Starting Certificate Manager...")

	// Schedule periodic certificate checks and renewals
	go func() {
		for {
			log.Println("Running periodic certificate renewal check...")
			certs.CheckAndRenewCertificates() // Periodic renewal logic
			time.Sleep(24 * time.Hour)       // Run daily
		}
	}()

	// Schedule periodic custom domain checks and certificate requests
	go func() {
		for {
			log.Println("Checking for certificates for custom domains...")
			certs.RequestCertificatesForCustomDomains()
			time.Sleep(1 * time.Hour) // Check hourly
		}
	}()

	// Set up API routes
	http.HandleFunc("/update-domains", api.UpdateDomainConfigsHandler) // Endpoint to update custom domains
	http.HandleFunc("/notify-expirations", api.NotifyExpirationsHandler) // (Optional) Notify manually via an endpoint

	// Start HTTP server for API endpoints
	go func() {
		serverAddress := ":8080"
		log.Printf("Starting API server on %s...\n", serverAddress)
		if err := http.ListenAndServe(serverAddress, nil); err != nil {
			log.Fatalf("Failed to start API server: %v", err)
		}
	}()

	// Prevent the application from exiting
	select {}
}
