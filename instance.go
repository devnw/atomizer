// Copyright © 2019 Developer Network, LLC
//
// This file is subject to the terms and conditions defined in
// file 'LICENSE', which is part of this source code package.

package engine

import (
	"context"
	"fmt"
	"time"

	"go.devnw.com/validator"
)

type instance struct {
	req    *Request
	trans  Transport
	proc   Processor
	prop   *Response
	ctx    context.Context
	cancel context.CancelFunc

	// TODO: add an actions channel here that the monitor can keep
	// an eye on for this bonded electron/atom combo
}

// bond bonds an instance of an electron with an instance of the
// corresponding atom in the atomizer registrations such that
// the execute method of the instance can properly exercise the
// Process method of the interface
func (i *instance) bond(atom Processor) (err error) {
	if err = validator.Assert(
		i.req,
		i.trans,
		atom,
	); err != nil {
		return &Error{
			Msg:   "error while bonding atom instance",
			Meta:  Metadata{PROCESSORID: ID(atom)},
			Inner: err,
		}
	}

	// register the atom internally because
	// the instance is valid
	i.proc = atom

	return nil
}

// complete marks the completion of execution and pushes
// the results to the conductor
func (i *instance) complete() error {
	// Set the end time and status in the properties
	i.prop.End = time.Now()

	if !validator.Valid(i.trans) {
		return &Error{
			Msg: "conductor validation failed",
			Meta: Metadata{
				PROCESSORID: ID(i.proc),
				REQUESTID:   i.req.ID,
			},
		}
	}

	// Push the completed instance properties to the conductor
	return i.trans.Complete(i.ctx, i.prop)
}

// execute runs the process method on the bonded atom / electron pair
func (i *instance) execute(ctx context.Context) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = &Error{
				Msg: "panic in atomizer",
				Meta: Metadata{
					PROCESSORID: ID(i.proc),
					REQUESTID:   i.req.ID,
				},
				Inner: fmt.Errorf("%v", r),
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
			Msg: "instance validation failed",
			Meta: Metadata{
				PROCESSORID: ID(i.proc),
				REQUESTID:   i.req.ID,
			},
		}
	}

	// Establish internal context
	i.ctx, i.cancel = i.req.Context(ctx)

	i.prop = &Response{
		RequestID:   i.req.ID,
		ProcessorID: ID(i.proc),
		Start:       time.Now(),
	}

	// TODO: Setup with a heartbeat for monitoring processing of the
	// bonded atom stream in from the process method

	// Execute the process method of the atom
	i.prop.Result, i.prop.Error = i.proc.Process(i.ctx, i.req.Payload)

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
			i.req,
			i.trans,
			i.proc) {
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
