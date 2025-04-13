package scheduler

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/luongquochai/s3-performance-test/internal/config"
)

// JobFunc defines a function that will be executed on schedule
type JobFunc func(ctx context.Context) error

// Scheduler manages scheduled jobs using cron expressions
type Scheduler struct {
	scheduler *gocron.Scheduler
	jobFunc   JobFunc
	cfg       *config.Config
	isRunning bool
}

// NewScheduler creates a new scheduler with the given configuration
func NewScheduler(cfg *config.Config, jobFunc JobFunc) *Scheduler {
	scheduler := gocron.NewScheduler(time.UTC)
	return &Scheduler{
		scheduler: scheduler,
		jobFunc:   jobFunc,
		cfg:       cfg,
		isRunning: false,
	}
}

// Start begins scheduling jobs according to the cron expression
func (s *Scheduler) Start(ctx context.Context) error {
	if !s.cfg.Schedule.Enabled {
		return fmt.Errorf("scheduler is not enabled in configuration")
	}

	if s.isRunning {
		return fmt.Errorf("scheduler is already running")
	}

	// Parse the cron expression
	cronSchedule := s.cfg.Schedule.Cron
	if cronSchedule == "" {
		return fmt.Errorf("cron expression is empty")
	}

	// Schedule the job using the cron expression
	_, err := s.scheduler.Cron(cronSchedule).Do(func() {
		jobCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
		defer cancel()

		log.Printf("Running scheduled job at %s", time.Now().Format(time.RFC3339))
		if err := s.jobFunc(jobCtx); err != nil {
			log.Printf("Scheduled job failed: %v", err)
		} else {
			log.Printf("Scheduled job completed successfully")
		}
	})

	if err != nil {
		return fmt.Errorf("failed to schedule job: %w", err)
	}

	// Start the scheduler
	s.scheduler.StartAsync()
	s.isRunning = true
	log.Printf("Scheduler started with cron expression: %s", cronSchedule)
	return nil
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	if s.isRunning {
		s.scheduler.Stop()
		s.isRunning = false
		log.Println("Scheduler stopped")
	}
}

// IsRunning returns whether the scheduler is currently running
func (s *Scheduler) IsRunning() bool {
	return s.isRunning
}

// NextRun returns the time of the next scheduled run
func (s *Scheduler) NextRun() (time.Time, error) {
	if !s.isRunning {
		return time.Time{}, fmt.Errorf("scheduler is not running")
	}

	jobs := s.scheduler.Jobs()
	if len(jobs) == 0 {
		return time.Time{}, fmt.Errorf("no jobs scheduled")
	}

	return jobs[0].NextRun(), nil
}
