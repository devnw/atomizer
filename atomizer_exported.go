// Copyright © 2019 Developer Network, LLC
//
// This file is subject to the terms and conditions defined in
// file 'LICENSE', which is part of this source code package.

package engine

import (
	"context"
	"fmt"

	"go.devnw.com/event"
	"go.devnw.com/validator"
)

// Atomizer interface implementation
type Atomizer interface {
	Exec() error
	Register(value ...interface{}) error
	Wait()
	Publisher() *event.Publisher

	// private methods enforce only this
	// package can return an atomizer
	isAtomizer()
}

// Atomize initialize instance of the atomizer to start reading from
// conductors and execute bonded electrons/atoms
//
// NOTE: Registrations can be added through this method and OVERRIDE any
// existing registrations of the same Atom or Conductor.
func Atomize(
	ctx context.Context,
	registrations ...interface{},
) (Atomizer, error) {
	err := Register(registrations...)
	if err != nil {
		return nil, err
	}

	ctx, cancel := _ctx(ctx)

	return &atomizer{
		ctx:           ctx,
		cancel:        cancel,
		electrons:     make(chan instance),
		bonded:        make(chan instance),
		registrations: make(chan interface{}),
		atoms:         make(map[string]chan<- instance),
		publisher:     event.NewPublisher(ctx),
	}, nil
}

func (*atomizer) isAtomizer() {}

// Publisher returns the atomizer's event publisher
func (a *atomizer) Publisher() *event.Publisher { return a.publisher }

// Exec kicks off the processing of the atomizer by pulling in the
// pre-registrations through init calls on imported libraries and
// starts up the receivers for atoms and conductors
func (a *atomizer) Exec() (err error) {
	// Execute on the atomizer should only ever be run once
	a.execSyncOnce.Do(func() {
		defer a.publisher.EventFunc(a.ctx, func() event.Event {
			return makeEvent("pulling conductor and atom registrations")
		})

		// Initialize the registrations in the Atomizer package
		for _, r := range Registrations() {
			a.register(r)
		}

		// Start up the receivers
		go a.receive()

		// Setup the distribution loop for incoming electrons
		// so that they can be properly fanned out to the
		// atom receivers
		go a.distribute()

		// TODO: Setup the instance receivers for monitoring of
		// individual instances as well as sending of outbound
		// electrons
	})

	return err
}

// Register allows you to add additional type registrations to the atomizer
// (ie. Conductors and Atoms)
func (a *atomizer) Register(values ...interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = &Error{
				Event: &Event{
					Message: "panic in atomizer",
				},
				Internal: ptoe(r),
			}
		}
	}()

	for _, value := range values {
		if !validator.Valid(value) {
			// TODO: create event here indicating that
			// a value was invalid and not registered
			continue
		}

		switch v := value.(type) {
		case Conductor, Atom:
			// Pass the value on the registrations
			// channel to be received
			select {
			case <-a.ctx.Done():
				return simple("context closed", nil)
			case a.registrations <- v:
			}
		default:
			return simple(
				fmt.Sprintf(
					"invalid value in registration %s",
					ID(value),
				),
				nil,
			)
		}
	}

	return err
}

// Wait blocks on the context done channel to allow for the executable
// to block for the atomizer to finish processing
func (a *atomizer) Wait() {
	<-a.ctx.Done()
}
