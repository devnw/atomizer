package engine

import (
	"context"

	"go.devnw.com/event"
)

type Option func(*Atomizer) error

type Publisher interface {
	ErrorFunc(context.Context, event.ErrorFunc)
	EventFunc(context.Context, event.EventFunc)
}

func WithPub(pub Publisher) Option {
	return func(a *Atomizer) error {
		a.pub = pub
		return nil
	}
}

func WithTransport(t ...Transport) Option {
	return func(a *Atomizer) error {
		// TODO: register the transports into the instance

		return nil
	}
}

func WithProcessors(p ...Processor) Option {
	return func(a *Atomizer) error {
		// TODO: register the processors into Atomizer

		return nil
	}
}
