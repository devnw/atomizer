package atomizer

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
)

func handle(ctx context.Context, obj interface{}, recovery func()) (err error) {

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

			// Recover if the context isn't closed
			select {
			case <-ctx.Done():
				// DO NOTHING, LET THE METHOD EXIT
			default:
				go recovery()
			}
		}
	}

	return err
}
