package atomizer

import (
	"encoding/gob"
	"strings"
)

func init() {
	gob.Register(Event{})
}

// Event indicates an atomizer event has taken
// place that is not categorized as an error
// Event implements the stringer interface but
// does NOT implement the error interface
type Event struct {
	// Message from atomizer about this error
	Message string `json:"message"`

	// ElectronID is the associated electron instance
	// where the error occurred. Empty ElectronID indicates
	// the error was not part of a running electron instance.
	ElectronID string `json:"electronID"`

	// AtomID is the atom which was processing when
	// the error occurred. Empty AtomID indicates
	// the error was not part of a running atom.
	AtomID string `json:"atomID"`

	// ConductorID is the conductor which was being
	// used for receiving instructions
	ConductorID string `json:"conductorID"`
}

func (e Event) String() string {

	var joins []string

	ids := e.ids()
	if ids != "" {
		joins = append(joins, "["+ids+"]")
	}

	// Add the message to part of the error
	if e.Message != "" {
		joins = append(joins, e.Message)
	}

	return strings.Join(joins, " ")
}

// ids returns the electron and atom ids as a combination string
func (e Event) ids() string {
	var ids []string

	// Include the conductor id if it is part of the event
	if e.ConductorID != "" {
		ids = append(ids, "cid:"+e.ConductorID)
	}

	// Include the atom id if it is part of the event
	if e.AtomID != "" {
		ids = append(ids, "aid:"+e.AtomID)
	}

	// Include the electron id if it is part of the event
	if e.ElectronID != "" {
		ids = append(ids, "eid:"+e.ElectronID)
	}

	return strings.Join(ids, " | ")
}

// makeEvent creates a base event with only a message and
// a time
func makeEvent(msg string) Event {
	return Event{
		Message: msg,
	}
}
