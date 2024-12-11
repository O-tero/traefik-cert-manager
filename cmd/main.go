package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/O-tero/pkg/api"
	"github.com/O-tero/pkg/certs"
	"github.com/O-tero/pkg/services"
	"github.com/O-tero/web"
	"github.com/spf13/cobra"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "web" {
		startWebInterface()
	} else if len(os.Args) > 1 {
		// Default to CLI for any other arguments
		if err := cliMain(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	} else {
		startDefaultMode()
	}
}

// Default mode: Starts periodic tasks and API server
func startDefaultMode() {
	log.Println("Starting Certificate Manager in default mode...")

	go scheduleCertificateRenewal()
	go scheduleCustomDomainCheck()
	startAPIServer()

	// Prevent the application from exiting
	select {}
}

// Web interface mode
func startWebInterface() {
	log.Println("Starting web interface...")
	web.StartServer()
}

// CLI mode
func cliMain() error {
	log.Println("Starting CLI...")

	// Root command
	var rootCmd = &cobra.Command{
		Use:   "cert-manager",
		Short: "Certificate Manager CLI",
		Long:  `A CLI tool for managing SSL/TLS certificates using Traefik.`,
	}

	// Command to request a certificate for a specific domain
	var requestCmd = &cobra.Command{
		Use:   "request-certificate [domain]",
		Short: "Request a new SSL/TLS certificate for a domain",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			domain := args[0]
			err := certs.RequestCertificate(domain)
			if err != nil {
				return fmt.Errorf("failed to request certificate for domain %s: %w", domain, err)
			}
			fmt.Printf("Certificate successfully requested for domain: %s\n", domain)
			return nil
		},
	}

	// Command to check the status of all certificates
	var checkCmd = &cobra.Command{
		Use:   "check-certificates",
		Short: "Check the status of all certificates",
		RunE: func(cmd *cobra.Command, args []string) error {
			status, err := certs.CheckCertificatesStatus()
			if err != nil {
				return fmt.Errorf("failed to check certificate statuses: %w", err)
			}
			fmt.Println("Certificate Status:")
			for _, s := range status {
				fmt.Printf("Domain: %s | Expiry: %s | Status: %s\n", s.Domain, s.Expiry, s.Status)
			}
			return nil
		},
	}

	// Command to send notifications for expiring certificates
	var notifyCmd = &cobra.Command{
		Use:   "send-notifications",
		Short: "Send notifications for expiring certificates",
		RunE: func(cmd *cobra.Command, args []string) error {
			err := services.SendExpirationNotifications()
			if err != nil {
				return fmt.Errorf("failed to send notifications: %w", err)
			}
			fmt.Println("Notifications sent for expiring certificates.")
			return nil
		},
	}

	// Add commands to root
	rootCmd.AddCommand(requestCmd, checkCmd, notifyCmd)

	// Execute the CLI
	return rootCmd.Execute()
}

// Schedules periodic certificate renewal checks
func scheduleCertificateRenewal() {
	for {
		log.Println("Running periodic certificate renewal check...")
		certs.CheckAndRenewCertificates()
		time.Sleep(24 * time.Hour)
	}
}

// Schedules periodic custom domain checks and certificate requests
func scheduleCustomDomainCheck() {
	for {
		log.Println("Checking for certificates for custom domains...")
		certs.RequestCertificatesForCustomDomains()
		time.Sleep(1 * time.Hour)
	}
}

// Starts the API server
func startAPIServer() {
	go func() {
		http.HandleFunc("/update-domains", api.UpdateDomainConfigsHandler)
		http.HandleFunc("/notify-expirations", api.NotifyExpirationsHandler)

		serverAddress := ":8080"
		log.Printf("Starting API server on %s...\n", serverAddress)
		if err := http.ListenAndServe(serverAddress, nil); err != nil {
			log.Fatalf("Failed to start API server: %v", err)
		}
	}()
}
