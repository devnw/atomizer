package atomizer

import (
	"context"
	"testing"
	"time"

	"github.com/benjivesterby/atomizer/registration"

	"github.com/benjivesterby/validator"
)

type invalidconductor struct{}

type validcondcutor struct {
	id    string
	echan <-chan Electron
	valid bool
}

func (cond *validcondcutor) ID() string                                    { return cond.id }
func (cond *validcondcutor) Receive() <-chan Electron                      { return cond.echan }
func (cond *validcondcutor) Send(electron Electron) (result <-chan []byte) { return nil }
func (cond *validcondcutor) Validate() (valid bool)                        { return cond.valid && cond.echan != nil }
func (cond *validcondcutor) Complete(properties Properties)                {}

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
			&validcondcutor{"ValidTestValidConductor", make(<-chan Electron), true},
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

		mizer := Atomize(context.Background())
		if err := mizer.Exec(); test.err && err == nil {
			if !validator.IsValid(mizer) {
				t.Errorf("atomizer was expected to be valid but was returned invalid")
			}
		} else if !test.err && err != nil {
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
		value Conductor
		err   bool
	}{
		{
			"ValidTestEmptyConductor",
			&validcondcutor{"ValidTestEmptyConductor", make(<-chan Electron), true},
			false,
		},
		{
			"InvalidTestConductor",
			&validcondcutor{"InvalidTestConductor", make(<-chan Electron), false},
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

		func() {
			var ctx, cancel = context.WithTimeout(context.Background(), time.Second*5)
			defer cancel()

			// Create an instance of the atomizer to test the add conductor with
			mizer := Atomize(ctx)
			if errs, err := mizer.Errors(0); err == nil {
				if err = mizer.Exec(); err == nil {

					if validator.IsValid(mizer) {

						// Add the conductor
						if err = registration.Register(test.key, test.value); err == nil {

							select {
							case <-ctx.Done():
								// context for the atomizer was cancelled
							case aerr, ok := <-errs:
								if ok && aerr == nil && test.err {
									t.Errorf("expected error for test [%s] but received success", test.key)
								} else if ok && aerr != nil && !test.err {
									t.Errorf("expected success for test [%s] but received error [%s]", test.key, err)
								}
							}
						} else if !test.err {
							t.Errorf("expected success for test [%s] but received error [%s]", test.key, err)
						}
					} else {
						t.Errorf("expected the atomizer to be valid but it was invalid for ALL tests")
					}
				} else {
					t.Errorf("expected successful atomizer creation for test [%s] but received error while initializing atomizer [%s]", test.key, err.Error())
				}
			} else {
				t.Errorf("error while getting the errors channel from the atomizer")
			}
		}()

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
