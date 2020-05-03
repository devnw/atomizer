// Copyright © 2019 Developer Network, LLC
//
// This file is subject to the terms and conditions defined in
// file 'LICENSE', which is part of this source code package.

package atomizer

import (
	"encoding/gob"
	"fmt"
	"strings"
)

func init() {
	gob.Register(Error{})
}

type wrappedErr interface {
	Unwrap() error
}

func simple(msg string, internal error) error {
	return Error{
		Event: Event{
			Message: msg,
		},
		Internal: internal,
	}
}

// ptoe takes a result of a recover and coverts it
// to a string
func ptoe(r interface{}) error {
	return Error{
		Event: makeEvent(ptos(r)),
	}
}

// ptos takes a result of a recover and coverts it
// to a string
func ptos(r interface{}) string {
	return fmt.Sprintf("%v", r)
}

// Error is an error type which provides specific
// atomizer information as part of an error
type Error struct {

	// Event is the event that took place to create
	// the error and contains metadata relevant to the error
	Event Event `json:"event"`

	// Internal is the internal error
	Internal error `json:"internal"`
}

func (e Error) Error() string {
	return e.String()
}

func (e Error) String() string {
	var fields []string

	fields = append(fields, "atomizer error")

	msg := e.Event.String()
	if msg != "" {
		fields = append(fields, msg)
	}

	if e.Internal != nil {
		fields = append(
			fields,
			"| internal: ("+e.Internal.Error()+")",
		)
	}

	return strings.Join(fields, " ")
}

// Unwrap unwraps the error to the deepest error and returns that one
func (e Error) Unwrap() (err error) {

	err = e.Internal

	// Determine if the internal error implements
	// the wrappedErr interface then continue unwrapping
	// if it does
	if internal, ok := err.(wrappedErr); ok {

		// Recursive unwrap to get the lowest error
		err = internal.Unwrap()
	}

	return err
}

// Validate determines if this is a properly built error
func (e Error) Validate() bool {
	return e.Event.Validate()
}
