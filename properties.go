package atomizer

import (
	"time"
)

// Properties tracks the performance, status and errors that occur in an atom to be passed to the original requestor
// This should be returned to the sender over a notification channel
type Properties interface {

	// StartTime indicates the time the processing of an atom began (UTC)
	StartTime() (start time.Time)

	// EndTime indicates the time the processing of an atom ended (UTC)
	EndTime() (end time.Time)

	// ProcessingTime returns the duration of the process method on an atom for analysis by the calling system
	ProcessingTime() (ptime time.Duration)

	// Status is the status of the atom at the time the processing completed
	Status() (status int)

	// Errors returns the list of errors that occured on this atom after all the processing had been completed
	Errors() (errors []error)

	// Results returns the list of results which are also byte slices
	Results() (results [][]byte)
}

// properties is the struct for storing properties information after the processing
// of an atom has completed so that it can be sent to the original requestor
type properties struct {
	start   time.Time
	end     time.Time
	status  int
	errs    []error
	results [][]byte
}

// StartTime indicates the time the processing of an atom began (UTC)
func (prop *properties) StartTime() (start time.Time) {

	return prop.start
}

// EndTime indicates the time the processing of an atom ended (UTC)
func (prop *properties) EndTime() (end time.Time) {

	return prop.end
}

// ProcessingTime returns the duration of the process method on an atom for analysis by the calling system
func (prop *properties) ProcessingTime() (ptime time.Duration) {
	return prop.start.Sub(prop.end)
}

// Status is the status of the atom at the time the processing completed
func (prop *properties) Status() (status int) {
	return prop.status
}

// Errors returns the list of errors that occured on this atom after all the processing had been completed
func (prop *properties) Errors() (errors []error) {
	return prop.errs
}

// Results returns the list of results which are also byte slices
func (prop *properties) Results() (results [][]byte) {
	return prop.results
}

// AddResult adds a result entry to the notification
func (prop *properties) AddResult(result []byte) {

	// Only add the result if it's a valid result
	if result != nil && len(result) > 0 {
		prop.results = append(prop.results, result)
	}
}

// AddError adds an error entry to the notification
func (prop *properties) AddError(err error) {

	// Only add the error if it's non-nil
	if err != nil {
		prop.errs = append(prop.errs, err)
	}
}
