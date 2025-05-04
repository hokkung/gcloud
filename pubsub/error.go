package pubsub

import "errors"

var (
	ErrSubscriberAlreadyExists = errors.New("subscriber already exists")
	ErrSubscriberNotFound      = errors.New("subscriber not found")
	ErrSubscriberIsRunning     = errors.New("subscriber status is running")
)
