package main

import (
	"fmt"
	"log"
	"time"
	
	"github.com/O-tero/pkg/certs"
	"github.com/O-tero/pkg/services"

	"github.com/spf13/cobra"
	
)


func cliMain() {
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
		Run: func(cmd *cobra.Command, args []string) {
			domain := args[0]
			err := certs.RequestCertificate(domain)
			if err != nil {
				log.Fatalf("Failed to request certificate for domain %s: %v", domain, err)
			}
			fmt.Printf("Certificate successfully requested for domain: %s\n", domain)
		},
	}

	// Command to check the status of all certificates
	var checkCmd = &cobra.Command{
		Use:   "check-certificates",
		Short: "Check the status of all certificates",
		Run: func(cmd *cobra.Command, args []string) {
			status, err := certs.CheckCertificatesStatus()
			if err != nil {
				log.Fatalf("Failed to check certificate statuses: %v", err)
			}
			fmt.Println("Certificate Status:")
			for _, s := range status {
				expiryTime, err := time.Parse("2006-01-02", s.Expiry)
				if err != nil {
					log.Fatalf("Failed to parse expiry date for domain %s: %v", s.Domain, err)
				}
				fmt.Printf("Domain: %s | Expiry: %s | Status: %s\n", s.Domain, expiryTime.Format("2006-01-02"), s.Status)
			}
		},
	}

	// Command to send notifications for expiring certificates
	var notifyCmd = &cobra.Command{
		Use:   "send-notifications",
		Short: "Send notifications for expiring certificates",
		Run: func(cmd *cobra.Command, args []string) {
			err := services.SendExpirationNotifications()
			if err != nil {
				log.Fatalf("Failed to send notifications: %v", err)
			}
			fmt.Println("Notifications sent for expiring certificates.")
		},
	}
	

	// Add commands to root
	rootCmd.AddCommand(requestCmd, checkCmd, notifyCmd)

	// Execute the CLI
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("CLI error: %v", err)
	}
}
