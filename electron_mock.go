package atomizer

import "time"

// mockelectron is a struct for mocking electrons for unit testing
type mockelectron struct {
}

// Atom is the mock electron implementation for testing
func (electron mockelectron) Atom() (ID string) {
	return ID
}

// ID is the mock electron implementation for testing
func (electron mockelectron) ID() (ID string) {
	return ID
}

// Payload is the mock electron implementation for testing
func (electron mockelectron) Payload() (payload []byte) {
	return payload
}

// Timeout is the mock electron implementation for testing
func (electron mockelectron) Timeout() (timeout *time.Duration) {
	return timeout
}

// Validate is the mock electron implementation for testing
func (electron mockelectron) Validate() (valid bool) {

	return valid
}

// Callback is the mock electron implementation for testing
func (electron mockelectron) Callback(result []byte) (err error) {

	return err
}
