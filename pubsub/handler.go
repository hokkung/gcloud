package pubsub

import (
	cloudpubsub "cloud.google.com/go/pubsub"
	"context"
)

// MessageHandler is the interface that subscribers must implement to process messages
type MessageHandler interface {
	ProcessMessage(ctx context.Context, msg *cloudpubsub.Message) error
}
