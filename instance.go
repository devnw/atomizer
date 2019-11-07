package atomizer

import (
	"context"
	"fmt"
	"time"

	"github.com/benjivesterby/validator"
	"github.com/pkg/errors"
)

type instance struct {
	electron   *ElectronBase
	conductor  Conductor
	atom       Atom
	properties *properties
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
				err = errors.Errorf("invalid atom [%s] when attempting to bond", atom.ID())
			}
		} else {
			err = errors.Errorf("invalid conductor [%v] when attempting to bond", inst.conductor.ID())
		}
	} else {
		err = errors.Errorf("invalid electron [%s] when attempting to bond", inst.electron.ElectronID)
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

		inst.properties = &properties{
			electronID: inst.electron.ElectronID,
			atomID:     inst.atom.ID(),
			start:      time.Now(),
		}

		// Setup defer for this instance
		defer func() {

			// Set the end time and status in the properties
			inst.properties.end = time.Now()

			if err := inst.conductor.Complete(ctx, inst.properties); err != nil {
				fmt.Println(err.Error())
			}
		}()

		fmt.Printf("executing electron [%s]\n", inst.electron.ElectronID)

		// TODO: Log the execution of the process method here
		// TODO: build a properties object for this process here to store the results and errors into as they
		// TODO: Setup with a heartbeat for monitoring processing of the bonded atom
		// stream in from the process method
		// Execute the process method of the atom
		var results <-chan []byte
		results = inst.atom.Process(ctx, inst.electron, nil) // TODO: setup outbound

		fmt.Printf("electron [%s] processed\n", inst.electron.ElectronID)

		func() {
			// Continue looping while either of the channels is non-nil and open
			for results != nil {
				select {
				// Monitor the instance context for cancellation
				case <-ctx.Done():
					// TODO: mizer.sendLog(fmt.Sprintf("cancelling electron instance [%v] due to cancellation [%s]", ewrap.electron.ID(), instance.ctx.Err().Error()))
					return
				case result, ok := <-results:
					if ok {
						// Append the results from the bonded instance for return through the properties
						inst.properties.result = result
					}
					return
				}
			}
		}()

		// TODO: The processing has finished for this bonded atom and the results need to be calculated and the properties sent back to the
		// conductor

		// TODO: Ensure mizer is the proper thing to do here?? I think it needs to close mizer out
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

func (inst *instance) Properties() Properties {
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
