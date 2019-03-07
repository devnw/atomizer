package testing

import "time"

// MockElectron is a struct for mocking electrons for unit testing
type MockElectron struct {
}

// Atom is the mock electron implementation for testing
func (electron MockElectron) Atom() (ID string) {
	return ID
}

// ID is the mock electron implementation for testing
func (electron MockElectron) ID() (ID string) {
	return ID
}

// Payload is the mock electron implementation for testing
func (electron MockElectron) Payload() (payload []byte) {
	return payload
}

// Timeout is the mock electron implementation for testing
func (electron MockElectron) Timeout() (timeout *time.Duration) {
	return timeout
}

// Validate is the mock electron implementation for testing
func (electron MockElectron) Validate() (valid bool) {

	return valid
}

// Callback is the mock electron implementation for testing
func (electron MockElectron) Callback(result []byte) (err error) {

	return err
}
