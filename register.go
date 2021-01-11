// Copyright © 2019 Developer Network, LLC
//
// This file is subject to the terms and conditions defined in
// file 'LICENSE', which is part of this source code package.

package atomizer

import (
	"fmt"
	"sync"

	"github.com/devnw/validator"
)

var registrant sync.Map

// Registrations returns a channel which contains the init pre-registrations
// for use by the atomizer
func Registrations() []interface{} {
	registrations := make([]interface{}, 0)

	registrant.Range(func(key, value interface{}) bool {

		registrations = append(registrations, value)
		return true
	})
	return registrations
}

// Register adds entries of different types that are used by the atomizer
// and allows them to be pre-registered using an init script rather than
// having them passed in later at run time. This is useful for some situations
// where the user may not want to register explicitly
func Register(values ...interface{}) error {

	for _, value := range values {
		// Validate the value coming into the register method
		if !validator.Valid(value) {
			return simple(
				fmt.Sprintf(
					"Invalid registration %s",
					ID(value)),
				nil,
			)

		}

		// Type assert the value to ensure we're only
		// registering expected values in the maps
		switch value.(type) {
		case Conductor, Atom:

			// Registrations using the same key will
			// be overridden
			registrant.Store(ID(value), value)
		default:
			return simple(
				fmt.Sprintf(
					"unsupported type %s",
					ID(value)),
				nil,
			)
		}
	}

	return nil
}
