package atomizer

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"sync"
)

// Atomizer facilitates the execution of tasks (aka Electrons) which are received from the configured sources
// these electrons can be distributed across many instances of the Atomizer on different nodes in a distributed
// system or in memory. Atoms should be created to process "atomic" actions which are small in scope and overall
// processing complexity minimizing time to run and allowing for the distributed system to take on the burden of
// long running processes as a whole rather than a single process handling the overall load
type Atomizer struct {

	// Priority Channels
	electrons	chan Electron
	errors		chan error
	logs		chan string
	ctx			context.Context

	// Map for storing the context cancellation functions for each source
	conductorCancelFuncs sync.Map

	// Map for storing the contexts of specific electrons
	electronCancelFuncs sync.Map
}

func Atomize(ctx context.Context) (atomizer Atomizer, err error) {

	// Initialize the atomizer and establish the channels
	atomizer = Atomizer{
		electrons: make(chan Electron),
		ctx: ctx,
	}

	// Start up the receivers
	err = atomizer.receive()

	return atomizer, err
}

// Create a channel to receive errors from the Atomizer and return the channel for logging purposes
func (this Atomizer) Errors(buffer int) <- chan error {
	if this.errors == nil {

		// Ensure that a proper buffer size was passed for the channel
		if buffer < 0 {
			buffer = 0
		}

		// Only upon request should the error channel be established meaning a user should read from the channel
		this.errors = make(chan error, buffer)
	}

	return this.errors
}

// Create a channel to receive logs from the Atomizer, and electrons
func (this Atomizer) Logs(buffer int) <- chan string {

	if this.logs == nil {

		// Ensure that a proper buffer size was passed for the channel
		if buffer < 0 {
			buffer = 0
		}

		// Only upon request should the log channel be established meaning a user should read from the channel
		this.logs = make(chan string, buffer)
	}

	return this.logs
}

// If the error channel is not nil then send the error on the channel
func (this Atomizer) sendErr(err error) {
	if this.errors != nil {
		go func(err error) {
			this.errors <- err
		}(err)
	}
}

// if the log channel is not nil then send the log on the channel
func (this Atomizer) sendLog(log string) {
	if this.logs != nil {
		go func(log string) {
			this.logs <- log
		}(log)
	}
}

// Initialize the go routines that will read from the conductors concurrently while other parts of the
// atomizer reads in the inputs and executes the instances of electrons
func (this Atomizer) receive() (err error) {

	// Validate inputs
	if this.ctx != nil {

		// Validate the atomizer instance
		if this.Validate() {

			// Range over the sources and setup
			conductors.Range(func(key, value interface{}) bool {
				var ok bool

				select {
				case <- this.ctx.Done(): // Do nothing so that the loop breaks immediately
				default:

					// Create the source context with a cancellation option and store the cancellation in a sync map
					ctx, ctxFunc := context.WithCancel(this.ctx)
					this.conductorCancelFuncs.Store(key, ctxFunc)

					var conductor Conductor
					if conductor, ok = value.(Conductor); ok {
						var electrons = conductor.Receive()

						if electrons != nil {

							// Push off the reading into it's own go routine so that it's concurrent
							go this.distribute(ctx, key, electrons)
							ok = true
						} else {
							err = errors.Errorf("invalid electron channel for conductor [%s]", key)
						}
					}
				}

				return ok
			})
		} else {
			err = errors.Errorf("Invalid Atomizer [%v]", this)
		}
	}

	return err
}

// Reading in from a specific electron channel of a conductor and drop it onto the atomizer channel for electrons
func (this Atomizer) distribute(ctx context.Context, conductor interface{}, electrons <- chan Electron) {
	defer func() {
		if r := recover(); r != nil {
			this.sendLog(fmt.Sprintf("panic in distribute [%v] for conductor [%s]; restarting distributor", r, conductor))

			// Self Heal - Re-initialize this distribute method in event of panic
			go this.distribute(ctx, conductor, electrons)

		} else { // TODO: Determine if this is the correct action if the electron channel is closed
			// clean up the cancel method for this conductor and close the context
			if cFunc, ok := this.conductorCancelFuncs.Load(conductor); ok {

				// Type assert the cancel method and execute it to close the context
				if cancel, ok := cFunc.(context.CancelFunc); ok {

					// Execute the cancel method for this conductor
					cancel()
					this.sendLog(fmt.Sprintf("context for conductor [%s] successfully cancelled", conductor))
				}
			} else {
				this.sendErr(errors.Errorf("unable to find context cancellation function for conductor [%s]", conductor))
			}
		}
	}()

	// Read from the electron channel for this conductor and push onto the atomizer electron channel for processing
	for {
		select {
		case <- ctx.Done():
			// Break the loop to close out the receiver
			break
		case electron, ok := <- electrons:
			if ok {
				this.electrons <- electron
			} else { // Channel is closed, break out of the loop
				this.sendErr(errors.Errorf("electron channel for conductor [%s] is closed, exiting read cycle", conductor))
				break
			}
		}
	}
}

func (this Atomizer) spin() {

}

func (this Atomizer) bond(electron Electron) (err error) {

	return err
}

func (this Atomizer) Exit()  {
	// Make a clean exit for atomizer
}

func (this Atomizer) Validate() (valid bool) {

	if this.electrons != nil {
		valid = true
	}

	return valid
}

// Pull from a sync map containing cancellation functions for contexts
// if the sync map contains instances of context.CancelFunc then execute
// the cancellation method for that id in the sync map, otherwise error to
// the calling method
func (this Atomizer) nuke(cmap sync.Map, id string) (err error) {
	if len(id) > 0 {
		if cfunc, ok := cmap.Load(id); ok {
			if cancel, ok := cfunc.(context.CancelFunc); ok {

				// Cancel this sub-contextS
				cancel()
			} else {
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