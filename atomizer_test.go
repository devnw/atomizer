package atomizer

import (
	"context"
	"testing"

	"github.com/benji-vesterby/atomizer/registration"

	"github.com/benji-vesterby/atomizer/interfaces"
	"github.com/benji-vesterby/validator"
)

type invalidconductor struct{}

type validcondcutor struct {
	id    string
	echan <-chan interfaces.Electron
	valid bool
}

func (cond *validcondcutor) ID() string                                               { return cond.id }
func (cond *validcondcutor) Receive() <-chan interfaces.Electron                      { return cond.echan }
func (cond *validcondcutor) Send(electron interfaces.Electron) (result <-chan []byte) { return nil }
func (cond *validcondcutor) Validate() (valid bool)                                   { return cond.valid }
func (cond *validcondcutor) Complete(properties interfaces.Properties)                {}

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
			&validcondcutor{"ValidTestValidConductor", make(<-chan interfaces.Electron), true},
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
		registration.Clean()

		// Store the test conductor
		if test.err || (!test.err && test.value != nil) {
			// Store invalid conductor
			registration.Register(test.key, test.value)
		}

		if _, err := Atomize(context.Background()); !test.err && err != nil {
			t.Errorf("expected success for test [%s] but received error [%s]", test.key, err)
		} else if test.err && err == nil {
			t.Errorf("expected error for test [%s] but received success", test.key)
		}

		// Cleanup sync map for additional tests
		registration.Clean()
	}
}

func TestAtomizer_AddConductor(t *testing.T) {
	tests := []struct {
		key   string
		value interfaces.Conductor
		err   bool
	}{
		{
			"ValidTestEmptyConductor",
			&validcondcutor{"ValidTestEmptyConductor", make(<-chan interfaces.Electron), true},
			false,
		},
		{
			"InvalidTestConductor",
			&validcondcutor{"InvalidTestConductor", make(<-chan interfaces.Electron), false},
			true,
		},
		{
			"InvalidTestConductorNilElectron",
			&validcondcutor{"InvalidTestConductorNilElectron", nil, true},
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
		{ // Empty key test
			"",
			&validcondcutor{},
			true,
		},
	}

	for _, test := range tests {
		// Reset sync map for this test
		registration.Clean()

		// Create an instance of the atomizer to test the add conductor with
		if mizer, err := Atomize(context.Background()); err == nil {

			// Add the conductor
			if err = mizer.Register(test.value); !test.err && err != nil {
				t.Errorf("expected success for test [%s] but received error while adding atomizer [%s]", test.key, err)
			} else if test.err && err == nil {
				t.Errorf("expected error for test [%s] but received success", test.key)
			}

		} else {
			t.Errorf("expected successful atomizer creation for test [%s] but received error while initializing atomizer [%s]", test.key, err.Error())
		}

		// Cleanup sync map for additional tests
		registration.Clean()
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
	tests := []struct {
		key   string
		value interface{}
		err   bool
	}{
		{
			"ValidAtomizerTest",
			&atomizer{
				electrons: make(chan instance),
				bonded:    make(chan instance),
				ctx:       context.Background(),
				cancel: context.CancelFunc(func() {

				}),
			},
			false,
		},
		{
			"InvalidAtomizerNilElectrons",
			&atomizer{
				electrons: nil,
				bonded:    make(chan instance),
				ctx:       context.Background(),
				cancel: context.CancelFunc(func() {

				}),
			},
			true,
		},
		{
			"InvalidAtomizerNilInstances",
			&atomizer{
				electrons: make(chan instance),
				bonded:    nil,
				ctx:       context.Background(),
				cancel: context.CancelFunc(func() {

				}),
			},
			true,
		},
		{
			"InvalidAtomizerNilContext",
			&atomizer{
				electrons: make(chan instance),
				bonded:    make(chan instance),
				ctx:       nil,
				cancel: context.CancelFunc(func() {

				}),
			},
			true,
		},
		{
			"InvalidAtomizerNilCancel",
			&atomizer{
				electrons: make(chan instance),
				bonded:    make(chan instance),
				ctx:       context.Background(),
				cancel:    nil,
			},
			true,
		},
		{
			"InvalidAtomizerNilAtomizer",
			nil,
			true,
		},
	}

	for _, test := range tests {

		if ok := validator.IsValid(test.value); !test.err && !ok {
			t.Errorf("expected success for test [%s] but received failure", test.key)
		} else if test.err && ok {
			t.Errorf("expected error for test [%s] but received success", test.key)
		}
	}
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
	var mizer = &atomizer{
		electrons: make(chan instance),
		bonded:    make(chan instance),
		ctx:       context.Background(),
		cancel: context.CancelFunc(func() {

		}),
	}

	for n := 0; n < b.N; n++ {
		if !validator.IsValid(mizer) {
			b.Error("invalid atomizer, expected valid")
		}
	}
}
