package certs

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge/http01"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
)

// Global variables to track server instances
var (
	challengeServer      *http.Server
	proxyServer          *http.Server
	serverMutex          sync.Mutex
	defaultChallengePort int
)

// User represents an ACME user.
type User struct {
	Email        string
	Registration *registration.Resource
	Key          crypto.PrivateKey
}

func (u *User) GetEmail() string {
	return u.Email
}

func (u *User) GetRegistration() *registration.Resource {
	return u.Registration
}

func (u *User) GetPrivateKey() crypto.PrivateKey {
	return u.Key
}

type Config struct {
	CADirURL  string
	KeyType   certcrypto.KeyType
	Email     string
	TLSConfig interface{}
}

// CertificateStatus represents the status of a certificate.
type CertificateStatus struct {
	Domain string
	Expiry string
	Status string
}

// checkPortAvailable checks if a port is available
func checkPortAvailable(port string) bool {
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

// findAvailablePort finds the next available port starting from the given port
func findAvailablePort(startPort int) (int, error) {
	for port := startPort; port < startPort+100; port++ {
		if checkPortAvailable(fmt.Sprintf("%d", port)) {
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available ports found in range %d-%d", startPort, startPort+100)
}

func generateUserPrivateKey() (crypto.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, 2048)
}

func NewUser(email string) (*User, error) {
	user := &User{
		Email: email,
	}
	key, err := generateUserPrivateKey()
	if err != nil {
		return nil, fmt.Errorf("error generating private key: %v", err)
	}
	user.Key = key
	return user, nil
}

func LoadCertificates() (map[string]CertificateStatus, error) {
	return map[string]CertificateStatus{
		"my-test-domain1.duckdns.org": {Domain: "my-test-domain1.duckdns.org", Expiry: "2024-12-31", Status: "Valid"},
		"my-test-domain2.duckdns.org": {Domain: "my-test-domain2.duckdns.org", Expiry: "2023-01-01", Status: "Expired"},
	}, nil
}

// SetupHTTPProxy creates a proxy server to handle ACME HTTP-01 challenges
func SetupHTTPProxy() error {
	serverMutex.Lock()
	defer serverMutex.Unlock()

	// Clean up any existing servers
	if proxyServer != nil {
		proxyServer.Close()
	}
	if challengeServer != nil {
		challengeServer.Close()
	}

	// Find available ports
	challengePort, err := findAvailablePort(8090)
	if err != nil {
		return fmt.Errorf("failed to find available challenge port: %v", err)
	}

	// Setup challenge server first
	challengeServer = &http.Server{
		Addr: fmt.Sprintf(":%d", challengePort),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/.well-known/acme-challenge/") {
				// Handle ACME challenge
				http.DefaultServeMux.ServeHTTP(w, r)
			} else {
				http.Error(w, "Not found", http.StatusNotFound)
			}
		}),
	}

	// Start challenge server
	go func() {
		if err := challengeServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Challenge server error: %v\n", err)
		}
	}()

	// Try to set up proxy on port 80, but fall back to high port if needed
	proxyPort := 80
	if !checkPortAvailable("80") {
		var err error
		proxyPort, err = findAvailablePort(8080)
		if err != nil {
			return fmt.Errorf("failed to find available proxy port: %v", err)
		}
		log.Printf("Warning: Port 80 not available, using port %d instead. Certificate validation may fail.", proxyPort)
	}

	proxyServer = &http.Server{
		Addr: fmt.Sprintf(":%d", proxyPort),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/.well-known/acme-challenge/") {
				proxy := httputil.NewSingleHostReverseProxy(&url.URL{
					Scheme: "http",
					Host:   fmt.Sprintf("localhost:%d", challengePort),
				})
				proxy.ServeHTTP(w, r)
				return
			}
			http.Error(w, "Not found", http.StatusNotFound)
		}),
	}

	go func() {
		if err := proxyServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Proxy server error: %v\n", err)
		}
	}()

	// Update the defaultChallengePort
	defaultChallengePort = challengePort

	log.Printf("Challenge server running on port %d, proxy server on port %d", challengePort, proxyPort)
	return nil
}

// RequestCertificate requests a new certificate for a domain.
func RequestCertificate(domain string) error {
	user := &User{
		Email: "zzv70525@msssg.com",
	}

	if user.Key == nil {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return fmt.Errorf("failed to generate private key: %v", err)
		}
		user.Key = key
	}

	config := lego.NewConfig(user)
	config.CADirURL = lego.LEDirectoryStaging
	config.Certificate.KeyType = certcrypto.RSA2048

	client, err := lego.NewClient(config)
	if err != nil {
		return fmt.Errorf("failed to create ACME client: %v", err)
	}

	// Use the discovered challenge port
	httpProvider := http01.NewProviderServer("", fmt.Sprintf("%d", defaultChallengePort))
	err = client.Challenge.SetHTTP01Provider(httpProvider)
	if err != nil {
		return fmt.Errorf("failed to set HTTP-01 provider: %v", err)
	}

	// Register if not already registered
	_, err = client.Registration.Register(registration.RegisterOptions{
		TermsOfServiceAgreed: true,
	})
	if err != nil {
		return fmt.Errorf("registration failed: %v", err)
	}

	request := certificate.ObtainRequest{
		Domains: []string{domain},
		Bundle:  true,
	}

	cert, err := client.Certificate.Obtain(request)
	if err != nil {
		return fmt.Errorf("certificate request failed for domain %s: %v", domain, err)
	}

	return StoreCertificate(cert, domain)
}

// IsCertificateExpiring checks if a certificate is expiring within 30 days.
func IsCertificateExpiring(cert CertificateStatus) bool {
	expiryDate, err := time.Parse("2006-01-02", cert.Expiry)
	if err != nil {
		log.Printf("Failed to parse expiry date for domain %s: %v\n", cert.Domain, err)
		return false
	}
	return time.Until(expiryDate) < 30*24*time.Hour
}

func CheckAndRenewCertificates() error {
	certificates, err := LoadCertificates()
	if err != nil {
		return fmt.Errorf("failed to load certificates: %v", err)
	}

	for domain, cert := range certificates {
		if IsCertificateExpiring(cert) {
			log.Printf("Renewing certificate for domain: %s\n", domain)
			err := RequestCertificate(domain)
			if err != nil {
				log.Printf("Failed to renew certificate for %s: %v\n", domain, err)
			} else {
				log.Printf("Certificate renewed successfully for domain: %s\n", domain)
			}
		}
	}
	return nil
}

func StartCertificateManager(cfg Config) error {
	// Set up the HTTP proxy first
	if err := SetupHTTPProxy(); err != nil {
		return fmt.Errorf("failed to set up HTTP proxy: %v", err)
	}

	// Do an initial check immediately
	if err := CheckAndRenewCertificates(); err != nil {
		log.Printf("Initial certificate check failed: %v\n", err)
	}

	// Start periodic checks
	ticker := time.NewTicker(24 * time.Hour)
	go func() {
		for range ticker.C {
			if err := CheckAndRenewCertificates(); err != nil {
				log.Printf("Periodic certificate check failed: %v\n", err)
			}
		}
	}()

	return nil
}

// StartScheduler starts a scheduler with a custom interval.
func StartScheduler(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		<-ticker.C
		CheckAndRenewCertificates()
	}
}

func CheckCertificatesStatus() ([]CertificateStatus, error) {
	certificates, err := ListCertificates()
	if err != nil {
		return nil, fmt.Errorf("failed to list certificates: %w", err)
	}

	var statuses []CertificateStatus
	for _, cert := range certificates {
		expiryDate, err := GetCertificateExpiry(cert.Cert)
		if err != nil {
			fmt.Printf("Failed to parse certificate for domain %s: %v\n", cert.Domain, err)
			continue
		}

		timeLeft := time.Until(expiryDate)
		status := "Valid"

		if timeLeft <= 0 {
			status = "Expired"
		} else if timeLeft <= expirationThreshold {
			status = "Expiring Soon"
		}

		statuses = append(statuses, CertificateStatus{
			Domain: cert.Domain,
			Expiry: expiryDate.Format("2006-01-02"),
			Status: status,
		})
	}

	return statuses, nil
}

// CleanupServers cleans up the HTTP servers
func CleanupServers() {
	serverMutex.Lock()
	defer serverMutex.Unlock()

	if proxyServer != nil {
		proxyServer.Close()
		proxyServer = nil
	}
	if challengeServer != nil {
		challengeServer.Close()
		challengeServer = nil
	}
}
