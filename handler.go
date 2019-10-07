package atomizer

import (
	"fmt"

	"github.com/pkg/errors"
)

func nuke(obj interface{}) (err error) {

	if c, ok := obj.(cancelable); ok {
		// Cancel the object
		err = c.Cancel()

	}

	return err
}

func handle(obj interface{}, recovery func()) (err error) {

	// Only nuke the object if it's valid
	if obj != nil {

		// Nuke the object if it's cancelable
		if err = nuke(obj); err != nil {
			err = errors.Errorf("error while cancelling context | %s", err.Error())
		}
	}

	// Handle the panic scenario by re-queueing the receiver for this conductor
	if r := recover(); r != nil {

		// Add the other errors above the panic in the error
		var prelim string
		if err != nil {
			prelim = fmt.Sprintf("%s | ", err.Error())
		}

		err = errors.Errorf("%s panic occurred [%s]", prelim, r)

		// Initiate the recovery of the method that had a panic
		if recovery != nil {
			go recovery()
		}
	}

	return err
}
