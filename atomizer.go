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
	electrons	chan Electron
	errors		chan error
	logs		chan string
	ctx			context.Context
	cancel		context.CancelFunc

	// Map for storing the context cancellation functions for each source
	conductorCancelFuncs sync.Map

	// Map for storing the instances of specific electrons
	instances sync.Map
}

func Atomize(ctx context.Context) (atomizer *Atomizer, err error) {
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)
	
	// Initialize the atomizer and establish the channels
	atomizer = &Atomizer{
		electrons: make(chan Electron),
		ctx: ctx,
		cancel: cancel,
	}

	// Start up the receivers
	if err = atomizer.receive(); err == nil {

		// Initialize the bonding of electrons and atoms
		go atomizer.bond()
	}

	return atomizer, err
}

// Create a channel to receive errors from the Atomizer and return the channel for logging purposes
func (this *Atomizer) Errors(buffer int) <- chan error {
	if this.errors == nil {

		// Ensure that a proper buffer size was passed for the channel
		if buffer < 0 {
			buffer = 0
		}

		// Only upon request should the error channel be established meaning a user should read from the channel
		this.errors = make(chan error, buffer)
	}

	return this.errors
}

// Create a channel to receive logs from the Atomizer, and electrons
func (this *Atomizer) Logs(buffer int) <- chan string {

	if this.logs == nil {

		// Ensure that a proper buffer size was passed for the channel
		if buffer < 0 {
			buffer = 0
		}

		// Only upon request should the log channel be established meaning a user should read from the channel
		this.logs = make(chan string, buffer)
	}

	return this.logs
}

// If the error channel is not nil then send the error on the channel
func (this *Atomizer) sendErr(err error) {
	if this.errors != nil {
		go func(err error) {
			this.errors <- err
		}(err)
	}
}

// if the log channel is not nil then send the log on the channel
func (this *Atomizer) sendLog(log string) {
	if this.logs != nil {
		go func(log string) {
			this.logs <- log
		}(log)
	}
}

// Initialize the go routines that will read from the conductors concurrently while other parts of the
// atomizer reads in the inputs and executes the instances of electrons
func (this *Atomizer) receive() (err error) {

	// Validate inputs
	if this.ctx != nil {

		// Validate the atomizer instance
		if validator.IsValid(this) {

			// Range over the sources and setup
			conductors.Range(func(key, value interface{}) bool {
				var ok bool

				select {
				case <- this.ctx.Done(): // Do nothing so that the loop breaks immediately
				default:

					// Create the source context with a cancellation option and store the cancellation in a sync map
					ctx, ctxFunc := context.WithCancel(this.ctx)
					this.conductorCancelFuncs.Store(key, ctxFunc)

					var conductor Conductor
					if conductor, ok = value.(Conductor); ok {
						var electrons = conductor.Receive()

						if electrons != nil {

							// Push off the reading into it's own go routine so that it's concurrent
							go this.distribute(ctx, key, electrons)
							ok = true
						} else {
							err = errors.Errorf("invalid electron channel for conductor [%s]", key)
						}
					}
				}

				return ok
			})
		} else {
			err = errors.Errorf("Invalid Atomizer [%v]", this)
		}
	}

	return err
}

// Reading in from a specific electron channel of a conductor and drop it onto the atomizer channel for electrons
func (this *Atomizer) distribute(ctx context.Context, conductor interface{}, electrons <- chan Electron) {
	defer func() {
		if r := recover(); r != nil {
			this.sendLog(fmt.Sprintf("panic in distribute [%v] for conductor [%s]; restarting distributor", r, conductor))

			// Self Heal - Re-initialize this distribute method in event of panic
			go this.distribute(ctx, conductor, electrons)

		} else { // TODO: Determine if this is the correct action if the electron channel is closed
			// clean up the cancel method for this conductor and close the context
			if cFunc, ok := this.conductorCancelFuncs.Load(conductor); ok {

				// Type assert the cancel method and execute it to close the context
				if cancel, ok := cFunc.(context.CancelFunc); ok {

					// Execute the cancel method for this conductor
					cancel()
					this.sendLog(fmt.Sprintf("context for conductor [%s] successfully cancelled", conductor))
				}
			} else {
				this.sendErr(errors.Errorf("unable to find context cancellation function for conductor [%s]", conductor))
			}
		}
	}()

	// Read from the electron channel for this conductor and push onto the atomizer electron channel for processing
	for {
		select {
		case <- ctx.Done():
			// Break the loop to close out the receiver
			this.sendErr(errors.Errorf("context closed for distribution of conductor [%s]; exiting [%s]", ctx.Err().Error()))
			break
		case electron, ok := <- electrons:
			if ok {
				this.electrons <- electron
			} else { // Channel is closed, break out of the loop
				this.sendErr(errors.Errorf("electron channel for conductor [%s] is closed, exiting read cycle", conductor))
				break
			}
		}
	}
}

// Combine an electron and an atom and initiate the processing of the atom in it's own go routine
func (this *Atomizer) bond() {
	defer func() {
		if r := recover(); r != nil {
			this.sendLog("panic in bond; re-initializing bonding")

			// Self Heal - Re-initialize the bondings
			go this.bond()
		}
	}()

	var err error

	for {
		select {
			case <- this.ctx.Done():
				// Context has closed, break the loop
				this.sendErr(errors.Errorf("context has been closed for bonding; exiting [%s]", this.ctx.Err().Error()))
				break
				case electron, ok := <- this.electrons:
					if ok {

						// Execute the processing for this electron using the atom that is registered
						go func(electron Electron) {
							// TODO: setup the handler for panics here, and determine what to do in the event of a failure

							// Create an electron instance object to contain internal state information about specific electrons
							var instance = instance{
								electron: electron,
							}

							// Initialize the new context object for the processing of the electron/atom
							if electron.Timeout() != nil {
								// Create a new context for the electron with a timeout
								instance.ctx, instance.cancel = context.WithTimeout(this.ctx, *electron.Timeout())
							} else {
								// Create a new context for the electron with a cancellation option
								instance.ctx, instance.cancel = context.WithCancel(this.ctx)
							}

							var atom Atom
							if atom, err = this.getAtom(electron.Atom()); err == nil {
								if validator.IsValid(atom) {
									instance.atom = atom

									// Store the instance of the bonded electron into the sync.Map
									if _, exists := this.instances.LoadOrStore(electron.Id(), instance); !exists {

										// outbound channel, when setting up the channel for reading it needs to be passed as outbound
										var outbound = make(chan Electron)

										// Set the instance outbound channel for reading
										instance.outbound = outbound

										// TODO: Log the execution of the process method here
										// Execute the process method of the atom
										err = instance.atom.Process(instance.ctx, instance.electron, outbound)
									} else {
										// TODO: This electron for id already exists. This needs to error back and it's already being processed
									}
								} else {
									// TODO:
								}
							} else {
								// TODO:
							}
						}(electron)

					} else {
						// Send error indicating that the atomizer is being disassembled due to a closed electron channel
						this.sendErr(errors.New("electron channel closed; tearing down atomizer"))

						// Cancel the atomizer context to close down go routines
						this.cancel()

						// Break the loop
						break
					}
		}
	}
}

// Load the atom from the registration sync map and retrieve a new instances
func (this *Atomizer) getAtom(id string) (Atom, error) {
	var err	error
	var atom Atom

	// Ensure this atomizer is valid
	if validator.IsValid(this) {

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

func (this *Atomizer) Exit()  {
	// Make a clean exit for atomizer
}

func (this *Atomizer) Validate() (valid bool) {

	if this.electrons != nil {
		valid = true
	}

	return valid
}

// Pull from a sync map containing cancellation functions for contexts
// if the sync map contains instances of context.CancelFunc then execute
// the cancellation method for that id in the sync map, otherwise error to
// the calling method
func (this *Atomizer) nuke(cmap sync.Map, id string) (err error) {
	if len(id) > 0 {
		if cfunc, ok := cmap.Load(id); ok {
			if cancel, ok := cfunc.(context.CancelFunc); ok {

				// Cancel this sub-contextS
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