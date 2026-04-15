package queue

import (
	"context"
	"errors"
	"io"
	"log"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type flakySender struct {
	failUntil int32
	calls     int32
}

func (s *flakySender) SendRegistrationConfirmation(to, confirmationToken string) error {
	call := atomic.AddInt32(&s.calls, 1)
	if call <= atomic.LoadInt32(&s.failUntil) {
		return errors.New("temporary smtp error")
	}
	return nil
}

func (s *flakySender) CallCount() int32 {
	return atomic.LoadInt32(&s.calls)
}

type blockingSender struct {
	release <-chan struct{}
	onSend  func()
}

func (s *blockingSender) SendRegistrationConfirmation(to, confirmationToken string) error {
	if s.onSend != nil {
		s.onSend()
	}
	<-s.release
	return nil
}

func eventually(t *testing.T, timeout time.Duration, check func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if check() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	require.True(t, check(), "condition not met within timeout")
}

func TestAsyncEmailSender_RetryThenSuccess(t *testing.T) {
	sender := &flakySender{failUntil: 2}
	q := NewAsyncEmailSender(sender, Config{
		Workers:    1,
		BufferSize: 8,
		MaxRetries: 3,
		RetryDelay: time.Millisecond,
	}, log.New(io.Discard, "", 0))

	err := q.SendRegistrationConfirmation("user@example.com", "tok")
	require.NoError(t, err)

	eventually(t, 200*time.Millisecond, func() bool {
		return sender.CallCount() == 3
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, q.Shutdown(ctx))
}

func TestAsyncEmailSender_MaxRetriesExceeded(t *testing.T) {
	sender := &flakySender{failUntil: 100}
	q := NewAsyncEmailSender(sender, Config{
		Workers:    1,
		BufferSize: 8,
		MaxRetries: 2,
		RetryDelay: time.Millisecond,
	}, log.New(io.Discard, "", 0))

	err := q.SendRegistrationConfirmation("user@example.com", "tok")
	require.NoError(t, err)

	eventually(t, 200*time.Millisecond, func() bool {
		return sender.CallCount() == 3
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, q.Shutdown(ctx))
}

func TestAsyncEmailSender_QueueFull(t *testing.T) {
	release := make(chan struct{})
	started := make(chan struct{}, 1)
	once := sync.Once{}
	sender := &blockingSender{
		release: release,
		onSend: func() {
			once.Do(func() { close(started) })
		},
	}

	q := NewAsyncEmailSender(sender, Config{
		Workers:    1,
		BufferSize: 1,
		MaxRetries: 0,
		RetryDelay: time.Millisecond,
	}, log.New(io.Discard, "", 0))

	require.NoError(t, q.SendRegistrationConfirmation("a@example.com", "t1"))
	<-started

	require.NoError(t, q.SendRegistrationConfirmation("b@example.com", "t2"))
	err := q.SendRegistrationConfirmation("c@example.com", "t3")
	require.ErrorIs(t, err, ErrQueueFull)

	close(release)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, q.Shutdown(ctx))
}

func TestAsyncEmailSender_ClosedQueue(t *testing.T) {
	sender := &flakySender{}
	q := NewAsyncEmailSender(sender, Config{}, log.New(io.Discard, "", 0))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, q.Shutdown(ctx))

	err := q.SendRegistrationConfirmation("user@example.com", "tok")
	require.ErrorIs(t, err, ErrQueueClosed)
}

func TestAsyncEmailSender_ShutdownInterruptsBackoff(t *testing.T) {
	// alwaysFail never succeeds, so the worker will enter a backoff sleep.
	// We use a very long retry delay to ensure the worker is sleeping when
	// Shutdown is called; shutdown should unblock it promptly.
	sender := &flakySender{failUntil: 100}
	q := NewAsyncEmailSender(sender, Config{
		Workers:    1,
		BufferSize: 4,
		MaxRetries: 5,
		RetryDelay: 10 * time.Second, // long enough to still be sleeping at shutdown
	}, log.New(io.Discard, "", 0))

	require.NoError(t, q.SendRegistrationConfirmation("user@example.com", "tok"))

	// Wait until the first send attempt has been made (worker is about to sleep).
	eventually(t, time.Second, func() bool {
		return sender.CallCount() >= 1
	})

	// Shutdown with a short deadline; should return well before the 10 s backoff.
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, q.Shutdown(ctx))
	require.Less(t, time.Since(start), 5*time.Second, "shutdown should not wait for the full backoff")
}
