package pubsub

// SubscriberConfig holds configuration for a subscriber
type SubscriberConfig struct {
	ProjectID      string
	SubscriptionID string
	Handler        MessageHandler
}
