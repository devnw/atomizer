package atomizer

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Pallinder/go-randomdata"
	"github.com/devnw/validator"
)

func printEvents(
	ctx context.Context,
	t *testing.T,
	events <-chan interface{},
) {

	for {
		select {
		case <-ctx.Done():
			return
		case e, ok := <-events:
			if ok {
				t.Log(e)
			} else {
				return
			}
		}
	}
}

// TODO: Add result nil checks on conductor response
func TestAtomizer_Exec(t *testing.T) {

	// Execute clean at beginning and end
	reset()
	defer reset()

	// Setup a cancellation context for the test
	ctx, cancel := _ctxT(nil, time.Second*30)
	defer cancel()

	events := make(chan interface{}, 1000)

	t.Log("setting up harness")
	conductor, err := harness(ctx, events)
	if err != nil {
		t.Error(err)
	}

	t.Log("setting up printing of atomizer events")
	go printEvents(ctx, t, events)

	t.Log("creating test electron")
	msg := randomdata.SillyName()
	e := newElectron(
		ID(returner{}),
		[]byte(
			fmt.Sprintf("{\"message\":\"%s\"}", msg),
		),
	)

	test := &tresult{
		result:   msg,
		electron: e,
	}

	var sent = time.Now()

	t.Log("sending electron through conductor")
	// Send the electron onto the conductor
	result, err := conductor.Send(ctx, test.electron)
	if err != nil {
		t.Error(err)
	}

	t.Log("read result from passthrough conductor")
	// Block until a result is returned from the instance
	select {
	case <-ctx.Done():
		t.Error("context closed, test failed")
		return
	case result, ok := <-result:
		if !ok {
			t.Error("result channel closed, test failed")
			return
		}

		if len(result.Errors) > 0 {

			for _, e := range result.Errors {
				// TODO: see if this works with the error lis
				t.Error("Errors returned from atom", e)
			}

			return
		}

		if len(result.Result) == 0 {
			t.Error("results length is not 1")
			return
		}

		res := string(result.Result)
		if res != test.result {
			t.Errorf("%s != %s", test.result, res)
			return
		}

		t.Logf(
			"EID [%s] | Time [%s] - MATCH",
			result.ElectronID,
			result.End.Sub(result.Start).String(),
		)
	}

	t.Logf(
		"Processing Time Through Atomizer %s\n",
		time.Since(sent).String(),
	)
}

func TestAtomizer_Exec_Returner(t *testing.T) {

	reset()
	defer reset()

	ctx, cancel := _ctx(nil)
	defer cancel()

	t.Log("Initializing Test Harness")

	if conductor, err := harness(ctx, nil); err == nil {

		t.Log("Harness Successfully Created")

		var sent = time.Now()

		wg := sync.WaitGroup{}

		// Spawn electrons
		wg.Add(1)
		go func() {
			defer wg.Done()

			tests := spawnReturner(50)

			t.Logf("[%v] tests loaded", len(tests))

			results := make(chan *Properties)

			go func() {

				for _, test := range tests {

					wg.Add(1)
					go func(test *tresult) {
						defer wg.Done()

						var err error

						var result <-chan *Properties
						// Send the electron onto the conductor
						if result, err = conductor.Send(ctx, test.electron); err == nil {

							select {
							case <-ctx.Done():
								return
							case result, ok := <-result:
								if ok {
									if len(result.Errors) == 0 {

										if validator.Valid(result.Result) {
											res := string(result.Result)
											if res != test.result {
												t.Errorf("%s != %s", test.result, res)
											}
										} else {
											t.Error("results length is not 1")
										}
									} else {
										t.Error("Error returned from atom", result.Errors)
									}
								} else {
									t.Error("result channel closed, test failed")
								}
							}
						}
					}(test)
				}

				wg.Wait()
				close(results)
			}()
		}()

		t.Logf("Processing Time Through Atomizer %s\n", time.Since(sent).String())

	} else {
		t.Errorf("error while executing harness | %s", err)
	}
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
			&validconductor{make(chan Electron), true},
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
			&validconductor{},
			true,
		},
	}

	ctx, cancel := _ctx(nil)
	defer cancel()

	for _, test := range tests {
		// Reset sync map for this test
		reset()

		// Store the test conductor
		if test.err || (!test.err && test.value != nil) {
			// TODO: should the error be ignored here?
			// Store invalid conductor
			_ = Register(test.value)
		}

		a := Atomize(ctx, nil)
		if err := a.Exec(); err != nil {
			t.Error(err)
		}

		if !validator.Valid(a) {
			t.Errorf("atomizer was expected to be valid but was returned invalid")
		}

		// Cleanup sync map for additional tests
		reset()
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
			&validconductor{make(chan Electron), true},
			false,
		},
		{
			"InvalidTestConductor",
			&validconductor{make(chan Electron), false},
			true,
		},
		{
			"InvalidTestNilConductor",
			nil,
			true,
		},
		{
			"InvalidTestInvalidElectronChan",
			&validconductor{},
			true,
		},
		{ // Empty key test
			"",
			&validconductor{},
			true,
		},
	}

	for _, test := range tests {
		// Reset sync map for this test
		reset()

		func() {
			var ctx, cancel = context.WithTimeout(context.Background(), time.Second*5)
			defer cancel()

			var err error

			// Create an instance of the atomizer to test the add conductor with
			events := make(chan interface{})
			a := Atomize(ctx, events)

			if err := a.Exec(); err != nil {
				t.Error(err)
			}

			if validator.Valid(a) {

				// Add the conductor
				if err = Register(test.value); err == nil {

					select {
					case <-ctx.Done():
						// context for the atomizer was cancelled
					case event, ok := <-events:
						if ok {
							if _, ok := event.(Error); ok {
								if ok && test.err {
									t.Errorf("expected error for test [%s] but received success", test.key)
								} else if ok && !test.err {
									t.Errorf("expected success for test [%s] but received error [%s]", test.key, err)
								}
							}
						} else {
							// TODO:
						}
					}
				} else if !test.err {
					t.Errorf("expected success for test [%s] but received error [%s]", test.key, err)
				}
			} else {
				t.Errorf("expected the atomizer to be valid but it was invalid for ALL tests")
			}
		}()

		// Cleanup sync map for additional tests
		reset()
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

		if ok := validator.Valid(test.value); !test.err && !ok {
			t.Errorf("expected success for test [%s] but received failure", test.key)
		} else if test.err && ok {
			t.Errorf("expected error for test [%s] but received success", test.key)
		}
	}
}

/********************************
*
*	BENCHMARKS
*
********************************/

func BenchmarkAtomizer_Exec_Single(b *testing.B) {

	reset()
	defer reset()

	ctx, cancel := _ctx(nil)
	defer cancel()

	if conductor, err := harness(ctx, nil); err == nil {

		// cleanup the benchmark timer to get correct measurements
		b.ResetTimer()

		for n := 0; n < b.N; n++ {
			e := newElectron("atomizer.noopatom", nil)

			var result <-chan *Properties
			// Send the electron onto the conductor
			if result, err = conductor.Send(ctx, e); err == nil {

				select {
				case <-ctx.Done():
					b.Error("context closed, test failed")
					return
				case result, ok := <-result:
					if ok {
						fmt.Printf("Step [%v]\n", n)
						if len(result.Errors) > 0 {
							b.Error("Error returned from atom", result.Errors)
						}
					} else {
						b.Error("result channel closed, test failed")
					}
				}

				// // Send the electron onto the conductor
				// resp := conductor.Send(ctx, e)

				// select {
				// case <-ctx.Done():
				// 	b.Error("context closed, test failed")
				// 	return
				// case result, ok := <-resp:
				// 	if ok {
				// 		if result != nil && result.Error() == nil {
				// 			// DO NOTHING
				// 		} else {
				// 			b.Error("invalid benchmark")
				// 		}
				// 	} else {
				// 		b.Error("result channel closed, test failed")
				// 	}
				// }
			}
		}
	} else {
		b.Errorf("test harness failed [%s]", err.Error())
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
	var a = &atomizer{
		electrons: make(chan instance),
		bonded:    make(chan instance),
		ctx:       context.Background(),
		cancel: context.CancelFunc(func() {

		}),
	}

	for n := 0; n < b.N; n++ {
		if !validator.Valid(a) {
			b.Error("invalid atomizer, expected valid")
		}
	}
}
