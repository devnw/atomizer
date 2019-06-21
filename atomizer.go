package atomizer

import (
	"context"
	"reflect"
	"sync"

	"github.com/benji-vesterby/validator"
	"github.com/pkg/errors"
)

// atomizer facilitates the execution of tasks (aka Electrons) which are received from the configured sources
// these electrons can be distributed across many instances of the atomizer on different nodes in a distributed
// system or in memory. Atoms should be created to process "atomic" actions which are small in scope and overall
// processing complexity minimizing time to run and allowing for the distributed system to take on the burden of
// long running processes as a whole rather than a single process handling the overall load
type atomizer struct {

	// Electron Channel
	electrons chan instance

	// channel for passing the instance to a monitoring go routine
	bonded chan instance

	// This communicates the different conductors and atoms that are registered
	// into the system while it's alive
	registrations chan interface{}

	// This is the communication channel for the atoms being read into the system
	// and is used to create atom workers for bonding purposes
	atoms chan Atom

	// This sync.Map contains the channels for handling each of the bondings for the
	// different atoms registered in the system
	atomFanOut    map[string]chan<- instance
	atomFanOutMut sync.RWMutex

	errors     chan error
	properties chan Properties
	ctx        context.Context
	cancel     context.CancelFunc
}

func (mizer *atomizer) init() *atomizer {

	// Initialize the context. In the event that the atomizer was initialized with
	// a context passed into it then use the context to create a new context with a cancel
	// otherwise create a new context with cancel from a background context
	if mizer.ctx == nil {
		mizer.ctx, mizer.cancel = context.WithCancel(context.Background())
	} else if mizer.cancel == nil {
		mizer.ctx, mizer.cancel = context.WithCancel(mizer.ctx)
	}

	select {
	case <-mizer.ctx.Done():
		return nil
	default:

		// Initialize the electrons channel
		if mizer.electrons == nil {
			mizer.electrons = make(chan instance)
		}

		// Initialize the bonded channel
		if mizer.bonded == nil {
			mizer.bonded = make(chan instance)
		}

		// Initialize the registrations channel
		if mizer.registrations == nil {
			mizer.registrations = make(chan interface{})
		}

		// Initialize the atom fan out map and mutex
		if mizer.atomFanOut == nil {
			mizer.atomFanOut = make(map[string]chan<- instance)
			mizer.atomFanOutMut = sync.RWMutex{}
		}
	}

	return mizer
}

// If the error channel is not nil then send the error on the channel
func (mizer *atomizer) sendErr(err error) {
	if validator.IsValid(mizer) {
		if mizer.errors != nil {
			select {
			case <-mizer.ctx.Done():

				// Close the errors channel and return the context error to the requester
				close(mizer.errors)
				err = mizer.ctx.Err()

			case mizer.errors <- err:
				// Sent the error on the channel
			}
		}
	}
}

// Initialize the go routines that will read from the conductors concurrently while other parts of the
// atomizer reads in the inputs and executes the instances of electrons
func (mizer *atomizer) receive(externalRegistations <-chan interface{}) {
	// TODO: handle panics and re-init
	// TODO: Self-heal with heartbeats

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
				return

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

// register the different receivable interfaces into the atomizer from wherever they were sent from
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
			case Conductor:
				err = mizer.receiveConductor(v)
			case Atom:
				err = mizer.receiveAtom(v)
			default:
				// TODO: error here because the type is unknown
			}
		} else {
			err = errors.Errorf("invalid [%v] passed to regsiter", reflect.TypeOf(registration))
		}
	} else {
		err = errors.New("invalid atomizer")
	}

	return err
}

// recieveConductor setups a retrieval loop for the conductor being passed in
func (mizer *atomizer) receiveConductor(conductor Conductor) (err error) {
	if validator.IsValid(mizer) {

		// Ensure this is a valid conductor
		if ok := validator.IsValid(conductor); ok {

			// Create the source context with a cancellation option and store the cancellation in a sync map
			ctx, ctxFunc := context.WithCancel(mizer.ctx)

			// Push off the reading into it's own go routine so that it's concurrent
			mizer.conduct(ctx, cwrapper{conductor, ctx, ctxFunc})

		} else {
			err = errors.Errorf("invalid conductor [%v] set to be received", conductor.ID())
		}

	} else {
		err = errors.New("invalid atomizer")
	}

	return err
}

// Reading in from a specific electron channel of a conductor and drop it onto the atomizer channel for electrons
func (mizer *atomizer) conduct(ctx context.Context, conductor Conductor) (err error) {
	if validator.IsValid(mizer) {
		go func(ctx context.Context, conductor Conductor) {
			defer mizer.sendErr(handle(conductor, func() {

				// Self Heal - Re-place the conductor on the register channel for the atomizer
				// to re-initialize so this stack can be garbage collected
				mizer.sendErr(mizer.Register(conductor))
			}))

			// Read from the electron channel for mizer conductor and push onto the mizer electron channel for processing
			for {
				select {
				case <-ctx.Done():
					// Break the loop to close out the receiver
					mizer.sendErr(errors.Errorf("context closed for distribution of conductor [%v]; exiting [%s]", conductor.ID(), ctx.Err().Error()))
					return
				case electron, ok := <-conductor.Receive():
					if ok {

						// Ensure that the electron being received is valid
						if validator.IsValid(electron) {

							// Send the electron down the electrons channel to be processed
							mizer.electrons <- instance{electron, conductor, nil, nil, nil}
						} else {
							mizer.sendErr(errors.Errorf("invalid electron passed to atomizer [%v]", electron))
						}
					} else { // Channel is closed, break out of the loop
						mizer.sendErr(errors.Errorf("electron channel for conductor [%v] is closed, exiting read cycle", conductor.ID()))
						return
					}
				}
			}
		}(ctx, conductor)
	} else {
		err = errors.New("mizer is invalid")
	}

	return err
}

// recieveConductor setups a retrieval loop for the conductor being passed in
func (mizer *atomizer) receiveAtom(atom Atom) (err error) {
	if validator.IsValid(mizer) {

		// Ensure this is a valid conductor
		if ok := validator.IsValid(atom); ok {

			var electrons chan<- instance
			// setup atom receiver
			if electrons, err = mizer.split(mizer.ctx, atom); err == nil {

				if electrons != nil {

					// Register the atom into the atomizer for receiving electrons
					mizer.atomFanOutMut.Lock()
					mizer.atomFanOut[atom.ID()] = electrons
					mizer.atomFanOutMut.Unlock()
				} else {
					// TODO: Error
				}
			} else {
				// TODO:
			}
		} else {
			err = errors.Errorf("invalid conductor [%v] set to be received", atom.ID())
		}

	} else {
		err = errors.New("invalid atomizer")
	}

	return err
}

func (mizer *atomizer) split(ctx context.Context, atom Atom) (chan<- instance, error) {
	electrons := make(chan instance)
	var err error

	if validator.IsValid(mizer) {

		go func(ctx context.Context, atom Atom, electrons <-chan instance) {
			defer mizer.sendErr(handle(atom, func() {

				// remove the electron channel from the map of atoms so that it can be
				// properly cleaned up before re-registering
				mizer.atomFanOutMut.Lock()
				delete(mizer.atomFanOut, atom.ID())
				mizer.atomFanOutMut.Unlock()

				// Self Heal - Re-place the atom on the register channel for the atomizer
				// to re-initialize so this stack can be garbage collected
				mizer.sendErr(mizer.Register(atom))
			}))

			// Read from the electron channel for mizer conductor and push onto the mizer electron channel for processing
			for {
				select {
				case <-ctx.Done():
					// Break the loop to close out the receiver
					mizer.sendErr(errors.Errorf("context closed for atom [%v]; exiting [%s]", atom.ID(), ctx.Err().Error()))
					return
				case inst, ok := <-electrons:
					if ok {

						// TODO: implement the processing push
						// TODO: after the processing has started push to instances channel for monitoring by the
						// sampler so that this second can focus on starting additional instances rather than on
						// individually bonded instances

						// Initialize a new copy of the atom
						newAtom := reflect.New(reflect.TypeOf(atom).Elem())

						// Type assert the new copy of the atom to an atom so that it can be used for processing
						// and returned as a pointer for bonding
						// the := is on purpose here to hide the original instance of the atom so that it's not
						// being accidentally used in this section
						if atom, ok := newAtom.Interface().(Atom); ok {
							if validator.IsValid(atom) {

								// bond the new atom instantiation to the electron instance
								if err = inst.bond(atom); err == nil {

									// Push the instance to the next part of the process
									mizer.bonded <- inst
								} else {
									// TODO:
								}
							} else {
								// TODO: Error here
							}
						} else {
							err = errors.Errorf("unable to type assert atom for id [%s]", inst.electron.Atom())
						}
					} else { // Channel is closed, break out of the loop
						mizer.sendErr(errors.Errorf("electron channel for conductor [%v] is closed, exiting read cycle", atom.ID()))
						return
					}
				}
			}
		}(ctx, atom, electrons)
	} else {
		err = errors.New("mizer is invalid")
	}

	return electrons, err
}

func (mizer *atomizer) distribute(ctx context.Context) {
	// TODO: defer

	// Only re-create the channel in the event that it's nil
	if mizer.electrons == nil {
		mizer.electrons = make(chan instance)
	}

	if validator.IsValid(mizer) {
		for {
			select {
			case <-ctx.Done():
				return
			case ewrap, ok := <-mizer.electrons:
				if ok {

					// TODO: how would this call be tracked going forward as part of a go routine? What if this blocks forever?
					go func() {
						// TODO: Handle the panic here in the event that the channel is closed and return the electron to the channel

						if validator.IsValid(ewrap) {
							var achan chan<- instance

							mizer.atomFanOutMut.RLock()
							achan = mizer.atomFanOut[ewrap.electron.Atom()]
							mizer.atomFanOutMut.RUnlock()

							// Pass the electron to the correct atom channel if it is not nil
							if achan != nil {
								achan <- ewrap
							} else {
								// TODO:
							}
						} else {
							// TODO:
						}
					}()
				} else {
					// TODO: panic here because atomizer can't work without electron distribution
					return
				}
			}
		}
	} else {
		// TODO:
	}
}
