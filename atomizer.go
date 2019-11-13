package atomizer

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/benjivesterby/validator"
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

	throttle *sampler

	// This sync.Map contains the channels for handling each of the bondings for the
	// different atoms registered in the system
	atomFanOut    map[string]chan<- instance
	atomFanOutMut sync.RWMutex

	outputMutty sync.Mutex
	events      chan string
	errors      chan error

	properties chan Properties
	ctx        context.Context
	cancel     context.CancelFunc

	execSyncOnce sync.Once
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

		// Initialize throttle sampler type here
		if mizer.throttle == nil {
			mizer.throttle = &sampler{
				ctx:     mizer.ctx,
				process: make(chan bool),
				once:    &sync.Once{},
			}
		}

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
func (mizer *atomizer) error(err error) {
	if validator.IsValid(mizer) {
		if err != nil {
			if mizer.errors != nil {

				select {
				case <-mizer.ctx.Done():
					defer close(mizer.errors)
					return

				case mizer.errors <- err:
				}
			}
		}

	}
}

// If the event channel is not nil then send the event on the channel
func (mizer *atomizer) event(event string) {
	if len(event) > 0 {
		if validator.IsValid(mizer) {
			if mizer.events != nil {

				select {
				case <-mizer.ctx.Done():
					defer close(mizer.events)
					return

				case mizer.events <- event:
					// Sent the error on the channel
				}
			}
		}
	}
}

// Initialize the go routines that will read from the conductors concurrently while other parts of the
// atomizer reads in the inputs and executes the instances of electrons
func (mizer *atomizer) receive(externalRegistations <-chan interface{}) (err error) {
	// Initialize the registrations channel if it's nil
	if mizer.registrations == nil {
		mizer.registrations = make(chan interface{})
	}

	// Validate the mizer instance
	if validator.IsValid(mizer) {

		go func() {
			// TODO: handle panics and re-init
			// TODO: Self-heal with heartbeats

			// Close the registrations channel
			defer close(mizer.registrations)

			// Cancel out the atomizer in the event of a panic at the receiver because
			// this cannot be effectively restarted without creating an inconsistent state
			defer mizer.cancel()

			for {
				select {
				case <-mizer.ctx.Done():
					return

				// Handle the external-registrations
				case registration, ok := <-externalRegistations:
					if ok {
						if err := mizer.register(registration); err != nil {
							mizer.error(err)
						}
					} else {
						// channel closed
						panic("unexpected closing of the externalRegistations channel in the atomizer")
					}

				// Handle the real-time registrations
				case registration, ok := <-mizer.registrations:
					if ok {
						if err := mizer.register(registration); err != nil {
							mizer.error(err)
						}
					} else {
						// channel closed
						panic("unexpected closing of the registrations channel in the atomizer")
					}
				}
			}
		}()
	} else {
		err = errors.New("invalid atomizer object")
	}

	return err
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
				mizer.event(fmt.Sprintf("receiving conductor [%s]", v.ID()))
				err = mizer.receiveConductor(v)
			case Atom:
				mizer.event(fmt.Sprintf("receiving atom [%s]", v.ID()))
				err = mizer.receiveAtom(v)
			default:
				// error here because the type is unknown
				err = errors.Errorf("unknown registration type [%v]", reflect.TypeOf(registration))
			}
		} else {
			err = errors.Errorf("invalid [%v] passed to regsiter", reflect.TypeOf(registration))
		}
	} else {
		err = errors.New("invalid atomizer")
	}

	return err
}

// receiveConductor setups a retrieval loop for the conductor being passed in
func (mizer *atomizer) receiveConductor(conductor Conductor) (err error) {
	if validator.IsValid(mizer) {

		// Ensure this is a valid conductor
		if ok := validator.IsValid(conductor); ok {

			// Create the source context with a cancellation option and store the cancellation in a sync map
			ctx, ctxFunc := context.WithCancel(mizer.ctx)

			// Push off the reading into it's own go routine so that it's concurrent
			go mizer.conduct(ctx, cwrapper{conductor, ctx, ctxFunc})

		} else {
			err = errors.Errorf("invalid conductor [%v] set to be received", conductor.ID())
		}

	} else {
		err = errors.New("invalid atomizer")
	}

	return err
}

// Reading in from a specific electron channel of a conductor and drop it onto the atomizer channel for electrons
func (mizer *atomizer) conduct(ctx context.Context, conductor Conductor) {
	// defer mizer.error(handle(ctx, conductor, func() {

	// 	// TODO: HANDLE the sampler panic differently here so that it properly crashes
	// 	// the atomizer

	// 	// Self Heal - Re-place the conductor on the register channel for the atomizer
	// 	// to re-initialize so this stack can be garbage collected
	// 	mizer.error(mizer.Register(conductor))
	// }))

	mizer.event(fmt.Sprintf("conductor [%s] initialized", conductor.ID()))
	receiver := conductor.Receive(ctx)

	// Read from the electron channel for mizer conductor and push onto the mizer electron channel for processing
	for {

		// Sampler throttle here
		mizer.throttle.Wait()

		select {
		case <-ctx.Done():
			// Break the loop to close out the receiver
			// TODO: Error here?
			// mizer.sendErr(errors.Errorf("context closed for distribution of conductor [%v]; exiting [%s]", conductor.ID(), ctx.Err().Error()))
			return
		case e, ok := <-receiver:
			if ok {

				mizer.event("marshalling incoming electron")
				// Ensure that the electron being received is valid
				if validator.IsValid(e) {

					mizer.event(fmt.Sprintf("electron [%s] received", e.ID))

					// Send the electron down the electrons channel to be processed
					select {
					case <-mizer.ctx.Done():
						return
					case mizer.electrons <- instance{e, conductor, nil, nil, nil}:
						mizer.event(fmt.Sprintf("electron instance [%s] pushed to distribution", e.ID))
					}
				} else {
					err := errors.Errorf("invalid electron passed to atomizer [%v]", e)
					props := &Properties{
						ElectronID: e.ID,
						AtomID:     e.AtomID,
						Start:      time.Now(),
						End:        time.Now(),
						Error:      err,
						Result:     nil,
					}

					mizer.error(err)
					conductor.Complete(ctx, props)
				}
			} else { // Channel is closed, break out of the loop
				mizer.error(errors.Errorf("electron channel for conductor [%v] is closed, exiting read cycle", conductor.ID()))
				return
			}
		}
	}
}

// receiveAtom setups a retrieval loop for the conductor being passed in
func (mizer *atomizer) receiveAtom(atom Atom) (err error) {
	if validator.IsValid(mizer) {

		// Ensure this is a valid conductor
		if ok := validator.IsValid(atom); ok {

			var electrons chan<- instance
			// setup atom receiver
			if electrons, err = mizer.split(mizer.ctx, atom); err == nil {
				mizer.event(fmt.Sprintf("splitting atom [%s]", atom.ID()))

				if electrons != nil {

					// Register the atom into the atomizer for receiving electrons
					mizer.atomFanOutMut.Lock()
					mizer.event(fmt.Sprintf("registering electron channel for atom [%s]", atom.ID()))
					mizer.atomFanOut[atom.ID()] = electrons
					mizer.atomFanOutMut.Unlock()
				} else {
					err = errors.Errorf("atom [%s] returned nil electron channel post split", atom.ID())
				}
			} else {
				err = errors.Errorf("error splitting atom [%v] | %s", atom.ID(), err.Error())
			}
		} else {
			err = errors.Errorf("invalid atom [%v] to be split", atom.ID())
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
			defer mizer.error(handle(ctx, atom, func() {

				// remove the electron channel from the map of atoms so that it can be
				// properly cleaned up before re-registering
				mizer.atomFanOutMut.Lock()
				delete(mizer.atomFanOut, atom.ID())
				mizer.atomFanOutMut.Unlock()

				// Self Heal - Re-place the atom on the register channel for the atomizer
				// to re-initialize so this stack can be garbage collected
				mizer.error(mizer.Register(atom))
			}))

			// Read from the electron channel for mizer conductor and push onto the mizer electron channel for processing
			for {
				select {
				case <-ctx.Done():
					// Break the loop to close out the receiver
					// TODO: Error here?
					//mizer.sendErr(errors.Errorf("context closed for atom [%v]; exiting [%s]", atom.ID(), ctx.Err().Error()))
					return
				case inst, ok := <-electrons:
					if ok {

						mizer.event(fmt.Sprintf("creating new instance of electron [%s]", inst.electron.ID))

						// TODO: implement the processing push
						// TODO: after the processing has started push to instances channel for monitoring by the
						// sampler so that this second can focus on starting additional instances rather than on
						// individually bonded instances

						// Initialize a new copy of the atom
						newAtom := reflect.New(reflect.TypeOf(atom).Elem())

						mizer.event(fmt.Sprintf("new instance of electron [%s] created", inst.electron.ID))

						go mizer.exec(ctx, inst, newAtom)
					} else { // Channel is closed, break out of the loop
						mizer.error(errors.Errorf("electron channel for conductor [%v] is closed, exiting read cycle", atom.ID()))
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

func (mizer *atomizer) exec(ctx context.Context, inst instance, newAtom reflect.Value) {
	var err error
	// TODO: Handler here
	// Type assert the new copy of the atom to an atom so that it can be used for processing
	// and returned as a pointer for bonding
	// the := is on purpose here to hide the original instance of the atom so that it's not
	// being accidentally used in this section
	if a, ok := newAtom.Interface().(Atom); ok {
		if validator.IsValid(a) {

			mizer.event(fmt.Sprintf("bonding electron [%s]", inst.electron.ID))

			// bond the new atom instantiation to the electron instance
			if err = inst.bond(a); err == nil {

				// TODO: add this back in after the sampler is working
				// Push the instance to the next part of the process
				// select {
				// case <-ctx.Done():
				// 	return
				// case mizer.bonded <- inst:

				// 	// Execute the instance after it's been picked up for monitoring
				// 	inst.execute(ctx)
				// }

				// Execute the instance after it's been picked up for monitoring
				inst.execute(ctx)
			} else {
				mizer.error(errors.Errorf("error while bonding atom [%s]: [%s]", a.ID(), err.Error()))
			}
		} else {
			mizer.error(errors.Errorf("invalid atom [%s]", a.ID()))
		}
	} else {
		mizer.error(errors.Errorf("unable to type assert atom [%s] for electron id [%s]", inst.electron.AtomID, inst.electron.ID))
	}
}

func (mizer *atomizer) distribute() {
	// TODO: defer
	defer close(mizer.electrons)

	mizer.event("setting up atom distribution channels")

	if validator.IsValid(mizer) {
		for {
			select {
			case <-mizer.ctx.Done():
				return
			case ewrap, ok := <-mizer.electrons:
				if ok {

					mizer.event(fmt.Sprintf("electron [%s] set for distribution to atom [%s]", ewrap.electron.ID, ewrap.electron.AtomID))
					// TODO: how would this call be tracked going forward as part of a go routine? What if this blocks forever?
					// TODO: Handle the panic here in the event that the channel is closed and return the electron to the channel

					if validator.IsValid(ewrap) {
						var achan chan<- instance

						mizer.atomFanOutMut.RLock()
						achan = mizer.atomFanOut[ewrap.electron.AtomID]
						mizer.atomFanOutMut.RUnlock()

						// Pass the electron to the correct atom channel if it is not nil
						if achan != nil {
							select {
							case <-mizer.ctx.Done():
								return
							case achan <- ewrap:
							}
						} else {
							mizer.error(errors.Errorf("atom [%s] channel for receiving electrons is nil", ewrap.electron.AtomID))
						}
					} else {
						mizer.error(errors.Errorf("invalid electron wrapper for atom [%s]", ewrap.electron.AtomID))
					}
				} else {
					// TODO: panic here because atomizer can't work without electron distribution
					mizer.error(errors.New("electron distribution channel closed"))
					defer mizer.cancel()
					return
				}
			}
		}
	} else {
		mizer.error(errors.New("invalid atomizer instance"))
	}
}
