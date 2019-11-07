package atomizer

import (
	"context"
	"sync"

	"github.com/benjivesterby/alog"
	"github.com/benjivesterby/validator"
	"github.com/pkg/errors"
)

// preRegistrations of atoms and conductors that are loaded using init
// by libraries utilizing the atomizer
var preRegistrations sync.Map

// TODO: Should this be a constant?
var preRegSingleton = sync.Once{}
var regchan chan interface{}

// Used to get rid of race conditions for shared resources in registration
var mutty = sync.RWMutex{}

// Registrations returns a channel which contains the init pre-registrations
// for use by the atomizer
func Registrations(ctx context.Context) <-chan interface{} {

	// Initialize the registrations channel
	mutty.Lock()
	regchan = make(chan interface{})
	mutty.Unlock()

	// Execute a singleton instance to read the pre-registations from the sync.Map for init registrations
	// and push those to a channel for use by the receiver
	preRegSingleton.Do(func() {

		// Push off in go routine so that the channel can be returned immediately
		go func() {

			// Lock the reads and modifications of the pre-registrations to keep from
			// having race conditions in the registrations
			mutty.RLock()
			preRegistrations.Range(func(key, value interface{}) bool {
				var ok bool

				alog.Printf("pushing pre-registration %v", key)

				// Pass the value over the channel
				select {
				case <-ctx.Done():
					close(regchan)
				case regchan <- value:
					alog.Printf("deleting pre-registration key %v", key)

					// Delete the key from the map to indicate that it's been received
					preRegistrations.Delete(key)

					ok = true
				}

				return ok
			})
			mutty.RUnlock()
		}()
	})

	return regchan
}

// Register adds entries of different types that are used by the atomizer
// and allows them to be pre-registered using an init script rather than
// having them passed in later at run time. This is useful for some situations
// where the user may not want to register explicitly
func Register(ctx context.Context, key interface{}, value interface{}) (err error) {

	// Validate the key coming into the register method
	if validator.IsValid(key) {
		if validator.IsValid(value) {

			// Type assert the value to ensure we're only registering expected values in the maps
			switch value.(type) {
			case Conductor, Atom:

				// If the regchan is valid and initialized then send the value directly to the receiver
				// this will handle any plugins where init is called at plugin load time so that the
				// different registrations are handled immediately and the map doesn't have to be monitored
				// by the registration methods
				if regchan != nil {
					select {
					case <-ctx.Done():
						err = ctx.Err()
						return
					case regchan <- value:
					}
				} else {

					// Ensure the key is not being duplicated in the pre-registration map
					if _, ok := preRegistrations.Load(key); !ok {
						preRegistrations.Store(key, value)
					} else {
						err = errors.Errorf("cannot register item [%s] because this key is already in use", key)
					}
				}
			default:
				err = errors.Errorf("cannot register item [%s] because it is not a supported type", key)
			}
		} else {
			err = errors.Errorf("cannot register item [%s] because it is invalid", key)
		}
	} else {
		err = errors.Errorf("key is empty; cannot register value [%v]", value)
	}

	return err
}

// Clean clears out the pre-registered values and channels uses for registration
// this method is primarily for testing and should not be used otherwise
func Clean() {

	mutty.Lock()
	preRegistrations = sync.Map{}
	preRegSingleton = sync.Once{}
	regchan = nil
	mutty.Unlock()
}
