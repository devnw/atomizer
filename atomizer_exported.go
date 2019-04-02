package atomizer

import (
	"context"

	"github.com/benji-vesterby/atomizer/interfaces"
	"github.com/benji-vesterby/validator"
	"github.com/pkg/errors"
)

// Atomize initialize instance of the atomizer to start reading from conductors and execute bonded electrons/atoms
func Atomize(ctx context.Context) (interfaces.Atomizer, error) {
	var mizer *atomizer
	var err error

	// If a nil context was passed then create a background context to be used instead
	if ctx == nil {
		ctx = context.Background()
	}

	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)

	// Initialize the atomizer and establish the channels
	mizer = &atomizer{
		electrons: make(chan ewrappers),
		instances: make(chan instance),
		ctx:       ctx,
		cancel:    cancel,
	}

	// TODO: Setup the instance receivers for monitoring of individual instances as well as sending of outbound electrons

	// Start up the receivers
	if err = mizer.receive(); err == nil {

		// Initialize the bonding of electrons and atoms
		go mizer.bond()
	}

	return mizer, err
}

// AddConductor allows you to add additional conductors to be received from after the atomizer has been created
func (mizer *atomizer) AddConductor(conductor interfaces.Conductor) (err error) {

	// validate the automizer initialization itself
	if validator.IsValid(mizer) {

		// Determine if the conductor is valid
		if validator.IsValid(conductor) {

			// Register the source in the sync map for the conductors
			if err = RegisterSource(conductor); err == nil {

				if _, err = mizer.receiveConductor(conductor); err != nil {
					err = errors.Errorf("error while receiving conductor [%s] : [%s]", conductor.ID(), err.Error())
				}
			} else {
				err = errors.Errorf("error while registering conductor [%s] : [%s]", conductor.ID(), err.Error())
			}
		} else {
			err = errors.Errorf("error while registering conductor. conductor is invalid.")
		}
	} else {
		err = errors.New("invalid atomizer")
	}

	return err
}

// properties initializes the properties channel if it isn't already allocated and then returns the properties channel of
// the atomizer so that the requestor can start getting properties as processing finishes on their atoms
func (mizer *atomizer) Properties(buffer int) (<-chan interfaces.Properties, error) {
	var err error

	// validate the automizer initialization itself
	if validator.IsValid(mizer) {
		if mizer.properties == nil {

			// Ensure that a proper buffer size was passed for the channel
			if buffer < 0 {
				buffer = 0
			}

			// Only upon request should the error channel be established meaning a user should read from the channel
			mizer.properties = make(chan interfaces.Properties, buffer)
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

// Logs creates a channel to receive logs from the atomizer, and electrons
func (mizer *atomizer) Logs(buffer int) (<-chan string, error) {
	var err error

	// validate the automizer initialization itself
	if validator.IsValid(mizer) {
		if mizer.logs == nil {

			// Ensure that a proper buffer size was passed for the channel
			if buffer < 0 {
				buffer = 0
			}

			// Only upon request should the log channel be established meaning a user should read from the channel
			mizer.logs = make(chan string, buffer)
		}
	} else {
		err = errors.New("invalid atomizer")
	}

	return mizer.logs, err
}

// Validate verifies that this instance of the atomizer is correctly initialized. This imports the validator library
// for extended use with the Validate method
func (mizer *atomizer) Validate() (valid bool) {

	// Ensure the electrons channel is initialized
	if mizer.electrons != nil {

		// Ensure the instances channel to pass out for monitoring is initialized
		if mizer.instances != nil {

			// Ensure a valid context has been passed to the mizer
			if mizer.ctx != nil {

				// Ensure that the cancel method of the new context has been registered in the mizer so that
				// the mizer can be torn down internally
				if mizer.cancel != nil {
					valid = true
				}
			}
		}
	}

	return valid
}