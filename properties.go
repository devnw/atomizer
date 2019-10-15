package atomizer

import (
	"time"
)

// TODO: Set it up so that requests can be made to check the properties of a bonded electron / atom at runtime

// properties is the struct for storing properties information after the processing
// of an atom has completed so that it can be sent to the original requestor
type properties struct {
	electronID string
	atomID     string
	start      time.Time
	end        time.Time
	err        error
	result     []byte
}

// ElectronID returns the identifier of the electron
func (prop *properties) ElectronID() string {
	return prop.electronID
}

// AtomID returns the identifier of the atom
func (prop *properties) AtomID() string {
	return prop.atomID
}

// StartTime indicates the time the processing of an atom began (UTC)
func (prop *properties) StartTime() time.Time {
	return prop.start
}

// EndTime indicates the time the processing of an atom ended (UTC)
func (prop *properties) EndTime() time.Time {
	return prop.end
}

// Duration returns the duration of the process method on an atom for analysis by the calling system
func (prop *properties) Duration() time.Duration {
	return prop.start.Sub(prop.end)
}

// Error return an error if one occurred during processing
func (prop *properties) Error() error {
	return prop.err
}

// Result returns the list of results which are also byte slices
func (prop *properties) Result() []byte {
	return prop.result
}
