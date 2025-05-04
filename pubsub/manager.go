package pubsub

import (
	cloudpubsub "cloud.google.com/go/pubsub"
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
)

// SubscriberManager manages multiple Pub/Sub subscribers
type SubscriberManager struct {
	subscribers map[string]*Subscriber
	wg          sync.WaitGroup
	mu          sync.Mutex
}

// NewSubscriberManager creates a new SubscriberManager
func NewSubscriberManager() (*SubscriberManager, func(), error) {
	manager := &SubscriberManager{
		subscribers: make(map[string]*Subscriber),
	}

	cleanUp := func() {
		if err := manager.Close(); err != nil {
			log.Printf("unable to close subscribers: %+v", err)
		}
	}

	return manager, cleanUp, nil
}

// Add adds a new subscriber configuration
func (m *SubscriberManager) Add(name string, cfg SubscriberConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.subscribers[name]; exists {
		return ErrSubscriberAlreadyExists
	}

	client, err := cloudpubsub.NewClient(context.Background(), cfg.ProjectID)
	if err != nil {
		return err
	}

	sub, _, err := NewSubscriber(client, cfg)
	if err != nil {
		return err
	}

	m.subscribers[name] = sub

	return nil
}

// StartAllSubscribers starts all configured subscribers
func (m *SubscriberManager) StartAllSubscribers(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, subscriber := range m.subscribers {
		if err := m.startSubscriber(ctx, name, subscriber); err != nil {
			return err
		}
	}

	return nil
}

// StartSubscriber starts a specific subscriber by name
func (m *SubscriberManager) StartSubscriber(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	subscriber, exists := m.subscribers[name]
	if !exists {
		return ErrSubscriberNotFound
	}

	return m.startSubscriber(ctx, name, subscriber)
}

func (m *SubscriberManager) startSubscriber(ctx context.Context, name string, subscriber *Subscriber) error {
	subscriber, exists := m.subscribers[name]
	if !exists {
		return ErrSubscriberNotFound
	}

	if subscriber.IsRunning() {
		return ErrSubscriberIsRunning
	}

	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		log.Printf("Starting subscriber: %s", name)

		err := subscriber.Start(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("Subscriber %s error: %v", name, err)
		}

		log.Printf("Subscriber %s stopped", name)
	}()

	return nil
}

// StopAllSubscribers stops all running subscribers
func (m *SubscriberManager) StopAllSubscribers() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error
	for name, subscriber := range m.subscribers {
		if err := subscriber.Stop(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close subscriber %s: %w", name, err))
		}
		delete(m.subscribers, name)
	}

	return errors.Join(errs...)
}

// StopSubscriber stops a specific subscriber by name
func (m *SubscriberManager) StopSubscriber(name string) error {
	m.mu.Lock()
	subscriber, exists := m.subscribers[name]
	m.mu.Unlock()

	if !exists {
		return ErrSubscriberNotFound
	}

	if err := subscriber.Stop(); err != nil {
		return err
	}

	delete(m.subscribers, name)

	return nil
}

// Wait waits for all subscribers to finish
func (m *SubscriberManager) Wait() {
	m.wg.Wait()
}

// Close cleans up all resources
func (m *SubscriberManager) Close() error {
	if err := m.StopAllSubscribers(); err != nil {
		return err
	}
	m.Wait()

	return nil
}
