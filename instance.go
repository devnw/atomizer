package atomizer

import (
	"context"
	"fmt"
	"time"

	"github.com/benjivesterby/alog"
	"github.com/benjivesterby/validator"
	"github.com/pkg/errors"
)

type instance struct {
	electron   *Electron
	conductor  Conductor
	atom       Atom
	properties *Properties
	cancel     context.CancelFunc

	// TODO: add an actions channel here that the monitor can keep an eye on for this bonded electron/atom combo
}

func (i *instance) bond(atom Atom) (err error) {

	if err = validator.Assert(i.electron, i.conductor, atom); err == nil {
		i.atom = atom
	} else {
		err = aErr{err, fmt.Sprintf("error while bonding atom instance %s", ID(atom))}
	}

	return err
}

// execute runs the process method on the bonded atom / electron pair
func (i *instance) execute(ctx context.Context) {

	// Initialize the new context object for the processing of the electron/atom
	if i.electron.Timeout != nil {
		// Create a new context for the electron with a timeout
		ctx, i.cancel = context.WithTimeout(ctx, *i.electron.Timeout)
	} else {
		// Create a new context for the electron with a cancellation option
		ctx, i.cancel = context.WithCancel(ctx)
	}

	i.properties = &Properties{
		ElectronID: i.electron.ID,
		AtomID:     ID(i.atom),
		Start:      time.Now(),
	}

	// Setup defer for this instance
	defer func() {
		if r := recover(); r != nil {
			i.properties.Error = errors.Errorf(
				"panic in processing of electron [%s] | panic: [%s] | error:[%s]",
				i.electron.ID,
				r,
				i.properties.Error,
			)
		}

		// Set the end time and status in the properties
		i.properties.End = time.Now()

		if err := i.conductor.Complete(ctx, i.properties); err == nil {
			alog.Printf("completed electron [%s]\n", i.electron.ID)
		} else {
			alog.Errorf(err, "error marking electron [%s] as complete", i.electron.ID)
		}
	}()

	alog.Printf("executing electron [%s]\n", i.electron.ID)

	// TODO: Setup with a heartbeat for monitoring processing of the bonded atom
	// stream in from the process method
	// Execute the process method of the atom
	i.properties.Result, i.properties.Error = i.atom.Process(ctx, i.conductor, i.electron)
	alog.Printf("electron [%s] processed\n", i.electron.ID)

	// TODO: The processing has finished for this bonded atom and the results need to be calculated and the properties sent back to the
	// conductor

	// TODO: Ensure a is the proper thing to do here?? I think it needs to close a out
	//  at the conductor rather than here... unless the conductor overrode the call back

	// TODO: Execute the callback with the notification here?
	// TODO: determine if this is the correct location or if this is something that should be handled purely by the conductor
	// TODO: Handle this properly
	// if inst.electron.Resp != nil {
	// 	// Drop the return for this electron onto the channel to be sent back to the requester
	// 	inst.electron.Resp <- inst.properties
	// }
}

func (i *instance) Properties() *Properties {
	return i.properties
}

func (i *instance) Cancel() (err error) {
	return wrapcancel(i.cancel)
}

func (i *instance) Validate() (valid bool) {

	if validator.Valid(i.electron, i.conductor, i.atom) {
		valid = true
	}

	return valid
}
