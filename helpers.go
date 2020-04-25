package atomizer

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// _ctx returns a valid context with cancel even if it the
// supplied context is initially nil. If the supplied context
// is nil it uses the background context
func _ctx(c context.Context) (context.Context, context.CancelFunc) {
	if c == nil {
		c = context.Background()
	}

	return context.WithCancel(c)
}

// _ctxT returns a context with a timeout that is passed in as a time.Duration
func _ctxT(
	c context.Context,
	duration time.Duration,
) (context.Context, context.CancelFunc) {
	if c == nil {
		c = context.Background()
	}

	return context.WithTimeout(c, duration)
}

// recInst recovers an instance inside of atomizer
// accepting the atom and electron id where applicable
// so that a more complete error is pushed to an event
func recInst(aID, eID string) error {

	if r := recover(); r != nil {

		return Error{
			Event: Event{
				Message:    "panic in atomizer",
				AtomID:     aID,
				ElectronID: eID,
			},
			Internal: ptoe(r),
		}

	}

	return nil
}

// rec recovers from a panic, creating an atomizer error
func rec() error {
	return recInst("", "")
}

// ID returns the registration id for the passed in object type
func ID(v interface{}) string {
	return strings.Trim(fmt.Sprintf("%T", v), "*")
}
