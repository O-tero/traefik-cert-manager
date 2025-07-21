package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/O-tero/traefik-cert-manager/internal/certmanager"
	"github.com/O-tero/traefik-cert-manager/internal/config"
	"github.com/O-tero/traefik-cert-manager/internal/traefik"
)

const (
	defaultConfigPath = "./configs/config.yaml"
	version           = "1.0.0"
)

func main() {
	var (
		configPath  = flag.String("config", defaultConfigPath, "Path to configuration file")
		showVersion = flag.Bool("version", false, "Show version information")
		runOnce     = flag.Bool("once", false, "Run certificate check once and exit")
		verbose     = flag.Bool("verbose", false, "Enable verbose logging")
		checkHealth = flag.Bool("health", false, "Check certificate health and exit")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("Traefik Certificate Manager v%s\n", version)
		return
	}

	// Setup logging
	logLevel := log.LstdFlags
	if *verbose {
		logLevel = log.LstdFlags | log.Lshortfile
	}
	logger := log.New(os.Stdout, "[CertManager] ", logLevel)

	logger.Printf("Starting Traefik Certificate Manager v%s", version)

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		logger.Fatalf("Failed to load configuration: %v", err)
	}

	logger.Printf("Configuration loaded from: %s", *configPath)
	logger.Printf("ACME CA: %s", cfg.ACME.CADirURL)
	logger.Printf("Storage path: %s", cfg.Certificates.StoragePath)
	logger.Printf("Renewal threshold: %d days", cfg.Certificates.RenewalDays)

	// Ensure storage directory exists
	if err := os.MkdirAll(cfg.Certificates.StoragePath, 0755); err != nil {
		logger.Fatalf("Failed to create storage directory: %v", err)
	}

	// Create certificate manager
	certManager, err := certmanager.NewCertificateManager(cfg, logger)
	if err != nil {
		logger.Fatalf("Failed to create certificate manager: %v", err)
	}

	// Create Traefik API client
	timeout, _ := cfg.GetTimeout()
	traefikClient := traefik.NewAPIClient(cfg.TraefikAPI, timeout)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	if err := traefikClient.IsHealthy(ctx); err != nil {
		logger.Fatalf("Failed to connect to Traefik API: %v", err)
	}
	cancel()
	logger.Printf("Connected to Traefik API: %s", cfg.TraefikAPI)

	if *checkHealth {
		runHealthCheck(certManager, logger)
		return
	}

	if *runOnce {
		runOnceMode(certManager, logger)
		return
	}

	// Create and start scheduler for continuous operation
	scheduler, err := certmanager.NewScheduler(cfg, certManager, logger)
	if err != nil {
		logger.Fatalf("Failed to create scheduler: %v", err)
	}

	logger.Printf("Processing initial certificates...")
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Minute)
	if err := certManager.ProcessAllDomains(ctx); err != nil {
		logger.Printf("Warning: Failed to process some domains: %v", err)
	}
	cancel()

	// Start the scheduler
	if err := scheduler.Start(); err != nil {
		logger.Fatalf("Failed to start scheduler: %v", err)
	}

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	logger.Printf("Certificate manager started successfully")
	logger.Printf("Next check scheduled for: %s", scheduler.GetNextRunTime().Format(time.RFC3339))

	// Wait for shutdown signal
	<-sigChan
	logger.Printf("Shutdown signal received, stopping...")

	// Graceful shutdown
	if err := scheduler.Stop(); err != nil {
		logger.Printf("Error stopping scheduler: %v", err)
	}

	logger.Printf("Certificate manager stopped")
}

// runHealthCheck performs a health check and displays certificate status
func runHealthCheck(certManager *certmanager.CertificateManager, logger *log.Logger) {
	logger.Printf("Running certificate health check...")

	health := certManager.CheckCertificateHealth()
	if len(health) == 0 {
		logger.Printf("No certificates found")
		return
	}

	logger.Printf("Certificate Health Report:")
	logger.Printf("========================")

	var validCount, renewalCount, expiredCount int

	for domain, status := range health {
		logger.Printf("Domain: %s", domain)
		logger.Printf("  Status: %s", status.Status)
		logger.Printf("  Issued: %s", status.IssuedAt.Format(time.RFC3339))
		logger.Printf("  Expires: %s", status.ExpiresAt.Format(time.RFC3339))
		logger.Printf("  Days until expiry: %d", status.DaysUntilExpiry)
		logger.Printf("  Needs renewal: %t", status.NeedsRenewal)
		logger.Printf("  Is expired: %t", status.IsExpired)
		logger.Printf("")

		switch status.Status {
		case "valid":
			validCount++
		case "needs_renewal":
			renewalCount++
		case "expired":
			expiredCount++
		}
	}

	logger.Printf("Summary:")
	logger.Printf("  Total certificates: %d", len(health))
	logger.Printf("  Valid: %d", validCount)
	logger.Printf("  Need renewal: %d", renewalCount)
	logger.Printf("  Expired: %d", expiredCount)

	if renewalCount > 0 || expiredCount > 0 {
		os.Exit(1)
	}
}

// runOnceMode runs the certificate manager once and exits
func runOnceMode(certManager *certmanager.CertificateManager, logger *log.Logger) {
	logger.Printf("Running in single-execution mode...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Process all configured domains
	if err := certManager.ProcessAllDomains(ctx); err != nil {
		logger.Printf("Error processing domains: %v", err)
	}

	// Check for and renew certificates that need it
	if err := certManager.RenewExpiredCertificates(ctx); err != nil {
		logger.Printf("Error renewing certificates: %v", err)
	}

	// Display final health status
	logger.Println("Final certificate health status after single run:")
	runHealthCheck(certManager, logger)

	logger.Println("Single-execution mode finished.")
}
