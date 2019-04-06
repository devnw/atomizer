package registration

import (
	"context"
	"sync"

	"github.com/benji-vesterby/validator"
	"github.com/pkg/errors"
)

// preRegistrations of atoms and conductors that are loaded using init
// by libraries utilizing the atomizer
var preRegistrations sync.Map
var preRegSingleton = sync.Once{}
var regchan chan interface{}

// Registrations returns a channel which contains the init pre-registrations
// for use by the atomizer
func Registrations(ctx context.Context) <-chan interface{} {

	// Initialize the registrations channel
	regchan = make(chan interface{})

	// Execute a singleton instance to read the pre-registations from the sync.Map for init registrations
	// and push those to a channel for use by the reciever
	go preRegSingleton.Do(func() {

		// Range over the map and pass back through channel
		preRegistrations.Range(func(key, value interface{}) bool {
			var ok bool

			select {
			case <-ctx.Done():
				close(regchan)
			default:

				// Pass the value over the channel
				regchan <- value

				// Delete the key from the map to indicate that it's been received
				// TODO: Determine if this is the correct functionality here
				preRegistrations.Delete(key)
			}

			return ok
		})
	})

	return regchan
}

// Register adds entries of different types that are used by the atoimzer
// and allows them to be pre-registered using an init script rather than
// having them passed in later at run time. This is useful for some situations
// where the user may not want to register explicitly
func Register(key interface{}, value interface{}) (err error) {

	// Validate the key coming into the register method
	if validator.IsValid(key) {
		if validator.IsValid(value) {

			// If the regchan is valid and initialized then send the value directly to the receiver
			// this will handle any plugins where init is called at plugin load time so that the
			// different registrations are handled immediately and the map doesn't have to be monitored
			// by the registration methods
			if regchan != nil {
				regchan <- value
			} else {

				// Ensure the key is not being duplicated in the pre-registration map
				if _, ok := preRegistrations.Load(key); !ok {
					preRegistrations.Store(key, value)
				} else {
					err = errors.Errorf("cannot register item [%s] because this key is already in use", key)
				}
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

	preRegistrations = sync.Map{}
	preRegSingleton = sync.Once{}
	regchan = nil
}
