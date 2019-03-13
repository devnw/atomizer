package atomizer

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/benji-vesterby/validator"
	"github.com/pkg/errors"
)

// Atomizer interface implementation
type Atomizer interface {
	AddConductor(name string, conductor Conductor) error
	Errors(buffer int) (<-chan error, error)
	Logs(buffer int) (<-chan string, error)
	Properties(buffer int) (<-chan Properties, error)
	Validate() (valid bool)
}

// Atomize initialize instance of the atomizer to start reading from conductors and execute bonded electrons/atoms
func Atomize(ctx context.Context) (Atomizer, error) {
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

// atomizer facilitates the execution of tasks (aka Electrons) which are received from the configured sources
// these electrons can be distributed across many instances of the atomizer on different nodes in a distributed
// system or in memory. Atoms should be created to process "atomic" actions which are small in scope and overall
// processing complexity minimizing time to run and allowing for the distributed system to take on the burden of
// long running processes as a whole rather than a single process handling the overall load
type atomizer struct {

	// Priority Channels
	electrons chan ewrappers

	// channel for passing the instance to a monitoring go routine
	instances chan instance

	errors     chan error
	logs       chan string
	properites chan Properties
	ctx        context.Context
	cancel     context.CancelFunc

	// Map for storing the context cancellation functions for each source
	conductorCancelFuncs sync.Map
}

// AddConductor allows you to add additional conductors to be received from after the atomizer has been created
func (mizer *atomizer) AddConductor(name string, conductor Conductor) (err error) {

	// validate the automizer initialization itself
	if validator.IsValid(mizer) {

		// Ensure the name for this conductor is valid
		if len(name) > 0 {

			// Determine if the conductor is valid
			if validator.IsValid(conductor) {

				// Register the source in the sync map for the conductors
				if err = RegisterSource(name, conductor); err == nil {

					if _, err = mizer.receiveConductor(name, conductor); err != nil {
						err = errors.Errorf("error while receiving conductor [%s] : [%s]", name, err.Error())
					}
				} else {
					err = errors.Errorf("error while registering conductor [%s] : [%s]", name, err.Error())
				}
			} else {
				err = errors.Errorf("error while registering conductor [%s]. conductor is invalid.", name)
			}
		} else {
			err = errors.New("attempted to add a conductor with an empty name")
		}
	} else {
		err = errors.New("invalid atomizer")
	}

	return err
}

// properitess initializes the properitess channel if it isn't already allocated and then returns the properites channel of
// the atomizer so that the requestor can start getting properitess as processing finishes on their atoms
func (mizer *atomizer) Properties(buffer int) (<-chan Properties, error) {
	var err error

	// validate the automizer initialization itself
	if validator.IsValid(mizer) {
		if mizer.properites == nil {

			// Ensure that a proper buffer size was passed for the channel
			if buffer < 0 {
				buffer = 0
			}

			// Only upon request should the error channel be established meaning a user should read from the channel
			mizer.properites = make(chan Properties, buffer)
		}
	} else {
		err = errors.New("invalid atomizer")
	}

	return mizer.properites, err
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

// If the error channel is not nil then send the error on the channel
func (mizer *atomizer) sendErr(err error) {
	if validator.IsValid(mizer) {
		if mizer.errors != nil {
			mizer.errors <- err
		}
	}
}

// if the log channel is not nil then send the log on the channel
func (mizer *atomizer) sendLog(log string) {
	if validator.IsValid(mizer) {
		if mizer.logs != nil {
			mizer.logs <- log
		}
	}
}

// Initialize the go routines that will read from the conductors concurrently while other parts of the
// atomizer reads in the inputs and executes the instances of electrons
func (mizer *atomizer) receive() (err error) {

	// Validate the mizer instance
	if validator.IsValid(mizer) {

		// Range over the sources and setup
		conductors.Range(func(key, value interface{}) bool {
			var ok bool

			select {
			case <-mizer.ctx.Done(): // Do nothing so that the loop breaks immediately
			default:

				ok, err = mizer.receiveConductor(key, value)

			}

			return ok
		})
	} else {
		err = errors.Errorf("Invalid mizer [%v]", mizer)
	}

	return err
}

// recieveConductor setups a retrieval loop for the conductor being passed in
func (mizer *atomizer) receiveConductor(key interface{}, value interface{}) (ok bool, err error) {
	if validator.IsValid(mizer) {

		// Ensure this is a valid conductor
		if ok = validator.IsValid(value); ok {

			// Create the source context with a cancellation option and store the cancellation in a sync map
			ctx, ctxFunc := context.WithCancel(mizer.ctx)
			mizer.conductorCancelFuncs.Store(key, ctxFunc)

			// Type assert the conductor as a Conductor interface
			var conductor Conductor
			if conductor, ok = value.(Conductor); ok {

				// Pull the receive channel from the conductor for new electrons
				var electrons = conductor.Receive()

				// Ensure the channel is valid
				if electrons != nil {

					// Push off the reading into it's own go routine so that it's concurrent
					mizer.distribute(ctx, key, electrons)
					ok = true
				} else {
					err = errors.Errorf("invalid electron channel for conductor [%s]", key)
				}
			} else {
				err = errors.Errorf("invalid conductor [%v] failed type assertion.", key)
			}

		} else {
			err = errors.Errorf("invalid conductor [%v] set to be received", key)
		}

	} else {
		err = errors.New("invalid atomizer")
	}

	return ok, err
}

// Reading in from a specific electron channel of a conductor and drop it onto the atomizer channel for electrons
func (mizer *atomizer) distribute(ctx context.Context, conductor interface{}, electrons <-chan Electron) {
	if validator.IsValid(mizer) {
		go func(ctx context.Context, conductor interface{}, electrons <-chan Electron) {
			defer func() {
				if r := recover(); r != nil {
					mizer.sendLog(fmt.Sprintf("panic in distribute [%v] for conductor [%s]; restarting distributor", r, conductor))

					// Self Heal - Re-initialize mizer distribute method in event of panic
					go mizer.distribute(ctx, conductor, electrons)

				} else { // TODO: Determine if mizer is the correct action if the electron channel is closed
					// clean up the cancel method for mizer conductor and close the context
					if cFunc, ok := mizer.conductorCancelFuncs.Load(conductor); ok {

						// Type assert the cancel method and execute it to close the context
						if cancel, ok := cFunc.(context.CancelFunc); ok {

							// Execute the cancel method for mizer conductor
							cancel()
							mizer.sendLog(fmt.Sprintf("context for conductor [%s] successfully cancelled", conductor))
						}
					} else {
						mizer.sendErr(errors.Errorf("unable to find context cancellation function for conductor [%s]", conductor))
					}
				}
			}()

			// Read from the electron channel for mizer conductor and push onto the mizer electron channel for processing
			for {
				select {
				case <-ctx.Done():
					// Break the loop to close out the receiver
					mizer.sendErr(errors.Errorf("context closed for distribution of conductor [%s]; exiting [%s]", conductor, ctx.Err().Error()))
					break
				case electron, ok := <-electrons:
					if ok {

						// pull the conductor from the
						if cond, err := mizer.getConductor(conductor); err == nil {

							// Ensure that the electron being received is valid
							if validator.IsValid(electron) {

								mizer.electrons <- ewrappers{electron, cond}
							} else {
								// TODO:
							}
						} else {
							// TODO: send an error showing that the processing for mizer electron couldn't start because the
							//  conductor was unable to be pulled from the registry
						}
					} else { // Channel is closed, break out of the loop
						mizer.sendErr(errors.Errorf("electron channel for conductor [%s] is closed, exiting read cycle", conductor))
						break
					}
				}
			}
		}(ctx, conductor, electrons)
	} else {
		// TODO:
	}
}

// Combine an electron and an atom and initiate the processing of the atom in it's own go routine
func (mizer *atomizer) bond() {
	if validator.IsValid(mizer) {
		defer func() {
			if r := recover(); r != nil {
				mizer.sendLog("panic in bond; re-initializing bonding")

				// Self Heal - Re-initialize the bondings
				go mizer.bond()
			}
		}()

		var err error

		for {
			select {
			case <-mizer.ctx.Done():
				// Context has closed, break the loop
				mizer.sendErr(errors.Errorf("context has been closed for bonding; exiting [%s]", mizer.ctx.Err().Error()))
				break
			case ewrap, ok := <-mizer.electrons:
				if ok {

					// Execute the processing for mizer electron using the atom that is registered
					go func(ewrap ewrappers) {
						// TODO: setup the handler for panics here, and determine what to do in the event of a failure

						// Create an electron instance object to contain internal state information about specific electrons
						var instance = instance{
							ewrap: ewrap,
						}

						// Initialize the new context object for the processing of the electron/atom
						if ewrap.electron.Timeout() != nil {
							// Create a new context for the electron with a timeout
							instance.ctx, instance.cancel = context.WithTimeout(mizer.ctx, *ewrap.electron.Timeout())
						} else {
							// Create a new context for the electron with a cancellation option
							instance.ctx, instance.cancel = context.WithCancel(mizer.ctx)
						}

						var atom Atom
						if atom, err = mizer.getAtom(ewrap.electron.Atom()); err == nil {
							if validator.IsValid(atom) {
								instance.atom = atom

								// outbound channel, when setting up the channel for reading it needs to be passed as outbound
								var outbound = make(chan Electron)

								// Set the instance outbound channel for reading
								instance.outbound = outbound

								// Push the instance to the instances channel to be monitored by the mizer
								mizer.instances <- instance

								var prop = &properties{}

								// TODO: Log the execution of the process method here
								// TODO: build a properites object for this process here to store the results and errors into as they
								// stream in from the process method
								// Execute the process method of the atom
								var results <-chan []byte
								var errstream <-chan error
								var done <-chan bool
								results, errstream, done = instance.atom.Process(instance.ctx, instance.ewrap.electron, outbound)

								if results != nil && errstream != nil {
									for {
										select {
										case <-mizer.ctx.Done():
											// TODO:
										case result, ok := <-results:
											if ok {
												prop.AddResult(result)
											}
										case err := <-errstream:
											if ok {
												prop.AddError(err)
											}
										case <-done: // When the done channel closes break the read loop for these channels

											// TODO: Trigger the end time for the properites

											// Close any of the monitoring go routines monitoring mizer atom
											instance.cancel()

											// TODO: Ensure mizer is the proper thing to do here?? I think it needs to close mizer out
											//  at the conductor rather than here... unless the conductor overrode the call back
											// Execute the callback for the electron
											// TODO: Execute the callback with the noficiation here?
											// instance.ewrap.electron.Callback(result)

											// Break the loop we're done with this process
											break
										default:
											// TODO:
										}
									}
								}
							} else {
								// TODO:
							}
						} else {
							// TODO:
						}
					}(ewrap)

				} else {
					// Send error indicating that the mizer is being disassembled due to a closed electron channel
					mizer.sendErr(errors.New("electron channel closed; tearing down mizer"))

					// Cancel the mizer context to close down go routines
					mizer.cancel()

					// Break the loop
					break
				}
			}
		}
	} else {
		// TODO:
	}
}

// Load the atom from the registration sync map and retrieve a new instances
func (mizer *atomizer) getAtom(id string) (Atom, error) {
	var err error
	var atom Atom

	// Ensure mizer mizer is valid
	if validator.IsValid(mizer) {

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
		err = errors.New("mizer is invalid")
	}

	return atom, err
}

// Load a conductor from the sync map by key
func (mizer *atomizer) getConductor(key interface{}) (conductor Conductor, err error) {
	if validator.IsValid(mizer) {

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
	} else {
		err = errors.New("invalid atomizer")
	}

	return conductor, err
}

// Pull from a sync map containing cancellation functions for contexts
// if the sync map contains instances of context.CancelFunc then execute
// the cancellation method for that id in the sync map, otherwise error to
// the calling method
func (mizer *atomizer) nuke(cmap *sync.Map, id string) (err error) {
	if validator.IsValid(mizer) {
		if len(id) > 0 {
			if cfunc, ok := cmap.Load(id); ok {
				if cancel, ok := cfunc.(context.CancelFunc); ok {

					// Cancel mizer sub-contextS
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
	} else {
		err = errors.New("invalid atomizer")
	}

	return err
}
