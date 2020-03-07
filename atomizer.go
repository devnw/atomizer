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

func (a *atomizer) init() *atomizer {

	// Initialize the context. In the event that the atomizer was initialized with
	// a context passed into it then use the context to create a new context with a cancel
	// otherwise create a new context with cancel from a background context
	if a.ctx == nil {
		a.ctx, a.cancel = context.WithCancel(context.Background())
	} else if a.cancel == nil {
		a.ctx, a.cancel = context.WithCancel(a.ctx)
	}

	select {
	case <-a.ctx.Done():
		return nil
	default:

		// Initialize throttle sampler type here
		if a.throttle == nil {
			a.throttle = &sampler{
				ctx:     a.ctx,
				process: make(chan bool),
				once:    &sync.Once{},
			}
		}

		// Initialize the electrons channel
		if a.electrons == nil {
			a.electrons = make(chan instance)
		}

		// Initialize the bonded channel
		if a.bonded == nil {
			a.bonded = make(chan instance)
		}

		// Initialize the registrations channel
		if a.registrations == nil {
			a.registrations = make(chan interface{})
		}

		// Initialize the atom fan out map and mutex
		if a.atomFanOut == nil {
			a.atomFanOut = make(map[string]chan<- instance)
			a.atomFanOutMut = sync.RWMutex{}
		}
	}

	return a
}

// If the error channel is not nil then send the error on the channel
func (a *atomizer) error(err error) {

	if validator.Valid(a.errors, err) {
		select {
		case <-a.ctx.Done():
			defer close(a.errors)
			return

		case a.errors <- err:
			// Sent the error on the channel
		}
	}
}

// If the event channel is not nil then send the event on the channel
func (a *atomizer) event(event string) {

	if validator.Valid(a.events, event) {
		select {
		case <-a.ctx.Done():
			defer close(a.events)
			return

		case a.events <- event:
			// Sent the event on the channel
		}
	}
}

// Initialize the go routines that will read from the conductors concurrently while other parts of the
// atomizer reads in the inputs and executes the instances of electrons
func (a *atomizer) receive(external <-chan interface{}) (err error) {
	// Initialize the registrations channel if it's nil
	if a.registrations == nil {
		a.registrations = make(chan interface{})
	}

	go func(ext <-chan interface{}, reg chan interface{}) {
		// TODO: handle panics and re-init
		defer func() {
			if r := recover(); r != nil {
				a.error(aErr{errors.Errorf("%s", r), "panic in atomizer receive"})
			}
		}()

		// Close the registrations channel
		defer close(reg)

		// TODO: Self-heal with heartbeats

		// Cancel out the atomizer in the event of a panic at the receiver because
		// this cannot be effectively restarted without creating an inconsistent state
		defer a.cancel()

		for {
			select {
			case <-a.ctx.Done():
				return

			// Handle the external-registrations
			case registration, ok := <-ext:
				if ok {
					if err := a.register(registration); err != nil {
						a.error(err)
					}
				} else {
					// channel closed
					panic("unexpected closing of the external registrations channel in the atomizer")
				}

			// Handle the real-time registrations
			case registration, ok := <-reg:
				if ok {
					if err := a.register(registration); err != nil {
						a.error(err)
					}
				} else {
					// channel closed
					panic("unexpected closing of the registrations channel in the atomizer")
				}
			}
		}
	}(external, a.registrations)

	return err
}

// register the different receivable interfaces into the atomizer from wherever they were sent from
func (a *atomizer) register(registration interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.Errorf("panic in register, unable to register [%v]; [%s]", reflect.TypeOf(registration), r)
		}
	}()

	if validator.IsValid(a) {

		if validator.IsValid(registration) {

			// Switch over the registration type
			switch v := registration.(type) {
			case Conductor:
				a.event(fmt.Sprintf("receiving conductor [%s]", ID(v)))
				err = a.receiveConductor(v)
			case Atom:
				a.event(fmt.Sprintf("receiving atom [%s]", ID(v)))
				err = a.receiveAtom(v)
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
func (a *atomizer) receiveConductor(conductor Conductor) (err error) {
	if validator.IsValid(a) {

		// Ensure this is a valid conductor
		if ok := validator.IsValid(conductor); ok {

			// Create the source context with a cancellation option and store the cancellation in a sync map
			// ctx, ctxFunc := context.WithCancel(a.ctx)
			// TODO: use the conductor wrapper: cwrapper{conductor, ctx, ctxFunc} so that the conductor
			// can be cancelled if necessary

			// Push off the reading into it's own go routine so that it's concurrent
			go a.conduct(a.ctx, conductor)

		} else {
			err = errors.Errorf("invalid conductor [%s] set to be received", ID(conductor))
		}

	} else {
		err = errors.New("invalid atomizer")
	}

	return err
}

// Reading in from a specific electron channel of a conductor and drop it onto the atomizer channel for electrons
func (a *atomizer) conduct(ctx context.Context, conductor Conductor) {
	// defer a.error(handle(ctx, conductor, func() {

	// 	// TODO: HANDLE the sampler panic differently here so that it properly crashes
	// 	// the atomizer

	// 	// Self Heal - Re-place the conductor on the register channel for the atomizer
	// 	// to re-initialize so this stack can be garbage collected
	// 	a.error(a.Register(conductor))
	// }))

	a.event(fmt.Sprintf("conductor [%s] initialized", ID(conductor)))
	receiver := conductor.Receive(ctx)

	// Read from the electron channel for a conductor and push onto the a electron channel for processing
	for {

		// Sampler throttle here
		a.throttle.Wait()

		select {
		case <-ctx.Done():
			// Break the loop to close out the receiver
			// TODO: Error here?
			// a.sendErr(errors.Errorf("context closed for distribution of conductor [%v]; exiting [%s]", conductor.ID(), ctx.Err().Error()))
			return
		case e, ok := <-receiver:
			if ok {

				a.event("marshalling incoming electron")
				// Ensure that the electron being received is valid
				if validator.IsValid(e) {

					a.event(fmt.Sprintf("electron [%s] received", e.ID))

					// Send the electron down the electrons channel to be processed
					select {
					case <-a.ctx.Done():
						return
					case a.electrons <- instance{e, conductor, nil, nil, nil}:
						a.event(fmt.Sprintf("electron instance [%s] pushed to distribution", e.ID))
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

					a.error(err)
					_ = conductor.Complete(ctx, props)
				}
			} else { // Channel is closed, break out of the loop
				a.error(errors.Errorf("electron channel for conductor [%s] is closed, exiting read cycle", ID(conductor)))
				return
			}
		}
	}
}

// receiveAtom setups a retrieval loop for the conductor being passed in
func (a *atomizer) receiveAtom(atom Atom) (err error) {
	if validator.IsValid(a) {

		// Ensure this is a valid conductor
		if ok := validator.IsValid(atom); ok {

			var electrons chan<- instance
			// setup atom receiver
			if electrons, err = a.split(a.ctx, atom); err == nil {
				a.event(fmt.Sprintf("splitting atom [%s]", ID(atom)))

				if electrons != nil {

					// Register the atom into the atomizer for receiving electrons
					a.atomFanOutMut.Lock()
					a.event(fmt.Sprintf("registering electron channel for atom [%s]", ID(atom)))
					a.atomFanOut[ID(atom)] = electrons
					a.atomFanOutMut.Unlock()
				} else {
					err = errors.Errorf("atom [%s] returned nil electron channel post split", ID(atom))
				}
			} else {
				err = errors.Errorf("error splitting atom [%s] | %s", ID(atom), err.Error())
			}
		} else {
			err = errors.Errorf("invalid atom [%s] to be split", ID(atom))
		}
	} else {
		err = errors.New("invalid atomizer")
	}

	return err
}

func (a *atomizer) split(ctx context.Context, atom Atom) (chan<- instance, error) {
	electrons := make(chan instance)
	var err error

	if validator.IsValid(a) {

		go func(ctx context.Context, atom Atom, electrons <-chan instance) {
			defer a.error(handle(ctx, atom, func() {

				// remove the electron channel from the map of atoms so that it can be
				// properly cleaned up before re-registering
				a.atomFanOutMut.Lock()
				delete(a.atomFanOut, ID(atom))
				a.atomFanOutMut.Unlock()

				// Self Heal - Re-place the atom on the register channel for the atomizer
				// to re-initialize so this stack can be garbage collected
				a.error(a.Register(atom))
			}))

			// Read from the electron channel for a conductor and push onto the a electron channel for processing
			for {
				select {
				case <-ctx.Done():
					// Break the loop to close out the receiver
					// TODO: Error here?
					//a.sendErr(errors.Errorf("context closed for atom [%v]; exiting [%s]", atom.ID(), ctx.Err().Error()))
					return
				case inst, ok := <-electrons:
					if ok {

						a.event(fmt.Sprintf("creating new instance of electron [%s]", inst.electron.ID))

						// TODO: implement the processing push
						// TODO: after the processing has started push to instances channel for monitoring by the
						// sampler so that this second can focus on starting additional instances rather than on
						// individually bonded instances

						// Initialize a new copy of the atom
						newAtom := reflect.New(reflect.TypeOf(atom).Elem())

						a.event(fmt.Sprintf("new instance of electron [%s] created", inst.electron.ID))

						go a.exec(ctx, inst, newAtom)
					} else { // Channel is closed, break out of the loop
						a.error(errors.Errorf("electron channel for conductor [%s] is closed, exiting read cycle", ID(atom)))
						return
					}
				}
			}
		}(ctx, atom, electrons)
	} else {
		err = errors.New("a is invalid")
	}

	return electrons, err
}

func (a *atomizer) exec(ctx context.Context, inst instance, newAtom reflect.Value) {
	var err error
	// TODO: Handler here
	// Type assert the new copy of the atom to an atom so that it can be used for processing
	// and returned as a pointer for bonding
	// the := is on purpose here to hide the original instance of the atom so that it's not
	// being accidentally used in this section
	if atom, ok := newAtom.Interface().(Atom); ok {
		if validator.IsValid(a) {

			a.event(fmt.Sprintf("bonding electron [%s]", inst.electron.ID))

			// bond the new atom instantiation to the electron instance
			if err = inst.bond(atom); err == nil {

				// TODO: add this back in after the sampler is working
				// Push the instance to the next part of the process
				// select {
				// case <-ctx.Done():
				// 	return
				// case a.bonded <- inst:

				// 	// Execute the instance after it's been picked up for monitoring
				// 	inst.execute(ctx)
				// }

				// Execute the instance after it's been picked up for monitoring
				inst.execute(ctx)
			} else {
				a.error(errors.Errorf("error while bonding atom [%s]: [%s]", ID(atom), err.Error()))
			}
		} else {
			a.error(errors.Errorf("invalid atom [%s]", ID(atom)))
		}
	} else {
		a.error(errors.Errorf("unable to type assert atom [%s] for electron id [%s]", inst.electron.AtomID, inst.electron.ID))
	}
}

func (a *atomizer) distribute() {
	// TODO: defer
	defer close(a.electrons)

	a.event("setting up atom distribution channels")

	if validator.IsValid(a) {
		for {
			select {
			case <-a.ctx.Done():
				return
			case ewrap, ok := <-a.electrons:
				if ok {

					a.event(fmt.Sprintf("electron [%s] set for distribution to atom [%s]", ewrap.electron.ID, ewrap.electron.AtomID))
					// TODO: how would this call be tracked going forward as part of a go routine? What if this blocks forever?
					// TODO: Handle the panic here in the event that the channel is closed and return the electron to the channel

					if validator.IsValid(ewrap) {
						var achan chan<- instance

						a.atomFanOutMut.RLock()
						achan = a.atomFanOut[ewrap.electron.AtomID]
						a.atomFanOutMut.RUnlock()

						// Pass the electron to the correct atom channel if it is not nil
						if achan != nil {
							select {
							case <-a.ctx.Done():
								return
							case achan <- ewrap:
							}
						} else {
							a.error(errors.Errorf("atom [%s] channel for receiving electrons is nil", ewrap.electron.AtomID))
						}
					} else {
						a.error(errors.Errorf("invalid electron wrapper for atom [%s]", ewrap.electron.AtomID))
					}
				} else {
					// TODO: panic here because atomizer can't work without electron distribution
					a.error(errors.New("electron distribution channel closed"))
					defer a.cancel()
					return
				}
			}
		}
	} else {
		a.error(errors.New("invalid atomizer instance"))
	}
}
