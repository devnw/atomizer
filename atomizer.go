package atomizer

import (
	"context"
	"fmt"
	"github.com/benji-vesterby/validator"
	"github.com/pkg/errors"
	"reflect"
	"sync"
)

// Atomizer facilitates the execution of tasks (aka Electrons) which are received from the configured sources
// these electrons can be distributed across many instances of the Atomizer on different nodes in a distributed
// system or in memory. Atoms should be created to process "atomic" actions which are small in scope and overall
// processing complexity minimizing time to run and allowing for the distributed system to take on the burden of
// long running processes as a whole rather than a single process handling the overall load
type Atomizer struct {

	// Priority Channels
	electrons chan ewrappers

	// channel for passing the instance to a monitoring go routine
	instances chan instance

	errors chan error
	logs   chan string
	ctx    context.Context
	cancel context.CancelFunc

	// Map for storing the context cancellation functions for each source
	conductorCancelFuncs sync.Map
}

// Atomize initialize instance of the atomizer to start reading from conductors and execute bonded electrons/atoms
func Atomize(ctx context.Context) (atomizer *Atomizer, err error) {
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)

	// Initialize the atomizer and establish the channels
	atomizer = &Atomizer{
		electrons: make(chan ewrappers),
		instances: make(chan instance),
		ctx:       ctx,
		cancel:    cancel,
	}

	// TODO: Setup the instance receivers for monitoring of individual instances as well as sending of outbound electrons

	// Start up the receivers
	if err = atomizer.receive(); err == nil {

		// Initialize the bonding of electrons and atoms
		go atomizer.bond()
	}

	return atomizer, err
}

// Errors creates a channel to receive errors from the Atomizer and return the channel for logging purposes
func (atomizer *Atomizer) Errors(buffer int) <-chan error {
	if atomizer.errors == nil {

		// Ensure that a proper buffer size was passed for the channel
		if buffer < 0 {
			buffer = 0
		}

		// Only upon request should the error channel be established meaning a user should read from the channel
		atomizer.errors = make(chan error, buffer)
	}

	return atomizer.errors
}

// Logs creates a channel to receive logs from the Atomizer, and electrons
func (atomizer *Atomizer) Logs(buffer int) <-chan string {

	if atomizer.logs == nil {

		// Ensure that a proper buffer size was passed for the channel
		if buffer < 0 {
			buffer = 0
		}

		// Only upon request should the log channel be established meaning a user should read from the channel
		atomizer.logs = make(chan string, buffer)
	}

	return atomizer.logs
}

// If the error channel is not nil then send the error on the channel
func (atomizer *Atomizer) sendErr(err error) {
	if atomizer.errors != nil {
		go func(err error) {
			atomizer.errors <- err
		}(err)
	}
}

// if the log channel is not nil then send the log on the channel
func (atomizer *Atomizer) sendLog(log string) {
	if atomizer.logs != nil {
		go func(log string) {
			atomizer.logs <- log
		}(log)
	}
}

// Initialize the go routines that will read from the conductors concurrently while other parts of the
// atomizer reads in the inputs and executes the instances of electrons
func (atomizer *Atomizer) receive() (err error) {

	// Validate inputs
	if atomizer.ctx != nil {

		// Validate the atomizer instance
		if validator.IsValid(atomizer) {

			// Range over the sources and setup
			conductors.Range(func(key, value interface{}) bool {
				var ok bool

				select {
				case <-atomizer.ctx.Done(): // Do nothing so that the loop breaks immediately
				default:

					// Create the source context with a cancellation option and store the cancellation in a sync map
					ctx, ctxFunc := context.WithCancel(atomizer.ctx)
					atomizer.conductorCancelFuncs.Store(key, ctxFunc)

					var conductor Conductor
					if conductor, ok = value.(Conductor); ok {
						var electrons = conductor.Receive()

						if electrons != nil {

							// Push off the reading into it's own go routine so that it's concurrent
							go atomizer.distribute(ctx, key, electrons)
							ok = true
						} else {
							err = errors.Errorf("invalid electron channel for conductor [%s]", key)
						}
					}
				}

				return ok
			})
		} else {
			err = errors.Errorf("Invalid Atomizer [%v]", atomizer)
		}
	}

	return err
}

// Reading in from a specific electron channel of a conductor and drop it onto the atomizer channel for electrons
func (atomizer *Atomizer) distribute(ctx context.Context, conductor interface{}, electrons <-chan Electron) {
	defer func() {
		if r := recover(); r != nil {
			atomizer.sendLog(fmt.Sprintf("panic in distribute [%v] for conductor [%s]; restarting distributor", r, conductor))

			// Self Heal - Re-initialize atomizer distribute method in event of panic
			go atomizer.distribute(ctx, conductor, electrons)

		} else { // TODO: Determine if atomizer is the correct action if the electron channel is closed
			// clean up the cancel method for atomizer conductor and close the context
			if cFunc, ok := atomizer.conductorCancelFuncs.Load(conductor); ok {

				// Type assert the cancel method and execute it to close the context
				if cancel, ok := cFunc.(context.CancelFunc); ok {

					// Execute the cancel method for atomizer conductor
					cancel()
					atomizer.sendLog(fmt.Sprintf("context for conductor [%s] successfully cancelled", conductor))
				}
			} else {
				atomizer.sendErr(errors.Errorf("unable to find context cancellation function for conductor [%s]", conductor))
			}
		}
	}()

	// Read from the electron channel for atomizer conductor and push onto the atomizer electron channel for processing
	for {
		select {
		case <-ctx.Done():
			// Break the loop to close out the receiver
			atomizer.sendErr(errors.Errorf("context closed for distribution of conductor [%s]; exiting [%s]", conductor, ctx.Err().Error()))
			break
		case electron, ok := <-electrons:
			if ok {

				// pull the conductor from the
				if cond, err := atomizer.getConductor(conductor); err == nil {

					// Ensure that the electron being received is valid
					if validator.IsValid(electron) {

						atomizer.electrons <- ewrappers{electron, cond}
					} else {
						// TODO:
					}
				} else {
					// TODO: send an error showing that the processing for atomizer electron couldn't start because the
					//  conductor was unable to be pulled from the registry
				}
			} else { // Channel is closed, break out of the loop
				atomizer.sendErr(errors.Errorf("electron channel for conductor [%s] is closed, exiting read cycle", conductor))
				break
			}
		}
	}
}

// Combine an electron and an atom and initiate the processing of the atom in it's own go routine
func (atomizer *Atomizer) bond() {
	defer func() {
		if r := recover(); r != nil {
			atomizer.sendLog("panic in bond; re-initializing bonding")

			// Self Heal - Re-initialize the bondings
			go atomizer.bond()
		}
	}()

	var err error

	for {
		select {
		case <-atomizer.ctx.Done():
			// Context has closed, break the loop
			atomizer.sendErr(errors.Errorf("context has been closed for bonding; exiting [%s]", atomizer.ctx.Err().Error()))
			break
		case ewrap, ok := <-atomizer.electrons:
			if ok {

				// Execute the processing for atomizer electron using the atom that is registered
				go func(ewrap ewrappers) {
					// TODO: setup the handler for panics here, and determine what to do in the event of a failure

					// Create an electron instance object to contain internal state information about specific electrons
					var instance = instance{
						ewrap: ewrap,
					}

					// Initialize the new context object for the processing of the electron/atom
					if ewrap.electron.Timeout() != nil {
						// Create a new context for the electron with a timeout
						instance.ctx, instance.cancel = context.WithTimeout(atomizer.ctx, *ewrap.electron.Timeout())
					} else {
						// Create a new context for the electron with a cancellation option
						instance.ctx, instance.cancel = context.WithCancel(atomizer.ctx)
					}

					var atom Atom
					if atom, err = atomizer.getAtom(ewrap.electron.Atom()); err == nil {
						if validator.IsValid(atom) {
							instance.atom = atom

							// outbound channel, when setting up the channel for reading it needs to be passed as outbound
							var outbound = make(chan Electron)

							// Set the instance outbound channel for reading
							instance.outbound = outbound

							// Push the instance to the instances channel to be monitored by the Atomizer
							atomizer.instances <- instance

							// TODO: Log the execution of the process method here
							// Execute the process method of the atom
							var result []byte
							if result, err = instance.atom.Process(instance.ctx, instance.ewrap.electron, outbound); err == nil {
								// Close any of the monitoring go routines monitoring atomizer atom
								instance.cancel()

								// TODO: Ensure atomizer is the proper thing to do here?? I think it needs to close atomizer out
								//  at the conductor rather than here... unless the conductor overrode the call back
								// Execute the callback for the electron
								instance.ewrap.electron.Callback(result)
							} else {
								// TODO: process method returned an error
							}
						} else {
							// TODO:
						}
					} else {
						// TODO:
					}
				}(ewrap)

			} else {
				// Send error indicating that the atomizer is being disassembled due to a closed electron channel
				atomizer.sendErr(errors.New("electron channel closed; tearing down atomizer"))

				// Cancel the atomizer context to close down go routines
				atomizer.cancel()

				// Break the loop
				break
			}
		}
	}
}

// Load the atom from the registration sync map and retrieve a new instances
func (atomizer *Atomizer) getAtom(id string) (Atom, error) {
	var err error
	var atom Atom

	// Ensure atomizer atomizer is valid
	if validator.IsValid(atomizer) {

		// Load the atom from the sync.Map
		if a, ok := atoms.Load(id); ok {

			// Ensure the value that was registered wasn't stored nil
			if validator.IsValid(a) {

				// Initialize a new copy of the atom that was registered
				newAtom := reflect.New(reflect.TypeOf(a))

				// Type assert the new copy of the atom to an atom so that it can be used for processing
				// and returned as a pointer for bonding
				if atom, ok = newAtom.Interface().(Atom); ok {
					if !validator.IsValid(atom) {
						err = errors.Errorf("invalid atom stored for id [%s]", id)
					}
				} else {
					err = errors.Errorf("unable to type assert atom for id [%s]", id)
				}
			}
		} else {
			err = errors.Errorf("unable to load atom for id [%s]", id)
		}
	} else {
		err = errors.New("atomizer is invalid")
	}

	return atom, err
}

// Load a conductor from the sync map by key
func (atomizer *Atomizer) getConductor(key interface{}) (conductor Conductor, err error) {

	// Ensure that a proper key has been passed
	if validator.IsValid(key) {

		// Load the conductor from the sync map
		if cond, ok := conductors.Load(key); ok {

			// Type assert the conductor
			if conductor, ok = cond.(Conductor); ok {

				// Run the conductor through validation checks
				if !validator.IsValid(conductor) {
					err = errors.Errorf("invalid entry in conductors sync.Map for key [%v]", key)
				}
			} else {
				err = errors.Errorf("unable to type assert value to conductor for key [%v]", key)
			}
		} else {
			err = errors.Errorf("conductor is not registered in sync.Map for key [%v]", key)
		}
	} else {
		err = errors.Errorf("key [%v] for conductors sync.Map is invalid", key)
	}

	return conductor, err
}

// Validate verifies that this instance of the atomizer is correctly initialized. This imports the validator library
// for extended use with the Validate method
func (atomizer *Atomizer) Validate() (valid bool) {

	if atomizer.electrons != nil {
		valid = true
	}

	return valid
}

// Pull from a sync map containing cancellation functions for contexts
// if the sync map contains instances of context.CancelFunc then execute
// the cancellation method for that id in the sync map, otherwise error to
// the calling method
func (atomizer *Atomizer) nuke(cmap sync.Map, id string) (err error) {
	if len(id) > 0 {
		if cfunc, ok := cmap.Load(id); ok {
			if cancel, ok := cfunc.(context.CancelFunc); ok {

				// Cancel atomizer sub-contextS
				cancel()
			} else {
				// TODO:
			}
		} else {
			// TODO:
		}
	} else {
		// TODO:
	}

	return err
}
