package notifier

import (
	"context"
	"fmt"
	"sync"

	"github.com/abhinavxd/libredesk/internal/attachment"
	"github.com/zerodha/logf"
)

const (
	ProviderEmail = "email"
)

// Message represents a message to be sent as a notification.
type Message struct {
	// Email addresses of the recipients
	RecipientEmails []string
	// Subject of the message
	Subject string
	// Body of the message
	Content string
	// Provider to send the message through
	Provider string
	// Attachments to be sent with the message
	Attachments []attachment.Attachment
	// Type of content ("plain" or "html")
	ContentType string
	// Alternative plain text version of the HTML content
	AltContent string
	// Additional email headers
	Headers map[string][]string
}

// Notifier defines the interface for sending notifications through various providers.
type Notifier interface {
	// Sends the notification message using the specified provider
	Send(message Message) error
	// Returns the name of the provider
	Name() string
}

// Service manages message providers and a worker pool.
type Service struct {
	providers      map[string]Notifier
	messageChannel chan Message
	concurrency    int
	lo             *logf.Logger
	closed         bool
	mu             sync.RWMutex
	wg             sync.WaitGroup
}

// NewService initializes the Service with given concurrency, channel capacity, and logger.
func NewService(providers map[string]Notifier, concurrency, capacity int, logger *logf.Logger) *Service {
	return &Service{
		providers:      providers,
		messageChannel: make(chan Message, capacity),
		concurrency:    concurrency,
		lo:             logger,
	}
}

// Send sends a message to the message channel.
func (s *Service) Send(message Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return fmt.Errorf("channel closed cannot send message")
	}

	select {
	case s.messageChannel <- message:
		return nil
	default:
		s.lo.Error("message channel is full")
		return fmt.Errorf("message channel is full")
	}
}

// Run starts the worker pool to process messages.
func (s *Service) Run(ctx context.Context) {
	for range s.concurrency {
		s.wg.Go(func() {
			s.worker(ctx)
		})
	}
	<-ctx.Done()
	s.Close()
}

// worker processes messages from the message channel and sends them using the set provider.
func (s *Service) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case message, ok := <-s.messageChannel:
			if !ok {
				return
			}
			provider, exists := s.providers[message.Provider]
			if !exists {
				s.lo.Error("unsupported provider", "provider", message.Provider)
				continue
			}
			if err := provider.Send(message); err != nil {
				s.lo.Error("error sending message", "error", err)
			}
		}
	}
}

// Close signals service to stop, closes the message channel and
// waits for all goroutine workers to finish.
func (s *Service) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.closed = true
	close(s.messageChannel)
	s.wg.Wait()
}
