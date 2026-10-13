package mail

import (
	"context"
	"log"
	"sync"
	"time"
)

const (
	defaultQueueSize = 256
	defaultRetries   = 3
	sendTimeout      = 30 * time.Second
)

// Queue sends emails in the background so HTTP handlers never wait for SMTP
// and never expose delivery failures to the client.
type Queue struct {
	mailer  Mailer
	jobs    chan Message
	retries int
	backoff time.Duration
	wg      sync.WaitGroup
	closing chan struct{}
	mu      sync.RWMutex // guards closed so Enqueue never sends on a closed channel
	closed  bool
}

// NewQueue starts one worker. backoff is the first retry delay; it doubles
// after each failed attempt (1s, 2s, 4s by default).
func NewQueue(mailer Mailer, size int, backoff time.Duration) *Queue {
	if size <= 0 {
		size = defaultQueueSize
	}
	if backoff <= 0 {
		backoff = time.Second
	}
	q := &Queue{
		mailer: mailer, jobs: make(chan Message, size),
		retries: defaultRetries, backoff: backoff, closing: make(chan struct{}),
	}
	q.wg.Add(1)
	go q.work()
	return q
}

// Enqueue never blocks. It returns false when the queue is full or closed.
func (q *Queue) Enqueue(message Message) bool {
	q.mu.RLock()
	defer q.mu.RUnlock()
	if q.closed {
		return false
	}
	select {
	case q.jobs <- message:
		return true
	default:
		log.Printf("mail queue full, dropping message to=%s subject=%q", MaskAddress(message.To), message.Subject)
		return false
	}
}

func (q *Queue) work() {
	defer q.wg.Done()
	for message := range q.jobs {
		q.deliver(message)
	}
}

func (q *Queue) deliver(message Message) {
	delay := q.backoff
	for attempt := 1; attempt <= q.retries+1; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), sendTimeout)
		err := q.mailer.Send(ctx, message.To, message.Subject, message.Text, message.HTML)
		cancel()
		if err == nil {
			return
		}
		if attempt > q.retries {
			log.Printf("mail delivery failed after %d attempts to=%s subject=%q: %v", attempt, MaskAddress(message.To), message.Subject, err)
			return
		}
		log.Printf("mail delivery attempt %d failed to=%s, retrying in %s: %v", attempt, MaskAddress(message.To), delay, err)
		select {
		case <-time.After(delay):
		case <-q.closing:
			// Shutting down: finish this message quickly without waiting.
			delay = 0
		}
		delay *= 2
	}
}

// Close stops accepting messages and waits for queued ones until ctx ends.
func (q *Queue) Close(ctx context.Context) error {
	q.mu.Lock()
	if !q.closed {
		q.closed = true
		close(q.closing)
		close(q.jobs)
	}
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
