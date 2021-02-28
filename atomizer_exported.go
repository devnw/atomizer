// Copyright © 2019 Developer Network, LLC
//
// This file is subject to the terms and conditions defined in
// file 'LICENSE', which is part of this source code package.

package engine

import (
	"context"
	"fmt"

	"devnw.com/validator"
)

// Atomizer interface implementation
type Atomizer interface {
	Exec() error
	Register(value ...interface{}) error
	Events(buffer int) <-chan interface{}
	Wait()

	// private methods enforce only this
	// package can return an atomizer
	init(context.Context) *atomizer
}

// Atomize initialize instance of the atomizer to start reading from
// conductors and execute bonded electrons/atoms
//
// NOTE: Registrations can be added through this method and OVERRIDE any
// existing registrations of the same Atom or Conductor.
func Atomize(
	ctx context.Context,
	events chan interface{},
	registrations ...interface{},
) (Atomizer, error) {

	err := Register(registrations...)
	if err != nil {
		return nil, err
	}

	return (&atomizer{events: events}).init(ctx), nil
}

// Exec kicks off the processing of the atomizer by pulling in the
// pre-registrations through init calls on imported libraries and
// starts up the receivers for atoms and conductors
func (a *atomizer) Exec() (err error) {

	// Execute on the atomizer should only ever be run once
	a.execSyncOnce.Do(func() {

		a.event("pulling conductor and atom registrations")

		// Start up the receivers
		go a.receive()

		// Setup the distribution loop for incoming electrons
		// so that they can be properly fanned out to the
		// atom receivers
		go a.distribute()

		// TODO: Setup the instance receivers for monitoring of
		// individual instances as well as sending of outbound
		// electrons

		//err = a.Register(Registrations()...)
	})

	return err
}

// Register allows you to add additional type registrations to the atomizer
// (ie. Conductors and Atoms)
func (a *atomizer) Register(values ...interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = Error{
				Event: Event{
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

// Events creates a channel to receive events from the atomizer and
// return the channel for logging purposes
func (a *atomizer) Events(buffer int) <-chan interface{} {

	if buffer < 0 {
		buffer = 0
	}

	a.eventsMu.Lock()
	defer a.eventsMu.Unlock()

	if a.events == nil {
		a.events = make(chan interface{}, buffer)
	}

	return a.events
}

// Wait blocks on the context done channel to allow for the executable
// to block for the atomizer to finish processing
func (a *atomizer) Wait() {
	<-a.ctx.Done()
}
