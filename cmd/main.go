package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/O-tero/pkg/api"
	"github.com/O-tero/pkg/certs"
	"github.com/O-tero/web"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "web" {
		startWebInterface()
	} else if len(os.Args) > 1 && os.Args[1] == "cli" {
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
func startCLI() {
	log.Println("Starting CLI...")
	os.Args = os.Args[1:] 
	mainCLI()             
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

// Placeholder function for CLI logic
func mainCLI() {
	log.Println("CLI logic not implemented in this main.go file.")
}
