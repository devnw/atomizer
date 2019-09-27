package atomizer

import (
	"context"

	"github.com/benjivesterby/validator"
	"github.com/pkg/errors"
)

// Atomizer interface implementation
type Atomizer interface {
	Exec() error
	Register(value interface{}) error
	Errors(buffer int) (<-chan error, error)
	Properties(buffer int) (<-chan Properties, error)
	Validate() bool
}

// Atomize initialize instance of the atomizer to start reading from conductors and execute bonded electrons/atoms
func Atomize(ctx context.Context) Atomizer {
	defer func() {
		if r := recover(); r != nil {
			// TODO:
			// err = errors.Errorf("panic in register, unable to register [%v]; [%s]", reflect.TypeOf(registration), r)
		}
	}()

	return (&atomizer{ctx: ctx}).init()
}

// Exec kicks off the processing of the atomizer by pulling in the pre-registrations through init calls
// on imported libraries and starts up the receivers for atoms and conductors
func (mizer *atomizer) Exec() (err error) {

	if validator.IsValid(mizer) {

		// Execute on the atomizer should only ever be run once
		mizer.execSyncOnce.Do(func() {

			// Start up the receivers
			err = mizer.receive(Registrations(mizer.ctx))

			// TODO: Setup the instance receivers for monitoring of individual instances as well as sending of outbound electrons
		})
	} else {
		// TODO:
	}

	return err
}

// Register allows you to add additional type registrations to the atomizer (ie. Conductors and Atoms)
func (mizer *atomizer) Register(value interface{}) (err error) {

	// validate the automizer initialization itself
	if validator.IsValid(mizer) {

		// Pass the value on the registrations channel to be received
		mizer.registrations <- value
	} else {
		err = errors.New("invalid object to register")
	}

	return err
}

// properties initializes the properties channel if it isn't already allocated and then returns the properties channel of
// the atomizer so that the requestor can start getting properties as processing finishes on their atoms
func (mizer *atomizer) Properties(buffer int) (<-chan Properties, error) {
	var err error

	// validate the automizer initialization itself
	if validator.IsValid(mizer) {
		if mizer.properties == nil {

			// Ensure that a proper buffer size was passed for the channel
			if buffer < 0 {
				buffer = 0
			}

			// Only upon request should the error channel be established meaning a user should read from the channel
			mizer.properties = make(chan Properties, buffer)
		}
	} else {
		err = errors.New("invalid atomizer")
	}

	return mizer.properties, err
}

// Errors creates a channel to receive errors from the atomizer and return the channel for logging purposes
func (mizer *atomizer) Errors(buffer int) (<-chan error, error) {
	var err error

	// validate the automizer initialization itself
	if validator.IsValid(mizer) {
		if mizer.errors == nil {

			// Ensure that a proper buffer size was passed for the channel
			if buffer < 0 {
				buffer = 0
			}

			// Only upon request should the error channel be established meaning a user should read from the channel
			mizer.errors = make(chan error, buffer)
		}
	} else {
		err = errors.New("invalid atomizer")
	}

	return mizer.errors, err
}

// Validate verifies that this instance of the atomizer is correctly initialized. This imports the validator library
// for extended use with the Validate method
func (mizer *atomizer) Validate() (valid bool) {
	valid = mizer.init() != nil

	return valid
}
