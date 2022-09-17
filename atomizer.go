// Copyright © 2019 Developer Network, LLC
//
// This file is subject to the terms and conditions defined in
// file 'LICENSE', which is part of this source code package.

package engine

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/mohae/deepcopy"
	"go.devnw.com/event"
	"go.devnw.com/validator"
)

// Atomize initialize instance of the atomizer to start reading from
// conductors and execute bonded electrons/atoms
//
// NOTE: Registrations can be added through this method and OVERRIDE any
// existing registrations of the same Atom or Conductor.
func Atomize(
	ctx context.Context,
	registrations ...any,
) (*Atomizer, error) {
	err := Register(registrations...)
	if err != nil {
		return nil, err
	}

	ctx, cancel := _ctx(ctx)

	return &Atomizer{
		ctx:      ctx,
		cancel:   cancel,
		requests: make(chan instance),
		bonded:   make(chan instance),
		reg:      make(chan any),
		atoms:    make(map[string]chan<- instance),
		pub:      event.NewPublisher(ctx),
	}, nil
}

// atomizer facilitates the execution of tasks (aka Electrons) which
// are received from the configured sources these electrons can be
// distributed across many instances of the atomizer on different nodes
// in a distributed system or in memory. Atoms should be created to
// process "atomic" actions which are small in scope and overall processing
// complexity minimizing time to run and allowing for the distributed
// system to take on the burden of long running processes as a whole
// rather than a single process handling the overall load
type Atomizer struct {

	// Requests Channel
	requests chan instance

	// channel for passing the instance to a monitoring go routine
	bonded chan instance

	// This communicates the different conductors and atoms that are
	// registered into the system while it's alive
	reg chan any

	// This sync.Map contains the channels for handling each of the
	// bondings for the different atoms registered in the system
	atomsMu sync.RWMutex
	atoms   map[string]chan<- instance

	pub *event.Publisher

	ctx    context.Context
	cancel context.CancelFunc

	execOnce sync.Once
}

// Events returns an go.devnw.com/event.EventStream for the atomizer
func (a *Atomizer) Events(buffer int) event.EventStream {
	return a.pub.ReadEvents(buffer)
}

// Errors returns an go.devnw.com/event.ErrorStream for the atomizer
func (a *Atomizer) Errors(buffer int) event.ErrorStream {
	return a.pub.ReadErrors(buffer)
}

// Exec kicks off the processing of the atomizer by pulling in the
// pre-registrations through init calls on imported libraries and
// starts up the receivers for atoms and conductors
func (a *Atomizer) Exec() (err error) {
	// Execute on the atomizer should only ever be run once
	a.execOnce.Do(func() {
		defer a.pub.EventFunc(a.ctx, func() event.Event {
			return &Event{Msg: "pulling conductor and atom registrations"}
		})

		// Initialize the registrations in the Atomizer package
		for _, r := range Registrations() {
			a.register(r)
		}

		// Start up the receivers
		go a.receive()

		// Setup the distribution loop for incoming electrons
		// so that they can be properly fanned out to the
		// atom receivers
		go a.distribute()

		// TODO: Setup the instance receivers for monitoring of
		// individual instances as well as sending of outbound
		// electrons
	})

	return err
}

// Register allows you to add additional type registrations to the atomizer
// (ie. Conductors and Atoms)
func (a *Atomizer) Register(values ...any) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = &Error{
				Msg:   "panic in atomizer",
				Inner: fmt.Errorf("%v", r),
			}
		}
	}()

	for _, value := range values {
		if !validator.Valid(value) {
			// TODO: create event here indicating that
			// a value was invalid and not registered
			continue
		}

		switch v := value.(type) {
		case Conductor, Processor:
			// Pass the value on the registrations
			// channel to be received
			select {
			case <-a.ctx.Done():
				return &Error{Msg: "context closed"}
			case a.reg <- v:
			}
		default:
			return &Error{
				Msg: "invalid value in registration",
				Meta: Metadata{
					"value": value,
				},
			}
		}
	}

	return err
}

// Wait blocks on the context done channel to allow for the executable
// to block for the atomizer to finish processing
func (a *Atomizer) Wait() {
	<-a.ctx.Done()
}

// Initialize the go routines that will read from the conductors concurrently
// while other parts of the atomizer reads in the inputs and executes the
// instances of electrons
func (a *Atomizer) receive() {
	if a.reg == nil {
		a.pub.ErrorFunc(a.ctx, func() error {
			return &Error{Msg: "nil registrations channel"}
		})
		return
	}

	// TODO: Self-heal with heartbeats
	for {
		select {
		case <-a.ctx.Done():
			return
		case r, ok := <-a.reg:
			if !ok {
				a.pub.ErrorFunc(a.ctx, func() error {
					return &Error{Msg: "registrations closed"}
				})
				return
			}

			// TODO: Only register the conductors
			a.register(r)
			a.pub.EventFunc(a.ctx, func() event.Event {
				return &Event{
					Msg: "registration entered",
					Meta: Metadata{
						"registration": ID(r),
					},
				}
			})
		}
	}
}

// register the different receivable interfaces into the atomizer from
// wherever they were sent from
func (a *Atomizer) register(input any) {
	if !validator.Valid(input) {
		a.pub.ErrorFunc(a.ctx, func() error {
			return &Error{
				Msg: "invalid registration",
				Meta: Metadata{
					"registration": ID(input),
				},
			}
		})
	}

	switch v := input.(type) {
	case Conductor:
		err := a.receiveConductor(v)
		if err == nil {
			a.pub.EventFunc(a.ctx, func() event.Event {
				return &Event{
					Msg:  "conductor received",
					Meta: Metadata{TRANSPORTID: ID(v)},
				}
			})
		}
	case Processor:

		err := a.receiveAtom(v)
		if err == nil {
			a.pub.EventFunc(a.ctx, func() event.Event {
				return &Event{
					Msg:  "conductor received",
					Meta: Metadata{PROCESSORID: ID(v)},
				}
			})
		}
	default:
		a.pub.ErrorFunc(a.ctx, func() error {
			return &Error{
				Msg: "invalid registration",
				Meta: Metadata{
					"registration": ID(input),
				},
			}
		})
	}
}

// receiveConductor setups a retrieval loop for the conductor
func (a *Atomizer) receiveConductor(conductor Conductor) error {
	if !validator.Valid(conductor) {
		return &Error{
			Msg:  "invalid conductor",
			Meta: Metadata{TRANSPORTID: ID(conductor)},
		}
	}

	go a.conduct(a.ctx, conductor)

	return nil
}

// conduct reads in from a specific electron channel of a conductor and drop
// it onto the atomizer channel for electrons
func (a *Atomizer) conduct(ctx context.Context, conductor Conductor) {
	// Self Heal - Re-place the conductor on the register channel for
	// the atomizer to re-initialize so this stack can be
	// garbage collected

	// 	a.publisher.EventFunc(a.ctx, a.Register(conductor))
	// }))

	receiver := conductor.Receive(ctx)

	// Read from the electron channel for a conductor and push onto
	// the a electron channel for processing
	for {
		select {
		case <-ctx.Done():
			return
		case e, ok := <-receiver:
			if !ok {
				a.pub.ErrorFunc(a.ctx, func() error {
					return &Error{
						Msg: "receiver closed",
						Meta: Metadata{
							TRANSPORTID: ID(conductor),
						},
					}
				})

				return
			}

			if !e.Validate() {
				a.pub.ErrorFunc(a.ctx, func() error {
					return &Error{
						Msg: "invalid electron",
						Meta: Metadata{
							REQUESTID:   e.ID,
							TRANSPORTID: ID(conductor),
						},
						Inner: conductor.Complete(
							ctx,
							&Properties{
								RequestID:   e.ID,
								ProcessorID: e.ProcessorID,
								Start:       time.Now(),
								End:         time.Now(),
								Error:       errors.New("invalid electron"),
								Result:      nil,
							},
						),
					}
				})

				continue
			}

			a.pub.EventFunc(a.ctx, func() event.Event {
				return &Event{
					Msg: "electron received",
					Meta: Metadata{
						REQUESTID:   e.ID,
						PROCESSORID: e.ProcessorID,
						TRANSPORTID: ID(conductor),
					},
				}
			})

			// Send the electron down the electrons
			// channel to be processed
			select {
			case <-a.ctx.Done():
				return
			case a.requests <- instance{
				req:   e,
				trans: conductor,
			}:
				a.pub.EventFunc(a.ctx, func() event.Event {
					return &Event{
						Msg: "electron distributed",
						Meta: Metadata{
							REQUESTID:   e.ID,
							PROCESSORID: e.ProcessorID,
							TRANSPORTID: ID(conductor),
						},
					}
				})
			}
		}
	}
}

// receiveAtom setups a retrieval loop for the conductor being passed in
func (a *Atomizer) receiveAtom(atom Processor) error {
	if !validator.Valid(atom) {
		return &Error{
			Msg:  "invalid atom",
			Meta: Metadata{PROCESSORID: ID(atom)},
		}
	}

	// Register the atom into the atomizer for receiving electrons
	a.atomsMu.Lock()
	defer a.atomsMu.Unlock()

	a.atoms[ID(atom)] = a.split(atom)
	a.pub.EventFunc(a.ctx, func() event.Event {
		return &Event{
			Msg:  "registered electron channel",
			Meta: Metadata{PROCESSORID: ID(atom)},
		}
	})

	return nil
}

func (a *Atomizer) split(atom Processor) chan<- instance {
	electrons := make(chan instance)

	go a._split(atom, electrons)

	return electrons
}

func (a *Atomizer) _split(
	atom Processor,
	electrons <-chan instance,
) {
	// Read from the electron channel for a conductor and push
	// onto the a electron channel for processing
	for {
		select {
		case <-a.ctx.Done():
			return
		case inst, ok := <-electrons:
			if !ok {
				a.pub.ErrorFunc(a.ctx, func() error {
					return &Error{
						Msg:  "atom receiver closed",
						Meta: Metadata{PROCESSORID: ID(atom)},
					}
				})
				return
			}

			a.pub.EventFunc(a.ctx, func() event.Event {
				return &Event{
					Msg: "new instance of electron",
					Meta: Metadata{
						REQUESTID:   inst.req.ID,
						PROCESSORID: inst.req.ProcessorID,
						TRANSPORTID: ID(inst.trans),
					},
				}
			})

			// TODO: implement the processing push
			// TODO: after the processing has started
			// push to instances channel for monitoring
			// by the sampler so that this second can
			// focus on starting additional instances
			// rather than on individually bonded
			// instances

			var outatom Processor
			// Copy the state of the original registration to
			// the new atom
			if inst.req.CopyState {
				outatom, _ = deepcopy.Copy(atom).(Processor)
			} else {
				// Initialize a new copy of the atom
				newAtom := reflect.New(
					reflect.TypeOf(atom).Elem(),
				)

				// ok is not checked here because this should
				// never fail since the originating data item
				// is what created this
				outatom, _ = newAtom.Interface().(Processor)
			}

			a.exec(inst, outatom)
		}
	}
}

func (a *Atomizer) exec(inst instance, atom Processor) {
	// bond the new atom instantiation to the electron instance
	if err := inst.bond(atom); err != nil {
		a.pub.ErrorFunc(a.ctx, func() error {
			return &Error{
				Msg: "error while bonding",
				Meta: Metadata{
					PROCESSORID: ID(atom),
					TRANSPORTID: ID(inst.trans),
				},
				Inner: err,
			}
		})
		return
	}

	// Execute the instance after it's been
	// picked up for monitoring
	err := inst.execute(a.ctx)
	if err != nil {
		err = &Error{
			Msg: "execution error",
			Meta: Metadata{
				PROCESSORID: ID(atom),
				REQUESTID:   ID(inst.req.ID),
				TRANSPORTID: ID(inst.trans),
			},
			Inner: &Error{
				Msg:   err.Error(),
				Inner: inst.prop.Error,
			},
		}

		inst.prop.Error = err
		defer a.pub.ErrorFunc(a.ctx, func() error {
			return err
		})

		if inst.trans != nil {
			completion := inst.trans.Complete(a.ctx, inst.prop)
			a.pub.ErrorFunc(a.ctx, func() error {
				return completion
			})
		}
	}
}

func (a *Atomizer) distribute() {
	for {
		select {
		case <-a.ctx.Done():
			return
		case inst, ok := <-a.requests:
			if !ok {
				a.pub.ErrorFunc(a.ctx, func() error {
					return &Error{Msg: "dist channel closed"}
				})

				return
			}

			var proc Processor
			select {
			case <-a.ctx.Done():
				return
			case proc, ok = <-inst.req.m.Make(a.ctx):
				if !ok {
					a.pub.ErrorFunc(a.ctx, func() error {
						return &Error{
							Msg: "processor make closed",
							Meta: Metadata{
								PROCESSORID: inst.req.ProcessorID,
								REQUESTID:   inst.req.ID,
								TRANSPORTID: ID(inst.trans),
							},
						}
					})
					return
				}
			}

			if !ok {
				// TODO: figure out what to do here
				// since the atom doesn't exist in
				// the registry

				a.pub.ErrorFunc(a.ctx, func() error {
					return &Error{
						Msg: "not registered",
						Meta: Metadata{
							PROCESSORID: inst.req.ProcessorID,
							REQUESTID:   inst.req.ID,
							TRANSPORTID: ID(inst.trans),
						},
					}
				})
				continue
			}

			a.pub.EventFunc(a.ctx, func() event.Event {
				return &Event{
					Msg: "pushing electron to atom",
					Meta: Metadata{
						PROCESSORID: inst.req.ProcessorID,
						REQUESTID:   inst.req.ID,
						TRANSPORTID: ID(inst.trans),
					},
				}
			})

			select {
			case <-a.ctx.Done():
				return
			case achan <- inst:
				a.pub.EventFunc(a.ctx, func() event.Event {
					return &Event{
						Msg: "pushed electron to atom",
						Meta: Metadata{
							PROCESSORID: inst.req.ProcessorID,
							REQUESTID:   inst.req.ID,
							TRANSPORTID: ID(inst.trans),
						},
					}
				})
			}
		}
	}
}
