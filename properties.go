package atomizer

import (
	"time"
)

// TODO: Set it up so that requests can be made to check the properties of a bonded electron / atom at runtime

// Properties is the struct for storing properties information after the processing
// of an atom has completed so that it can be sent to the original requestor
type Properties struct {
	ElectronID string    `json:"electronId"`
	AtomID     string    `json:"atomId"`
	Start      time.Time `json:"starttime"`
	End        time.Time `json:"endtime"`
	Error      error     `json:"error"`
	Result     []byte    `json:"result"`
}
