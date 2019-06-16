package atomizer

import (
	"context"
	"time"

	"github.com/benji-vesterby/validator"
)

type instance struct {
	electron   Electron
	conductor  Conductor
	atom       Atom
	properties *properties
	cancel     context.CancelFunc

	// TODO: add an actions channel here that the monitor can keep an eye on for this bonded electron/atom combo
}

func (inst *instance) bond(atom Atom) (err error) {

	if validator.IsValid(inst.electron) {

		if validator.IsValid(inst.conductor) {

			inst.atom = atom

			if !validator.IsValid(inst.atom) {
				// TODO:
			}
		} else {
			// TODO:
		}
	} else {
		// TODO:
	}

	return err
}

func (inst *instance) execute(ctx context.Context) {

	if validator.IsValid(inst) {

		// Initialize the new context object for the processing of the electron/atom
		if inst.electron.Timeout() != nil {
			// Create a new context for the electron with a timeout
			ctx, inst.cancel = context.WithTimeout(ctx, *inst.electron.Timeout())
		} else {
			// Create a new context for the electron with a cancellation option
			ctx, inst.cancel = context.WithCancel(ctx)
		}

		inst.properties = &properties{
			status: PENDING,
		}

		// TODO: Log the execution of the process method here
		// TODO: build a properties object for this process here to store the results and errors into as they
		// TODO: Setup with a heartbeat for monitoring processing of the bonded atom
		// stream in from the process method
		// Execute the process method of the atom
		var results <-chan []byte
		var errstream <-chan error
		results, errstream = inst.atom.Process(ctx, inst.electron, nil) // TODO: setup outbound

		// Update the status of the bonded atom/electron to show processing
		// TODO: inst.properties.status = PROCESSING

		// Continue looping while either of the channels is non-nil and open
		for results != nil || errstream != nil {
			select {
			// Monitor the instance context for cancellation
			case <-ctx.Done():
				// TODO: mizer.sendLog(fmt.Sprintf("cancelling electron instance [%v] due to cancellation [%s]", ewrap.electron.ID(), instance.ctx.Err().Error()))
			case result, ok := <-results:
				// Append the results from the bonded instance for return through the properties
				if ok {
					inst.properties.AddResult(result)
				} else {
					// nil out the result channel after it's closed so that the loop breaks
					result = nil
				}
			case err, ok := <-errstream:
				// Append the errors from the bonded instance for return through the properties
				if ok {
					inst.properties.AddError(err)
				} else {
					// nil out the err channel after it's closed so that the loop breaks
					errstream = nil
				}
			}
		}

		// TODO: The processing has finished for this bonded atom and the results need to be calculated and the properties sent back to the
		// conductor

		// Set the end time and status in the properties
		inst.properties.end = time.Now()
		inst.properties.status = COMPLETED

		// TODO: Ensure mizer is the proper thing to do here?? I think it needs to close mizer out
		//  at the conductor rather than here... unless the conductor overrode the call back

		// TODO: Execute the callback with the noficiation here?

		// TODO: Execute the electron callback here
		inst.electron.Callback(inst.properties)

	} else {
		// TODO: invalid
	}
}

func (inst *instance) Properties() Properties {
	return inst.properties
}

func (inst *instance) Cancel() (err error) {
	return cancel(inst.cancel)
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
