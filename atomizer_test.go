package atomizer

import (
	"context"
	"sync"
	"testing"
)

type invalidconductor struct{}

type validcondcutor struct{ echan <-chan Electron }

func (cond *validcondcutor) Receive() <-chan Electron                      { return cond.echan }
func (cond *validcondcutor) Send(electron Electron) (result <-chan []byte) { return nil }
func (cond *validcondcutor) Validate() (valid bool)                        { return true }

// Tests the atomizer creation method without a conductor
func TestAtomizeNoConductors(t *testing.T) {
	tests := []struct {
		key   string
		value interface{}
		err   bool
	}{
		{
			"ValidTestEmptyConductor",
			nil,
			false,
		},
		{
			"ValidTestValidConductor",
			&validcondcutor{make(<-chan Electron)},
			false,
		},
		{
			"InvalidTestInvalidConductor",
			&invalidconductor{},
			true,
		},
		{
			"InvalidTestNilConductor",
			nil,
			true,
		},
		{
			"InvalidTestInvalidElectronChan",
			&validcondcutor{},
			true,
		},
	}

	for _, test := range tests {
		// Reset sync map for this test
		conductors = sync.Map{}

		// Store the test conductor
		if test.err || (!test.err && test.value != nil) {
			// Store invalid conductor
			conductors.Store(test.key, test.value)
		}

		if _, err := Atomize(context.Background()); !test.err && err != nil {
			t.Errorf("expected success for test [%s] but received error [%s]", test.key, err)
		} else if test.err && err == nil {
			t.Errorf("expected error for test [%s] but received success", test.key)
		}

		// Cleanup sync map for additional tests
		conductors = sync.Map{}
	}
}

// Tests the proper functionality of errors passing over the atomizer channel
func TestAtomizer_Errors(t *testing.T) {

}

// Tests that the exit method properly cleans up the atomizer
func TestAtomizer_Exit(t *testing.T) {

}

// Tests that the log channel out of the atomizer works properly
func TestAtomizer_Logs(t *testing.T) {

}

// Validates the instance of the atomizer
func TestAtomizer_Validate(t *testing.T) {

}

// Benchmarks the creation of an atomizer instance
func BenchmarkAtomize(b *testing.B) {

}

// Benchmarks the cleanup of the atomizer given 1 electron
func BenchmarkAtomizer_Exit1(b *testing.B) {

}

// Benchmarks the cleanup of the atomizer given 10 electrons
func BenchmarkAtomizer_Exit10(b *testing.B) {

}

// Benchmarks the cleanup of the atomizer given 100 electrons
func BenchmarkAtomizer_Exit100(b *testing.B) {

}

// Benchmarks the validation method of the atomizer
func BenchmarkAtomizer_Validate(b *testing.B) {

}
