package atomizer

import (
	"context"

	"github.com/benjivesterby/validator"
	"github.com/pkg/errors"
)

// Atomizer interface implementation
type Atomizer interface {
	Exec() error
	Register(value ...interface{}) error
	Events(buffer int) <-chan string
	Errors(buffer int) <-chan error
	Properties(buffer int) (<-chan Properties, error)
	Wait()
}

// Atomize initialize instance of the atomizer to start reading from conductors and execute bonded electrons/atoms
func Atomize(ctx context.Context) Atomizer {
	return (&atomizer{ctx: ctx}).init()
}

// Exec kicks off the processing of the atomizer by pulling in the pre-registrations through init calls
// on imported libraries and starts up the receivers for atoms and conductors
func (a *atomizer) Exec() (err error) {

	// Execute on the atomizer should only ever be run once
	a.execSyncOnce.Do(func() {

		a.event("pulling conductor and atom registrations")

		// Start up the receivers
		if err = a.receive(Registrations(a.ctx)); err == nil {

			// Setup the distribution loop for incoming electrons
			// so that they can be properly fanned out to the atom
			// receivers
			go a.distribute()
		} else {
			err = aErr{err, "error while receiving registrations in atomizer execution"}
		}

		// TODO: Setup the instance receivers for monitoring of individual instances as well as sending of outbound electrons
	})

	return err
}

// Register allows you to add additional type registrations to the atomizer (ie. Conductors and Atoms)
func (a *atomizer) Register(values ...interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = aErr{errors.Errorf("%s", r), "panic occurred while registering elements in atomizer"}
		}
	}()

	for _, value := range values {
		if validator.Valid(value) {

			// Pass the value on the registrations channel to be received
			select {
			case <-a.ctx.Done():
				return
			case a.registrations <- value:
			}
		}
	}

	return err
}

// properties initializes the properties channel if it isn't already allocated and then returns the properties channel of
// the atomizer so that the requestor can start getting properties as processing finishes on their atoms
func (a *atomizer) Properties(buffer int) (<-chan Properties, error) {
	var err error

	if a.properties == nil {

		// Ensure that a proper buffer size was passed for the channel
		if buffer < 0 {
			buffer = 0
		}

		// Only upon request should the error channel be established meaning a user should read from the channel
		a.properties = make(chan Properties, buffer)
	}

	return a.properties, err
}

// Errors creates a channel to receive errors from the atomizer and return the channel for logging purposes
func (a *atomizer) Errors(buffer int) <-chan error {
	a.outputMutty.Lock()
	defer a.outputMutty.Unlock()

	if a.errors == nil {

		// Ensure that a proper buffer size was passed for the channel
		if buffer < 0 {
			buffer = 0
		}

		// Only upon request should the error channel be established meaning a user should read from the channel
		a.errors = make(chan error, buffer)
	}

	return a.errors
}

// Events creates a channel to receive events from the atomizer and return the channel for logging purposes
func (a *atomizer) Events(buffer int) <-chan string {
	a.outputMutty.Lock()
	defer a.outputMutty.Unlock()

	if a.events == nil {

		// Ensure that a proper buffer size was passed for the channel
		if buffer < 0 {
			buffer = 0
		}

		// Only upon request should the event channel be established meaning a user should read from the channel
		a.events = make(chan string, buffer)
	}

	return a.events
}

// Wait blocks on the context done channel to allow for the executable
// to block for the atomizer to finish processing
func (a *atomizer) Wait() {
	<-a.ctx.Done()
}
