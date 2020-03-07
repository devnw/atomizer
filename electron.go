package atomizer

import (
	"encoding/json"
	"time"
)

// Electron is the base electron that MUST parse from the payload from the conductor
type Electron struct {
	// SenderID is the identifier for the node that sent the electron
	SenderID string `json:"senderid"`

	// ID is the identifier of this electron
	ID string `json:"id"`

	// AtomID is the identifier of the atom for this electron instance
	AtomID string `json:"atomid"`

	// Timeout is the maximum time duration that should be allowed for this instance
	// to process. After the duration is exceeded the context should be cancelled and
	// the processing released and a failure sent back to the conductor
	Timeout *time.Duration `json:"timeout,omitempty"`

	// Payload is to be used by the registered atom to properly unmarshal
	// the json for the actual atom instance. RawMessage is used to delay unmarshal
	// of the payload information so the atom can do it internally
	Payload json.RawMessage `json:"payload"`
}

// Validate ensures that the electron information is intact for proper execution
func (e *Electron) Validate() (valid bool) {
	if len(e.SenderID) > 0 {
		if len(e.ID) > 0 {
			if len(e.AtomID) > 0 {
				valid = true
			}
		}
	}

	return valid
}