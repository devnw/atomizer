// Copyright © 2019 Developer Network, LLC
//
// This file is subject to the terms and conditions defined in
// file 'LICENSE', which is part of this source code package.

package engine

import (
	"sync"

	"go.devnw.com/validator"
)

var registrant sync.Map

// Registrations returns a channel which contains the init pre-registrations
// for use by the atomizer
func Registrations() []any {
	registrations := make([]any, 0)

	registrant.Range(func(key, value any) bool {
		registrations = append(registrations, value)
		return true
	})
	return registrations
}

func Get[T any](key string) (T, error) {
	var out T

	value, ok := registrant.Load(key)
	if !ok {
		return out, &Error{
			Msg: "key not found in registry",
			Meta: Metadata{
				"key": key,
			},
		}
	}

	out, ok = value.(T)
	if !ok {
		return out, &Error{
			Msg: "value not found in registry",
			Meta: Metadata{
				"key":  key,
				"type": ID(out),
			},
		}
	}

	return out, nil
}

// Register adds entries of different types that are used by the atomizer
// and allows them to be pre-registered using an init script rather than
// having them passed in later at run time. This is useful for some situations
// where the user may not want to register explicitly
func Register(values ...any) error {
	for _, value := range values {
		// Validate the value coming into the register method
		if !validator.Valid(value) {
			return &Error{
				Msg: "invalid registration",
				Meta: Metadata{
					"type": ID(value),
				},
			}
		}

		// Type assert the value to ensure we're only
		// registering expected values in the maps
		switch v := value.(type) {
		case Processor:
			// Register the atom
			registrant.Store(ID(v), &maker{v})
		case Transport:
			// Registrations using the same key will
			// be overridden
			registrant.Store(ID(v), v)
		default:
			return &Error{
				Msg: "unsupported type",
				Meta: Metadata{
					"type": ID(value),
				},
			}
		}
	}

	return nil
}
