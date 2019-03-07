package atomizer

import (
	"github.com/benji-vesterby/validator"
	"github.com/pkg/errors"
	"sync"
)

// Sync map that contains the atoms available to this instance of atomizer
var atoms sync.Map

// Sync map that contains the conductors available to pull atoms from for this atomizer
var conductors sync.Map

// RegisterAtom registers an atom for execution
func RegisterAtom(identifier string, atom Atom) {
	register(&atoms, identifier, atom)
}

// RegisterSource registers a source to collect atoms from
func RegisterSource(identifier string, conductor Conductor) {
	register(&conductors, identifier, conductor)
}

// Using the passed in sync map register the id and item into the
// sync map. This method is primarily used to register map entries
// as part of the init script for conductors and individual atoms
func register(smap *sync.Map, id interface{}, item interface{}) (err error) {

	// Validate the key coming into the register method
	if validator.IsValid(id) {
		if validator.IsValid(item) {
			if _, ok := smap.Load(id); !ok {
				smap.Store(id, item)
			} else {
				err = errors.Errorf("cannot register item [%s] because this key is already in use", id)
			}
		} else {
			err = errors.Errorf("cannot register item [%s] because it is invalid", id)
		}
	} else {
		err = errors.Errorf("key is empty; cannot register value [%v]", item)
	}

	return err
}
