package atomizer

import (
	"context"

	"github.com/devnw/validator"
)

// Atomizer interface implementation
type Atomizer interface {
	Exec() error
	Register(value ...interface{}) error
	Events(buffer int) <-chan interface{}
	Properties(buffer int) (<-chan Properties, error)
	Wait()

	// private methods enforce only this
	// package can return an atomizer
	init(context.Context) *atomizer
}

// Atomize initialize instance of the atomizer to start reading from
// conductors and execute bonded electrons/atoms
func Atomize(ctx context.Context, events chan interface{}) Atomizer {
	return (&atomizer{events: events}).init(ctx)
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
	defer func() { err = rec() }()

	for _, v := range values {
		if !validator.Valid(v) {
			// TODO: create event here indicating that
			// a value was invalid and not registered
			continue
		}

		// Pass the value on the registrations
		// channel to be received
		select {
		case <-a.ctx.Done():
			return
		case a.registrations <- v:
		}

	}

	return err
}

// Properties initializes the properties channel if it isn't already
// allocated and then returns the properties channel of the atomizer
// so that the requestor can start getting properties as processing
// finishes on their atoms
// XXX(benji): Figure out what the original purpose of this was
func (a *atomizer) Properties(buffer int) (<-chan Properties, error) {
	var err error

	if buffer < 0 {
		buffer = 0
	}

	if a.properties == nil {

		a.properties = make(chan Properties, buffer)
	}

	return a.properties, err
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

	// XXX(benji) ensure that is is correct and wont panic
	// this should also clean up the atom specific channels
	close(a.electrons)
	close(a.bonded)
	close(a.properties)
	close(a.events)
	close(a.registrations)
}
