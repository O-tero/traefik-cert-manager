package certmanager

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RenewalChecker provides methods for checking certificate renewal status
type RenewalChecker struct {
	logger      *log.Logger
	storagePath string
}

func NewRenewalChecker(storagePath string, logger *log.Logger) *RenewalChecker {
	if logger == nil {
		logger = log.New(os.Stdout, "[RenewalChecker] ", log.LstdFlags)
	}

	return &RenewalChecker{
		logger:      logger,
		storagePath: storagePath,
	}
}

// NeedsRenewal checks if a certificate needs renewal based on file path
func (rc *RenewalChecker) NeedsRenewal(certPath string) bool {
	keyPath := certPath + ".key"
	
	// Check if files exist
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		rc.logger.Printf("Certificate file not found: %s", certPath)
		return false
	}
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		rc.logger.Printf("Key file not found: %s", keyPath)
		return false
	}

	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		rc.logger.Printf("Failed to load certificate %s: %v", certPath, err)
		return false
	}

	// Check if certificate has leaf certificate
	if cert.Leaf == nil {
		if len(cert.Certificate) == 0 {
			rc.logger.Printf("Certificate %s has no certificate data", certPath)
			return false
		}

		// Get the first certificate in the chain and parse it as x509.Certificate
		leafCert, err := x509.ParseCertificate(cert.Certificate[0])
		if err != nil {
			rc.logger.Printf("Failed to parse certificate %s: %v", certPath, err)
			return false
		}
		cert.Leaf = leafCert
	}

	if cert.Leaf == nil {
		rc.logger.Printf("Certificate %s has no leaf certificate", certPath)
		return false
	}

	expiry := cert.Leaf.NotAfter
	timeUntilExpiry := time.Until(expiry)
	renewalThreshold := 30 * 24 * time.Hour // 30 days

	needsRenewal := timeUntilExpiry < renewalThreshold
	
	if needsRenewal {
		rc.logger.Printf("Certificate %s needs renewal (expires in %v)", certPath, timeUntilExpiry)
	}

	return needsRenewal
}

// NeedsRenewalByDomain checks if a certificate needs renewal by domain name
func (rc *RenewalChecker) NeedsRenewalByDomain(domain string) bool {
	certPath := filepath.Join(rc.storagePath, domain+".crt")
	return rc.NeedsRenewal(certPath)
}

func (rc *RenewalChecker) GetCertificateExpiry(certPath string) (time.Time, error) {
	keyPath := certPath + ".key"
	
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to load certificate: %w", err)
	}

	if cert.Leaf == nil {
		if len(cert.Certificate) == 0 {
			return time.Time{}, fmt.Errorf("certificate has no data")
		}
		// This is a workaround since tls.LoadX509KeyPair doesn't always populate Leaf
		return time.Time{}, fmt.Errorf("certificate leaf not available, use alternative method")
	}

	return cert.Leaf.NotAfter, nil
}

func (rc *RenewalChecker) GetAllCertificates() ([]string, error) {
	var certificates []string

	if _, err := os.Stat(rc.storagePath); os.IsNotExist(err) {
		rc.logger.Printf("Storage directory does not exist: %s", rc.storagePath)
		return certificates, nil
	}

	entries, err := os.ReadDir(rc.storagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read storage directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".crt" {
			// Skip issuer certificates
			if filepath.Base(entry.Name()) == "issuer.crt" {
				continue
			}
			
			certPath := filepath.Join(rc.storagePath, entry.Name())
			certificates = append(certificates, certPath)
		}
	}

	return certificates, nil
}

// CheckAllCertificates checks all certificates for renewal needs
func (rc *RenewalChecker) CheckAllCertificates() ([]string, error) {
	certificates, err := rc.GetAllCertificates()
	if err != nil {
		return nil, fmt.Errorf("failed to get certificates: %w", err)
	}

	var needsRenewal []string
	for _, certPath := range certificates {
		if rc.NeedsRenewal(certPath) {
			needsRenewal = append(needsRenewal, certPath)
		}
	}

	rc.logger.Printf("Found %d certificates that need renewal out of %d total", 
		len(needsRenewal), len(certificates))

	return needsRenewal, nil
}

// RenewalTask represents a certificate renewal task
type RenewalTask struct {
	Domain      string
	CertPath    string
	KeyPath     string
	Priority    int       
	ScheduledAt time.Time
}

// RenewalQueue manages renewal tasks
type RenewalQueue struct {
	tasks  []RenewalTask
	logger *log.Logger
}

func NewRenewalQueue(logger *log.Logger) *RenewalQueue {
	if logger == nil {
		logger = log.New(os.Stdout, "[RenewalQueue] ", log.LstdFlags)
	}

	return &RenewalQueue{
		tasks:  make([]RenewalTask, 0),
		logger: logger,
	}
}

// AddTask adds a renewal task to the queue
func (rq *RenewalQueue) AddTask(task RenewalTask) {
	rq.tasks = append(rq.tasks, task)
	rq.logger.Printf("Added renewal task for domain: %s", task.Domain)
}

func (rq *RenewalQueue) GetNextTask() *RenewalTask {
	if len(rq.tasks) == 0 {
		return nil
	}

	// Find task with highest priority that's ready to be executed
	var nextTask *RenewalTask
	var nextIndex int = -1
	
	for i, task := range rq.tasks {
		if time.Now().After(task.ScheduledAt) {
			if nextTask == nil || task.Priority > nextTask.Priority {
				nextTask = &task
				nextIndex = i
			}
		}
	}

	if nextTask != nil && nextIndex >= 0 {
		rq.tasks = append(rq.tasks[:nextIndex], rq.tasks[nextIndex+1:]...)
	}

	return nextTask
}

// HasPendingTasks returns true if there are pending tasks
func (rq *RenewalQueue) HasPendingTasks() bool {
	return len(rq.tasks) > 0
}

func (rq *RenewalQueue) Clear() {
	rq.tasks = make([]RenewalTask, 0)
	rq.logger.Printf("Cleared all renewal tasks")
}

// GetPendingCount returns the number of pending tasks
func (rq *RenewalQueue) GetPendingCount() int {
	count := 0
	now := time.Now()
	for _, task := range rq.tasks {
		if now.After(task.ScheduledAt) {
			count++
		}
	}
	return count
}

// RenewalService orchestrates the certificate renewal process
type RenewalService struct {
	checker    *RenewalChecker
	queue      *RenewalQueue
	manager    *CertificateManager
	logger     *log.Logger
	ctx        context.Context
	cancelFunc context.CancelFunc
}

// NewRenewalService creates a new renewal service
func NewRenewalService(manager *CertificateManager, storagePath string, logger *log.Logger) *RenewalService {
	if logger == nil {
		logger = log.New(os.Stdout, "[RenewalService] ", log.LstdFlags)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &RenewalService{
		checker:    NewRenewalChecker(storagePath, logger),
		queue:      NewRenewalQueue(logger),
		manager:    manager,
		logger:     logger,
		ctx:        ctx,
		cancelFunc: cancel,
	}
}

func (rs *RenewalService) ProcessRenewals() error {
	rs.logger.Printf("Starting renewal process")

	certificates, err := rs.checker.CheckAllCertificates()
	if err != nil {
		return fmt.Errorf("failed to check certificates: %w", err)
	}

	if len(certificates) == 0 {
		rs.logger.Printf("No certificates need renewal")
		return nil
	}

	var errors []error
	for _, certPath := range certificates {
		domain := rs.extractDomainFromPath(certPath)
		if domain == "" {
			rs.logger.Printf("Could not extract domain from path: %s", certPath)
			continue
		}

		rs.logger.Printf("Processing renewal for domain: %s", domain)
		
		if err := rs.manager.RenewCertificate(domain); err != nil {
			rs.logger.Printf("Failed to renew certificate for %s: %v", domain, err)
			errors = append(errors, fmt.Errorf("renewal failed for %s: %w", domain, err))
		} else {
			rs.logger.Printf("Successfully renewed certificate for %s", domain)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("renewal errors occurred: %v", errors)
	}

	return nil
}

func (rs *RenewalService) extractDomainFromPath(certPath string) string {
	filename := filepath.Base(certPath)
	if !strings.HasSuffix(filename, ".crt") {
		return ""
	}
	
	// Remove .crt extension
	domain := filename[:len(filename)-4]
	
	// Skip issuer certificates
	if domain == "issuer" {
		return ""
	}
	
	return domain
}


// Stop stops the renewal service
func (rs *RenewalService) Stop() {
	rs.logger.Printf("Stopping renewal service")
	if rs.cancelFunc != nil {
		rs.cancelFunc()
	}
}