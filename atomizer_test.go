package atomizer

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Pallinder/go-randomdata"
	"github.com/benjivesterby/validator"
)

func TestAtomizer_Exec_Returner(t *testing.T) {

	Clean()

	// Setup a cancellation context for the test so that it has a limited time
	var ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	var tests []*tresult
	for i := 0; i < 100; i++ {
		msg := randomdata.SillyName()

		e, _ := newElectron("returner", []byte(fmt.Sprintf("{\"message\":\"%s\"}", msg)))

		tests = append(tests, &tresult{
			result:   msg,
			electron: e,
		})
	}

	if conductor, err := harness(ctx); err == nil {
		var sent = time.Now()

		wg := sync.WaitGroup{}

		// Spawn electrons
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, test := range tests {

				wg.Add(1)
				go func(test *tresult) {
					defer wg.Done()

					// Send the electron onto the conductor
					resp := conductor.Send(ctx, test.electron)

					select {
					case <-ctx.Done():
						t.Error("context closed, test failed")
						return
					case result, ok := <-resp:
						if ok {
							if len(result.Errors()) == 0 {

								if len(result.Results()) > 0 {
									res := string(result.Results())
									if res == test.result {
										t.Logf("EID [%s] | Time [%s] - MATCH", result.ElectronID(), result.EndTime().Sub(result.StartTime()).String())
									} else {
										t.Errorf("%s != %s", test.result, res)
									}
								} else {
									t.Error("results length is not 1")
								}
							} else {
								t.Errorf("Error returned from atom: [%s]\n", result.Errors())
							}
						} else {
							t.Error("result channel closed, test failed")
						}
					}
				}(test)
			}
		}()

		wg.Wait()
		t.Logf("Processing Time Through Atomizer %s\n", time.Now().Sub(sent).String())

	} else {
		t.Error("error while executing harness")
	}

	Clean()
}

func TestAtomizer_Exec(t *testing.T) {

	Clean()

	// Setup a cancellation context for the test so that it has a limited time
	var ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	if conductor, err := harness(ctx); err == nil {

		var payload = []byte(`{"message":"ruh roh"}`)
		var sent = time.Now()
		var e *ElectronBase

		if e, err = newElectron("printer", payload); err == nil {

			// Send the electron onto the conductor
			resp := conductor.Send(ctx, e)

			// Block until a result is returned from the instance
			select {
			case <-ctx.Done():
				t.Error("context closed, test failed")
				return
			case result, ok := <-resp:
				if ok {
					if len(result.Errors()) == 0 {
						t.Log(fmt.Sprintf("Electron Elapsed Processing Time %s\n", result.EndTime().Sub(result.StartTime()).String()))
					} else {
						// TODO: Errors returned from atom
					}
				} else {
					t.Error("result channel closed, test failed")
				}
			}

			fmt.Printf("Processing Time Through Atomizer %s\n", time.Now().Sub(sent).String())
		} else {
			t.Error(err)
		}

	} else {

	}

	Clean()
}

func TestAtomizer_Exec_Multiples(t *testing.T) {

	Clean()

	// Setup a cancellation context for the test so that it has a limited time
	var ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	if conductor, err := harness(ctx); err == nil {
		var sent = time.Now()

		wg := sync.WaitGroup{}

		// Spawn electrons
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 10; i++ {

				var e *ElectronBase
				var message = fmt.Sprintf("{\"message\":\"%s\"}", randomdata.SillyName())
				var payload = []byte(message)
				if e, err = newElectron("printer", payload); err == nil {

					// Send the electron onto the conductor
					resp := conductor.Send(ctx, e)

					wg.Add(1)
					go func(resp <-chan Properties) {
						defer wg.Done()

						// Block until a result is returned from the instance
						select {
						case <-ctx.Done():
							t.Error("context closed, test failed")
							return
						case result, ok := <-resp:
							if ok {
								if len(result.Errors()) == 0 {
									fmt.Printf("Electron Elapsed Processing Time %s\n", result.EndTime().Sub(result.StartTime()).String())
								} else {
									// TODO: Errors returned from atom
								}
							} else {
								t.Error("result channel closed, test failed")
							}
						}
					}(resp)
				} else {
					t.Error(err)
				}

				time.Sleep(time.Millisecond * 50)
			}
		}()

		wg.Wait()
		fmt.Printf("Processing Time Through Atomizer %s\n", time.Now().Sub(sent).String())

	} else {

	}

	Clean()
}

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
			&validcondcutor{"ValidTestValidConductor", make(<-chan []byte), true},
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
		Clean()

		// Store the test conductor
		if test.err || (!test.err && test.value != nil) {
			// Store invalid conductor
			Register(nil, test.key, test.value)
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
		Clean()
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
			&validcondcutor{"ValidTestEmptyConductor", make(<-chan []byte), true},
			false,
		},
		{
			"InvalidTestConductor",
			&validcondcutor{"InvalidTestConductor", make(<-chan []byte), false},
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
		Clean()

		func() {
			var ctx, cancel = context.WithTimeout(context.Background(), time.Second*5)
			defer cancel()

			// Create an instance of the atomizer to test the add conductor with
			mizer := Atomize(ctx)
			if errs, err := mizer.Errors(0); err == nil {
				if err = mizer.Exec(); err == nil {

					if validator.IsValid(mizer) {

						// Add the conductor
						if err = Register(ctx, test.key, test.value); err == nil {

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
		Clean()
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
