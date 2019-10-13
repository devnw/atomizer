package atomizer

import "time"

// Properties tracks the performance, status and errors that occur in an atom to be passed to the original requestor
// This should be returned to the sender over a properties channel
type Properties interface {

	// ElectronID returns the identifier of the Electron
	ElectronID() string

	// AtomID returns the identifier of the Atom
	AtomID() string

	// StartTime indicates the time the processing of an atom began (UTC)
	StartTime() (start time.Time)

	// EndTime indicates the time the processing of an atom ended (UTC)
	EndTime() (end time.Time)

	// Duration returns the duration of the process method on an atom for analysis by the calling system
	Duration() (duration time.Duration)

	// Status is the status of the atom at the time the processing completed
	Status() (status int)

	// Errors returns the list of errors that occurred on this atom after all the processing had been completed
	Errors() (errors []error)

	// Results returns the results of the processing
	Results() (results []byte)
}
