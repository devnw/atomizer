package atomizer

import (
	"context"
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

func (inst *instance) bond(atom Atom) (err error) {

	if validator.IsValid(inst.electron) {
		if validator.IsValid(inst.conductor) {

			// Ensure the atom passed in is valid
			if validator.IsValid(atom) {
				inst.atom = atom
			} else {
				err = errors.Errorf("invalid atom [%s] when attempting to bond", ID(atom))
			}
		} else {
			err = errors.Errorf("invalid conductor [%s] when attempting to bond", ID(inst.conductor))
		}
	} else {
		err = errors.Errorf("invalid electron [%s] when attempting to bond", inst.electron.ID)
	}

	return err
}

// execute runs the process method on the bonded atom / electron pair
func (inst *instance) execute(ctx context.Context) {

	if validator.IsValid(inst) {

		// Initialize the new context object for the processing of the electron/atom
		if inst.electron.Timeout != nil {
			// Create a new context for the electron with a timeout
			ctx, inst.cancel = context.WithTimeout(ctx, *inst.electron.Timeout)
		} else {
			// Create a new context for the electron with a cancellation option
			ctx, inst.cancel = context.WithCancel(ctx)
		}

		inst.properties = &Properties{
			ElectronID: inst.electron.ID,
			AtomID:     ID(inst.atom),
			Start:      time.Now(),
		}

		// Setup defer for this instance
		defer func() {
			if r := recover(); r != nil {
				inst.properties.Error = errors.Errorf(
					"panic in processing of electron [%s] | panic: [%s] | error:[%s]",
					inst.electron.ID,
					r,
					inst.properties.Error,
				)
			}

			// Set the end time and status in the properties
			inst.properties.End = time.Now()

			if err := inst.conductor.Complete(ctx, inst.properties); err == nil {
				alog.Printf("completed electron [%s]\n", inst.electron.ID)
			} else {
				alog.Errorf(err, "error marking electron [%s] as complete", inst.electron.ID)
			}
		}()

		alog.Printf("executing electron [%s]\n", inst.electron.ID)

		// TODO: Setup with a heartbeat for monitoring processing of the bonded atom
		// stream in from the process method
		// Execute the process method of the atom
		inst.properties.Result, inst.properties.Error = inst.atom.Process(ctx, inst.conductor, inst.electron)
		alog.Printf("electron [%s] processed\n", inst.electron.ID)

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

	} else {
		// TODO: invalid
	}
}

// func (inst *instance) outbound(ctx context.Context) chan<- Electron {
// 	electrons := make(chan Electron)

// 	go func(electrons chan Electron) {
// 		defer close(electrons)

// 		for {
// 			select {
// 			case <-ctx.Done():
// 				return
// 			case e, ok := <-electrons:
// 				if ok {
// 					// Push the electron to the conductor
// 					if err := inst.conductor.Send(ctx, e); err != nil && e.Respond() != nil {
// 						select {
// 						case e.Respond() <- &Properties{
// 							ElectronID: e.ID(),
// 							AtomID:     e.AID(),
// 							Error:      err,
// 						}:
// 							alog.Errorf(err, "error occurred while sending outbound electron [%s]", e.ID())
// 						}
// 					}
// 				} else {
// 					return
// 				}
// 			}
// 		}

// 	}(electrons)

// 	return electrons
// }

func (inst *instance) Properties() *Properties {
	return inst.properties
}

func (inst *instance) Cancel() (err error) {
	return wrapcancel(inst.cancel)
}

func (inst *instance) Validate() (valid bool) {

	if validator.IsValid(inst.electron) &&
		validator.IsValid(inst.conductor) &&
		validator.IsValid(inst.atom) {
		{
			valid = true
		}
	}

	return valid
}
