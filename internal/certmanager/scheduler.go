package certmanager

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/O-tero/traefik-cert-manager/internal/config"
)

// Scheduler handles periodic certificate renewal checks
type Scheduler struct {
	config         *config.Config
	renewalService *RenewalService
	logger         *log.Logger
	ticker         *time.Ticker
	ctx            context.Context
	cancelFunc     context.CancelFunc
	wg             sync.WaitGroup
	isRunning      bool
	mu             sync.RWMutex
	lastRunTime    time.Time
	nextRunTime    time.Time
	stats          SchedulerStats
}

// SchedulerStats holds statistics about scheduler operations
type SchedulerStats struct {
	TotalRuns           int           `json:"total_runs"`
	SuccessfulRuns      int           `json:"successful_runs"`
	FailedRuns          int           `json:"failed_runs"`
	LastRunTime         time.Time     `json:"last_run_time"`
	LastRunDuration     time.Duration `json:"last_run_duration"`
	CertificatesRenewed int           `json:"certificates_renewed"`
	StartTime           time.Time     `json:"start_time"`
	NextRunTime         time.Time     `json:"next_run_time"`
}

func NewScheduler(cfg *config.Config, manager *CertificateManager, logger *log.Logger) (*Scheduler, error) {
	if logger == nil {
		logger = log.New(os.Stdout, "[Scheduler] ", log.LstdFlags)
	}

	checkInterval, err := cfg.GetCheckInterval()
	if err != nil {
		return nil, fmt.Errorf("invalid check interval: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	renewalService := NewRenewalService(manager, cfg.Certificates.StoragePath, logger)

	scheduler := &Scheduler{
		config:         cfg,
		renewalService: renewalService,
		logger:         logger,
		ticker:         time.NewTicker(checkInterval),
		ctx:            ctx,
		cancelFunc:     cancel,
		stats: SchedulerStats{
			StartTime: time.Now(),
		},
	}

	scheduler.nextRunTime = time.Now().Add(checkInterval)
	
	logger.Printf("Scheduler initialized with check interval: %v", checkInterval)
	return scheduler, nil
}

// Start begins the scheduler's periodic execution
func (s *Scheduler) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return fmt.Errorf("scheduler is already running")
	}

	s.logger.Printf("Starting certificate renewal scheduler")
	s.isRunning = true
	s.stats.StartTime = time.Now()

	s.wg.Add(1)
	go s.run()

	s.logger.Printf("Scheduler started successfully")
	return nil
}

// Stop stops the scheduler gracefully
func (s *Scheduler) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return fmt.Errorf("scheduler is not running")
	}

	s.logger.Printf("Stopping certificate renewal scheduler")
	
	// Signal shutdown
	s.cancelFunc()
	s.ticker.Stop()
	
	// Wait for goroutine to finish
	s.wg.Wait()
	
	s.isRunning = false
	s.renewalService.Stop()
	
	s.logger.Printf("Scheduler stopped successfully")
	return nil
}

// IsRunning returns true if the scheduler is currently running
func (s *Scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isRunning
}

// GetStats returns current scheduler statistics
func (s *Scheduler) GetStats() SchedulerStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	stats := s.stats
	if s.isRunning {
		stats.NextRunTime = s.nextRunTime
	}
	
	return stats
}

func (s *Scheduler) GetNextRunTime() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.nextRunTime
}

func (s *Scheduler) run() {
	defer s.wg.Done()
	
	s.logger.Printf("Scheduler main loop started")

	// Perform initial check after a short delay
	initialDelay := 30 * time.Second
	select {
	case <-time.After(initialDelay):
		s.performRenewalCheck()
	case <-s.ctx.Done():
		s.logger.Printf("Scheduler cancelled during initial delay")
		return
	}

	for {
		select {
		case <-s.ticker.C:
			s.performRenewalCheck()
		case <-s.ctx.Done():
			s.logger.Printf("Scheduler main loop stopped")
			return
		}
	}
}

// performRenewalCheck executes the certificate renewal check
func (s *Scheduler) performRenewalCheck() {
	startTime := time.Now()
	
	s.mu.Lock()
	s.stats.TotalRuns++
	s.stats.LastRunTime = startTime
	s.lastRunTime = startTime
	checkInterval, _ := s.config.GetCheckInterval()
	s.nextRunTime = startTime.Add(checkInterval)
	s.mu.Unlock()

	s.logger.Printf("Starting scheduled certificate renewal check (run #%d)", s.stats.TotalRuns)

	// Create a context with timeout for this operation
	timeout, err := s.config.GetTimeout()
	if err != nil {
		timeout = 10 * time.Minute // Default timeout
	}
	
	ctx, cancel := context.WithTimeout(s.ctx, timeout)
	defer cancel()

	// Perform the renewal process
	err = s.performRenewalWithContext(ctx)
	
	duration := time.Since(startTime)
	
	s.mu.Lock()
	s.stats.LastRunDuration = duration
	if err != nil {
		s.stats.FailedRuns++
		s.logger.Printf("Scheduled renewal check failed after %v: %v", duration, err)
	} else {
		s.stats.SuccessfulRuns++
		s.logger.Printf("Scheduled renewal check completed successfully in %v", duration)
	}
	s.mu.Unlock()
}

// performRenewalWithContext performs renewal with context cancellation support
func (s *Scheduler) performRenewalWithContext(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	health := s.renewalService.manager.CheckCertificateHealth()
	
	var renewalCount int
	var errors []error

	for domain, status := range health {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if status.NeedsRenewal {
			s.logger.Printf("Certificate for %s needs renewal (expires in %d days)", 
				domain, status.DaysUntilExpiry)
			
			if err := s.renewalService.manager.RenewCertificate(domain); err != nil {
				s.logger.Printf("Failed to renew certificate for %s: %v", domain, err)
				errors = append(errors, fmt.Errorf("failed to renew %s: %w", domain, err))
			} else {
				renewalCount++
				s.logger.Printf("Successfully renewed certificate for %s", domain)
			}
		}
	}

	s.mu.Lock()
	s.stats.CertificatesRenewed += renewalCount
	s.mu.Unlock()

	if len(errors) > 0 {
		return fmt.Errorf("renewal errors: %v", errors)
	}

	if renewalCount > 0 {
		s.logger.Printf("Renewed %d certificates during this check", renewalCount)
	} else {
		s.logger.Printf("No certificates needed renewal during this check")
	}

	return nil
}

// RunOnce performs a single renewal check outside of the regular schedule
func (s *Scheduler) RunOnce() error {
	s.logger.Printf("Performing manual certificate renewal check")
	
	timeout, err := s.config.GetTimeout()
	if err != nil {
		timeout = 10 * time.Minute
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	return s.performRenewalWithContext(ctx)
}

// Reschedule changes the scheduler interval
func (s *Scheduler) Reschedule(newInterval time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return fmt.Errorf("scheduler is not running")
	}

	s.logger.Printf("Rescheduling from %v to %v", s.ticker.C, newInterval)
	
	s.ticker.Stop()
	s.ticker = time.NewTicker(newInterval)
	s.nextRunTime = time.Now().Add(newInterval)
	
	return nil
}

func (s *Scheduler) GetUptime() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if s.isRunning {
		return time.Since(s.stats.StartTime)
	}
	
	return 0
}

// ResetStats resets the scheduler statistics
func (s *Scheduler) ResetStats() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.stats = SchedulerStats{
		StartTime: s.stats.StartTime, // Keep the start time
	}
	
	s.logger.Printf("Scheduler statistics reset")
}

// SchedulerStatus provides a summary of the scheduler state
type SchedulerStatus struct {
	IsRunning       bool          `json:"is_running"`
	Uptime          time.Duration `json:"uptime"`
	NextRunTime     time.Time     `json:"next_run_time"`
	LastRunTime     time.Time     `json:"last_run_time"`
	CheckInterval   string        `json:"check_interval"`
	Stats           SchedulerStats `json:"stats"`
}

func (s *Scheduler) GetStatus() SchedulerStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	interval, _ := s.config.GetCheckInterval()
	
	return SchedulerStatus{
		IsRunning:     s.isRunning,
		Uptime:        s.GetUptime(),
		NextRunTime:   s.nextRunTime,
		LastRunTime:   s.lastRunTime,
		CheckInterval: interval.String(),
		Stats:         s.stats,
	}
}