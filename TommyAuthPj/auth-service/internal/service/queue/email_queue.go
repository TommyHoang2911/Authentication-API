package queue

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"auth-service/internal/service"
)

var (
	ErrQueueFull   = errors.New("email queue is full")
	ErrQueueClosed = errors.New("email queue is closed")
)

// EmailJob represents a unit of work for sending an email.
type EmailJob struct {
	To                string
	ConfirmationToken string
}

// Config controls async worker behavior and retry policy.
type Config struct {
	Workers    int
	BufferSize int
	MaxRetries int
	RetryDelay time.Duration
}

func (c Config) withDefaults() Config {
	if c.Workers <= 0 {
		c.Workers = 2
	}
	if c.BufferSize <= 0 {
		c.BufferSize = 100
	}
	if c.MaxRetries < 0 {
		c.MaxRetries = 0
	}
	if c.RetryDelay <= 0 {
		c.RetryDelay = time.Second
	}
	return c
}

// AsyncEmailSender enqueues jobs and processes them in background workers.
type AsyncEmailSender struct {
	sender service.EmailSender
	logger *log.Logger
	cfg    Config

	jobs   chan EmailJob
	stopCh chan struct{}
	wg     sync.WaitGroup

	mu     sync.RWMutex
	closed bool
}

// NewAsyncEmailSender starts worker goroutines immediately.
func NewAsyncEmailSender(sender service.EmailSender, cfg Config, logger *log.Logger) *AsyncEmailSender {
	if logger == nil {
		logger = log.Default()
	}

	q := &AsyncEmailSender{
		sender: sender,
		logger: logger,
		cfg:    cfg.withDefaults(),
		stopCh: make(chan struct{}),
	}
	q.jobs = make(chan EmailJob, q.cfg.BufferSize)

	for i := 0; i < q.cfg.Workers; i++ {
		q.wg.Add(1)
		go q.worker(i + 1)
	}

	return q
}

// SendRegistrationConfirmation enqueues an email job for async processing.
func (q *AsyncEmailSender) SendRegistrationConfirmation(to, confirmationToken string) error {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if q.closed {
		return ErrQueueClosed
	}

	job := EmailJob{To: to, ConfirmationToken: confirmationToken}
	select {
	case q.jobs <- job:
		return nil
	default:
		return ErrQueueFull
	}
}

func (q *AsyncEmailSender) worker(workerID int) {
	defer q.wg.Done()

	for job := range q.jobs {
		q.processWithRetry(workerID, job)
	}
}

func (q *AsyncEmailSender) processWithRetry(workerID int, job EmailJob) {
	var lastErr error
	maxAttempts := q.cfg.MaxRetries + 1

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := q.sender.SendRegistrationConfirmation(job.To, job.ConfirmationToken); err == nil {
			return
		} else {
			lastErr = err
		}

		if attempt < maxAttempts {
			backoff := time.Duration(attempt) * q.cfg.RetryDelay
			q.logger.Printf("email queue worker=%d attempt=%d failed, retrying in %s: %v", workerID, attempt, backoff, lastErr)
			timer := time.NewTimer(backoff)
			select {
			case <-timer.C:
			case <-q.stopCh:
				if !timer.Stop() {
					// Drain when the timer has already fired to avoid leaving a stale value.
					select {
					case <-timer.C:
					default:
					}
				}
				q.logger.Printf("email queue worker=%d aborted retry for to=%s during shutdown", workerID, job.To)
				return
			}
		}
	}

	q.logger.Printf("email queue worker=%d dropped email after %d attempts to=%s err=%v", workerID, maxAttempts, job.To, lastErr)
}

// Stops accepting new jobs and waits for workers; retries are aborted during shutdown.
func (q *AsyncEmailSender) Shutdown(ctx context.Context) error {
	q.mu.Lock()
	if q.closed {
		q.mu.Unlock()
		return nil
	}
	q.closed = true
	close(q.stopCh)
	close(q.jobs)
	q.mu.Unlock()

	done := make(chan struct{})
	go func() {
		q.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
