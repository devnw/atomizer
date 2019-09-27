package atomizer

import (
	"encoding/json"
	"time"
)

// Electron is the interface which should be implemented by the messages sent through the conductors that trigger
// instances of each atom for processing.
type Electron interface {

	// ID returns the identifier for this electron
	ID() string

	// Payload returns the Raw Json Payload that was passed from the conductor
	Payload() *json.RawMessage

	// Respond returns the response channel for the properties that this electron
	// returns after processing
	Respond() chan<- Properties
}

// ElectronBase is the base electron that MUST parse from the payload from the conductor
type ElectronBase struct {

	// ID is the identifier of this electron
	ElectronID string `json:"id"`

	// AtomID is the identifier of the atom for this electron instance
	AtomID string `json:"atomid"`

	// Timeout is the maximum time duration that should be allowed for this instance
	// to process. After the duration is exceeded the context should be cancelled and
	// the processing released and a failure sent back to the conductor
	Timeout *time.Duration `json:"timeout"`

	// Load is to be used by the registered atom to properly unmarshall
	// the json for the actual atom instance. RawMessage is used to delay unmarshalling
	// of the payload information so the atom can do it internally
	Load *json.RawMessage `json:"payload,omitempty"`

	// Resp is the channel that returns messages from this spawned
	// instance of the electron. The channel allows for blocking and if
	// the channel is nil it will be ignored and no responses will be returned
	Resp chan Properties `json:"-"`
}

// ID returns the identifier for this electron
func (e *ElectronBase) ID() string {
	return e.ElectronID
}

// Payload returns the Raw Json Payload that was passed from the conductor
func (e *ElectronBase) Payload() *json.RawMessage {
	return e.Load
}

// Respond returns the response channel for the properties that this electron
// returns after processing
func (e *ElectronBase) Respond() chan<- Properties {
	return e.Resp
}

// Validate ensures that the electron information is intact for proper execution
func (e *ElectronBase) Validate() (valid bool) {
	if len(e.ElectronID) > 0 {
		if len(e.AtomID) > 0 {
			valid = true
		}
	}

	return valid
}
