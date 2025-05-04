package pubsub

import (
	"context"
	"log"
	"sync"

	cloudpubsub "cloud.google.com/go/pubsub"
)

type SubscriberStatus string

const (
	Running SubscriberStatus = "running"
	Stopped SubscriberStatus = "stopped"
)

type Subscriber struct {
	subscription *cloudpubsub.Subscription
	client       *cloudpubsub.Client
	config       SubscriberConfig
	cancelFunc   context.CancelFunc
	status       SubscriberStatus
	mu           sync.Mutex
}

func NewSubscriber(client *cloudpubsub.Client, cfg SubscriberConfig) (*Subscriber, func(), error) {
	sub := &Subscriber{
		config:       cfg,
		client:       client,
		subscription: client.Subscription(cfg.SubscriptionID),
	}

	cleanup := func() {
		err := sub.Stop()
		if err != nil {
			log.Printf("error stopping subscriber: %v", err)
		}
	}

	return sub, cleanup, nil
}

func (s *Subscriber) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	s.cancelFunc = cancel

	return s.subscription.Receive(ctx, func(ctx context.Context, msg *cloudpubsub.Message) {
		if err := s.config.Handler.ProcessMessage(ctx, msg); err != nil {
			msg.Nack()
		} else {
			msg.Ack()
		}
	})
}

func (s *Subscriber) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.status == Running
}

func (s *Subscriber) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cancelFunc != nil {
		s.cancelFunc()
		s.cancelFunc = nil
		s.status = Stopped
	}

	if err := s.client.Close(); err != nil {
		return err
	}

	return nil
}
