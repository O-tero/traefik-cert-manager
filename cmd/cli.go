package main

import (
	"fmt"
	"github.com/O-tero/pkg/certs"
	"github.com/O-tero/pkg/services"
	"github.com/spf13/cobra"
)

func cliMain() error {
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
