// Copyright © 2019 Developer Network, LLC
//
// This file is subject to the terms and conditions defined in
// file 'LICENSE', which is part of this source code package.

package engine

import (
	"context"
	"time"

	"devnw.com/validator"
)

type instance struct {
	electron   *Electron
	conductor  Conductor
	atom       Atom
	properties *Properties
	ctx        context.Context
	cancel     context.CancelFunc

	// TODO: add an actions channel here that the monitor can keep
	// an eye on for this bonded electron/atom combo
}

// bond bonds an instance of an electron with an instance of the
// corresponding atom in the atomizer registrations such that
// the execute method of the instance can properly exercise the
// Process method of the interface
func (i *instance) bond(atom Atom) (err error) {
	if err = validator.Assert(
		i.electron,
		i.conductor,
		atom,
	); err != nil {
		return &Error{
			Event: &Event{
				Message: "error while bonding atom instance",
				AtomID:  ID(atom),
			},
			Internal: err,
		}
	}

	// register the atom internally because
	// the instance is valid
	i.atom = atom

	return nil
}

// complete marks the completion of execution and pushes
// the results to the conductor
func (i *instance) complete() error {
	// Set the end time and status in the properties
	i.properties.End = time.Now()

	if !validator.Valid(i.conductor) {
		return &Error{
			Event: &Event{
				Message:    "conductor validation failed",
				AtomID:     ID(i.atom),
				ElectronID: i.electron.ID,
			},
		}
	}

	// Push the completed instance properties to the conductor
	return i.conductor.Complete(i.ctx, i.properties)
}

// execute runs the process method on the bonded atom / electron pair
func (i *instance) execute(ctx context.Context) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = &Error{
				Event: &Event{
					Message:    "panic in atomizer",
					AtomID:     ID(i.atom),
					ElectronID: i.electron.ID,
				},
				Internal: ptoe(r),
			}

			return
		}

		// ensure that when this method exits the completion
		// of this instance takes place and is pushed to the
		// conductor
		err = i.complete()
	}()

	// ensure the instance is valid before attempting
	// to execute processing
	if !validator.Valid(i) {
		return &Error{
			Event: &Event{
				Message: "instance validation failed",
				AtomID:  ID(i.atom),
			},
		}
	}

	// Establish internal context
	i.ctx, i.cancel = _ctxT(ctx, i.electron.Timeout)

	i.properties = &Properties{
		ElectronID: i.electron.ID,
		AtomID:     ID(i.atom),
		Start:      time.Now(),
	}

	// TODO: Setup with a heartbeat for monitoring processing of the
	// bonded atom stream in from the process method

	// Execute the process method of the atom
	i.properties.Result, i.properties.Error = i.atom.Process(
		i.ctx, i.conductor, i.electron)

	// TODO: The processing has finished for this bonded atom and the
	// results need to be calculated and the properties sent back to the
	// conductor

	// TODO: Ensure a is the proper thing to do here?? I think it needs
	// to close a out at the conductor rather than here... unless the
	// conductor overrode the call back

	// TODO: Execute the callback with the notification here?
	// TODO: determine if this is the correct location or if this is \
	// something that should be handled purely by the conductor

	// TODO: Handle this properly
	// if inst.electron.Resp != nil {
	//	// Drop the return for this electron onto the channel
	//      // to be sent back to the requester
	// 	inst.electron.Resp <- inst.properties
	// }

	return nil
}

// Validate ensures that the instance has the correct
// non-nil values internally so that it functions properly
func (i *instance) Validate() (valid bool) {
	if i != nil {
		if validator.Valid(
			i.electron,
			i.conductor,
			i.atom) {
			valid = true
		}
	}

	return valid
}

func NewTime(ctx context.Context) <-chan time.Time {
	tchan := make(chan time.Time)

	go func(tchan chan<- time.Time) {
		defer close(tchan)

		for {

			select {
			case <-ctx.Done():
				return
			case tchan <- time.Now():
			}
		}
	}(tchan)

	return tchan
}
