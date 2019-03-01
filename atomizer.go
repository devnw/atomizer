package atomizer

import (
	"context"
	"sync"
)

type Atomizer struct {

	// Priority Channels
	critical chan Electron
	high     chan Electron
	medium   chan Electron
	low      chan Electron

	// Map for storing the context cancellation functions for each source
	conductorCancelFuncs sync.Map

	// Map for storing the contexts of specific electrons
	electronCancelFuncs sync.Map
}

func Atomize() (atomizer Atomizer, err error) {



	return atomizer, err
}

func (this Atomizer) establishConductors(ctx context.Context) (err error) {

	if this.Validate() {

		// Range over the sources and setup
		conductors.Range(func(key, value interface{}) bool {
			var ok bool

			// Create the source context with a cancellation option and store the cancellation in a sync map
			ctx, ctxFunc := context.WithCancel(ctx)
			this.conductorCancelFuncs.Store(key, ctxFunc)

			var conductor Conductor
			if conductor, ok = value.(Conductor); ok {
				if err := this.receive(ctx, conductor.Receive()); err == nil {
					ok = true
				} else {
					// TODO: Decide how to handle the error here
				}
			}

			return ok
		})
	} else {
		// TODO: Error out
	}

	return err
}

func (this Atomizer) receive(ctx context.Context, electrons <- chan Electron) (err error) {

	if ctx != nil {
		if electrons != nil {
			if this.Validate() {

				// Push off the reading into it's own go routine so that it's concurrent
				go func(ctx context.Context, electrons <- chan Electron) {
					for {
						select {
						case <- ctx.Done():
							// Break the loop to close out the receiver
							break
						case electron,ok := <- electrons:
							if ok {
								switch electron.Priority() {
								case CRITICAL:
									this.critical <- electron
								case HIGH:
									this.high <- electron
								case MEDIUM:
									this.medium <- electron
								case LOW:
									fallthrough
								default:
									// Unknown priority dump to low channel
									this.low <- electron
								}
							} else { // Channel is closed, break out of the loop
								break
							}
						}
					}
				}(ctx, electrons)
			}
		} else {
			// TODO:
		}
	} else {
		// TODO:
	}

	return err
}

func (this Atomizer) bond(electron Electron) (err error) {

	return err
}

func (this Atomizer) Validate() (valid bool) {
	if this.critical != nil {
		if this.high != nil {
			if this.medium != nil {
				if this.low != nil {
					valid = true
				}
			}
		}
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