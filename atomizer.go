package atomizer

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/benji-vesterby/atomizer/interfaces"
	"github.com/benji-vesterby/validator"
	"github.com/pkg/errors"
)

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

	// This communicates the different conductors and atoms that are registered
	// into the system while it's alive
	registrations chan interface{}

	errors     chan error
	logs       chan string
	properties chan interfaces.Properties
	ctx        context.Context
	cancel     context.CancelFunc
	atoms      sync.Map

	// Map for storing the context cancellation functions for each source
	conductorCancelFuncs sync.Map
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
func (mizer *atomizer) receive(externalRegistations <-chan interface{}) {

	// Close the registrations channel
	defer close(mizer.registrations)

	// Cancel out the atomizer in the event of a panic at the receiver because
	// this cannot be effectively restarted without creating an 	 state
	defer mizer.cancel()

	// Initialize the registrations channel if it's nil
	if mizer.registrations == nil {
		mizer.registrations = make(chan interface{})
	}

	// Validate the mizer instance
	if validator.IsValid(mizer) {

		for {
			select {
			case <-mizer.ctx.Done():
				break

			// Handle the external-registrations
			case registration := <-externalRegistations:
				if err := mizer.register(registration); err != nil {
					mizer.sendErr(err)
				}

			// Handle the real-time registrations
			case registration, ok := <-mizer.registrations:
				if ok {
					if err := mizer.register(registration); err != nil {
						mizer.sendErr(err)
					}
				} else {
					// channel closed
					panic("unexpected closing of the registrations channel in the atomizer")
				}
			}
		}
	} else {
		panic("invalid atomizer object")
	}
}

// register the differnt receivable interfaces into the atomizer from wherever they were sent from
func (mizer *atomizer) register(registration interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.Errorf("panic in register, unable to register [%v]; [%s]", reflect.TypeOf(registration), r)
		}
	}()

	if validator.IsValid(mizer) {

		if validator.IsValid(registration) {

			// Switch over the registration type
			switch v := registration.(type) {
			case interfaces.Conductor:
				err = mizer.receiveConductor(v)
			case interfaces.Atom:
				if _, exists := mizer.atoms.LoadOrStore(v.ID(), v); exists {
					err = errors.New("atom [%v] already registered with the atomizer")
				}
			}
		} else {
			err = errors.Errorf("invalid [%v] passed to regsiter", reflect.TypeOf(registration))
		}
	} else {
		err = errors.New("invalid atomzier")
	}

	return err
}

// recieveConductor setups a retrieval loop for the conductor being passed in
func (mizer *atomizer) receiveConductor(conductor interfaces.Conductor) (err error) {
	if validator.IsValid(mizer) {

		// Ensure this is a valid conductor
		if ok := validator.IsValid(conductor); ok {

			// Create the source context with a cancellation option and store the cancellation in a sync map
			ctx, ctxFunc := context.WithCancel(mizer.ctx)
			mizer.conductorCancelFuncs.Store(conductor.ID(), ctxFunc)

			// Pull the receive channel from the conductor for new electrons
			var electrons = conductor.Receive()

			// Ensure the channel is valid
			if electrons != nil {

				// Push off the reading into it's own go routine so that it's concurrent
				mizer.distribute(ctx, conductor, electrons)
			} else {
				err = errors.Errorf("invalid electron channel for conductor [%v]", conductor.ID())
			}

		} else {
			err = errors.Errorf("invalid conductor [%v] set to be received", conductor.ID())
		}

	} else {
		err = errors.New("invalid atomizer")
	}

	return err
}

// Reading in from a specific electron channel of a conductor and drop it onto the atomizer channel for electrons
func (mizer *atomizer) distribute(ctx context.Context, conductor interfaces.Conductor, electrons <-chan interfaces.Electron) (err error) {
	if validator.IsValid(mizer) {
		go func(ctx context.Context, conductor interfaces.Conductor, electrons <-chan interfaces.Electron) {
			defer func() {
				if r := recover(); r != nil {
					mizer.sendLog(fmt.Sprintf("panic in distribute [%v] for conductor [%v]; pushing conductor back to atomizer", r, conductor.ID()))

					// Self Heal - Replace the conductor onto the map for the atomizer to re-initialize so this stack can be garbage collected
					err = mizer.Register(conductor)

				} else {
					// clean up the cancel method for mizer conductor and close the context
					if cFunc, ok := mizer.conductorCancelFuncs.Load(conductor.ID()); ok {

						// Type assert the cancel method and execute it to close the context
						if cancel, ok := cFunc.(context.CancelFunc); ok {

							// Execute the cancel method for mizer conductor
							cancel()
							mizer.sendLog(fmt.Sprintf("context for conductor [%v] successfully cancelled", conductor.ID()))
						} else {
							panic(fmt.Sprintf("unable to close context for conductor [%v] in atomizer", conductor.ID()))
						}
					} else {
						panic(fmt.Sprintf("unable to find context cancellation function for conductor [%v]", conductor.ID()))
					}
				}
			}()

			// Read from the electron channel for mizer conductor and push onto the mizer electron channel for processing
			for {
				select {
				case <-ctx.Done():
					// Break the loop to close out the receiver
					mizer.sendErr(errors.Errorf("context closed for distribution of conductor [%v]; exiting [%s]", conductor.ID(), ctx.Err().Error()))
					break
				case electron, ok := <-electrons:
					if ok {

						// Ensure that the electron being received is valid
						if validator.IsValid(electron) {

							// Send the electron down the electrons channel to be processed
							mizer.electrons <- ewrappers{electron, conductor}
						} else {
							mizer.sendErr(errors.Errorf("invalid electron passed to atomizer [%v]", electron))
						}
					} else { // Channel is closed, break out of the loop
						mizer.sendErr(errors.Errorf("electron channel for conductor [%v] is closed, exiting read cycle", conductor.ID()))
						break
					}
				}
			}
		}(ctx, conductor, electrons)
	} else {
		err = errors.New("mizer is invalid")
	}

	return err
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
						// Initizlize properties for this electron for tracking information
						var prop = &properties{
							status: PENDING,
						}

						// panic handler here in the event of the bonded atom and electron failing processing so that it doesn't
						// take down the whole atomizer and the sender knows about the issues by sending back through the conductor
						defer func() {
							if r := recover(); r != nil {
								// Complete the electron at the conductor level with the property information
								ewrap.conductor.Complete(prop)

								// Log the error
								mizer.sendErr(errors.Errorf("panic in execution of electron [%s]", ewrap.electron.ID()))
							}
						}()

						// TODO: Register this instance with the sampler so that it's monitored at runtime for heartbeats, etc...
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

						var atom interfaces.Atom
						if atom, err = mizer.getAtom(ewrap.electron.Atom()); err == nil {
							if validator.IsValid(atom) {
								instance.atom = atom

								// outbound channel, when setting up the channel for reading it needs to be passed as outbound
								var outbound = make(chan interfaces.Electron)

								// TODO: Register the outbound channel with the conductor that created this instance
								// Set the instance outbound channel for reading
								instance.outbound = outbound

								// Push the instance to the instances channel to be monitored by the mizer
								mizer.instances <- instance

								// TODO: Log the execution of the process method here
								// TODO: build a properties object for this process here to store the results and errors into as they
								// TODO: Setup with a heartbeat for monitoring processing of the bonded atom
								// stream in from the process method
								// Execute the process method of the atom
								var results <-chan []byte
								var errstream <-chan error
								results, errstream = instance.atom.Process(instance.ctx, instance.ewrap.electron, outbound)

								// Update the status of the bonded atom/electron to show processing
								prop.status = PROCESSING

								// Continue looping while either of the channels is non-nil and open
								for results != nil || errstream != nil {
									select {
									// Monitor the instance context for cancellation
									case <-instance.ctx.Done():
										mizer.sendLog(fmt.Sprintf("cancelling electron instance [%v] due to cancellation [%s]", ewrap.electron.ID(), instance.ctx.Err().Error()))
									case result, ok := <-results:
										// Append the results from the bonded instance for return through the properties
										if ok {
											prop.AddResult(result)
										} else {
											// nil out the result channel after it's closed so that the loop breaks
											result = nil
										}
									case err := <-errstream:
										// Append the errors from the bonded instance for return through the properties
										if ok {
											prop.AddError(err)
										} else {
											// nil out the err channel after it's closed so that the loop breaks
											errstream = nil
										}
									}
								}

								// TODO: The processing has finished for this bonded atom and the results need to be calculated and the properties sent back to the
								// conductor

								// Set the end time and status in the properties
								prop.end = time.Now()
								prop.status = COMPLETED

								// TODO: Ensure mizer is the proper thing to do here?? I think it needs to close mizer out
								//  at the conductor rather than here... unless the conductor overrode the call back

								// TODO: Execute the callback with the noficiation here?

								// TODO: Execute the electron callback here
								instance.ewrap.electron.Callback(prop)
							} else {
								mizer.sendErr(errors.Errorf("invalid atom [%s] loaded from registry", ewrap.electron.Atom()))
							}
						} else {
							mizer.sendErr(errors.Errorf("error occurred while loading atom [%s] from registry", ewrap.electron.Atom()))
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
		// Panic because this is always called as a go routine and will
		// hide errors otherwise so crash the application
		panic("mizer is invalid")
	}
}

// Load the atom from the registration sync map and retrieve a new instances
func (mizer *atomizer) getAtom(id string) (interfaces.Atom, error) {
	var err error
	var atom interfaces.Atom

	// Ensure mizer mizer is valid
	if validator.IsValid(mizer) {

		// Load the atom from the sync.Map
		if a, ok := mizer.atoms.Load(id); ok {

			// Ensure the value that was registered wasn't stored nil
			if validator.IsValid(a) {

				// Initialize a new copy of the atom that was registered
				newAtom := reflect.New(reflect.TypeOf(a))

				// Type assert the new copy of the atom to an atom so that it can be used for processing
				// and returned as a pointer for bonding
				if atom, ok = newAtom.Interface().(interfaces.Atom); ok {
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

// TODO: determine if this is still necessary
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
