package atomizer

import (
	"context"
	"time"

	"github.com/devnw/alog"
	"github.com/devnw/validator"
)

type instance struct {
	electron   Electron
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

		return Error{
			Event: Event{
				Message:    "error while bonding atom instance",
				AtomID:     ID(atom),
				ElectronID: i.electron.ID,
			},
			Internal: err,
		}
	}

	// register the atom internally because
	// the instance is valid
	i.atom = atom

	return nil
}

// setupCtx branches a new context off for use in
// the instance rather than using the parent context
// which allows for a cancel to be established for each
// instance of a bonded atom/electron pair
func (i *instance) setupCtx(ctx context.Context) (context.Context, error) {

	if err := validator.Assert(i.electron, ctx); err != nil {
		return nil, Error{
			Event: Event{
				Message: "invalid electron in instance",
				AtomID:  ID(i.atom),
			},
			Internal: err,
		}
	}

	// Initialize the new context object for the
	// processing of the electron/atom
	if i.electron.Timeout != nil {

		// Create a new context for the electron with a timeout
		i.ctx, i.cancel = _ctxT(
			ctx,
			*i.electron.Timeout,
		)
	} else {
		// Create a new context for the electron
		// with a cancellation option
		i.ctx, i.cancel = context.WithCancel(ctx)
	}

	return i.ctx, nil
}

// complete marks the completion of execution and pushes
// the results to the conductor
func (i *instance) complete() error {

	// Set the end time and status in the properties
	i.properties.End = time.Now()

	// Push the completed instance properties to the conductor
	return i.conductor.Complete(i.ctx, *i.properties)
}

// execute runs the process method on the bonded atom / electron pair
func (i *instance) execute(ctx context.Context) error {

	defer func() {
		if err := recInst(
			ID(i.atom),
			i.electron.ID,
		); err != nil {
			i.properties.Error = err
		}
	}()

	// ensure the instance is valid before attempting
	// to execute processing
	if !validator.Valid(i) {

		return Error{
			Event: Event{
				Message:    "instance validation failed",
				AtomID:     ID(i.atom),
				ElectronID: i.electron.ID,
			},
		}
	}

	// Establish internal context
	ctx, err := i.setupCtx(ctx)
	if err != nil {
		return Error{
			Event: Event{
				Message: "error setting instance ctx",
			},
			Internal: err,
		}
	}

	i.properties = &Properties{
		ElectronID: i.electron.ID,
		AtomID:     ID(i.atom),
		Start:      time.Now(),
	}

	// ensure that when this method exits the completion
	// of this instance takes place and is pushed to the
	// conductor
	defer func() {
		err = i.complete()
		if err != nil {
			// TODO: fix this, the errors are not propagating up
			alog.Error(err)
		}
	}()

	alog.Printf("executing electron [%s]\n", i.electron.ID)

	// TODO: Setup with a heartbeat for monitoring processing of the
	// bonded atom stream in from the process method

	// Execute the process method of the atom
	i.properties.Result, i.properties.Error = i.atom.Process(
		ctx, i.conductor, i.electron)

	alog.Printf("electron [%s] processed\n", i.electron.ID)

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

	return err
}

// Cancel the instance context
func (i *instance) Cancel() (err error) {
	return wrapcancel(i.cancel)
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
