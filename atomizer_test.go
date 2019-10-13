package atomizer

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Pallinder/go-randomdata"
	"github.com/benjivesterby/validator"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type invalidconductor struct{}

type validcondcutor struct {
	id    string
	echan <-chan []byte
	valid bool
}

func (cond *validcondcutor) ID() string {
	return cond.id
}

func (cond *validcondcutor) Receive(ctx context.Context) <-chan []byte {
	return cond.echan
}

func (cond *validcondcutor) Send(ctx context.Context, electron Electron) (result <-chan Properties) {
	return nil
}

func (cond *validcondcutor) Validate() (valid bool) {
	return cond.valid && cond.echan != nil
}

func (cond *validcondcutor) Complete(ctx context.Context, properties Properties) (err error) {
	return err
}

// TODO: Move passthrough as a conductor implementation for in-node processing
type passthrough struct {
	input   chan []byte
	results sync.Map
}

func (pt *passthrough) ID() string { return "passthrough" }

func (pt *passthrough) Receive(ctx context.Context) <-chan []byte {
	return pt.input
}

func (pt *passthrough) Validate() (valid bool) { return pt.input != nil }

func (pt *passthrough) Complete(ctx context.Context, properties Properties) (err error) {
	if validator.IsValid(properties) {
		// for rabbit mq drop properties onto the /basepath/electronid message path
		if value, ok := pt.results.Load(properties.ElectronID()); ok {
			if value != nil {
				if resultChan, ok := value.(chan Properties); ok {
					defer close(resultChan)

					// Push the properties of the instance onto the channel
					select {
					case <-ctx.Done():
						return
					case resultChan <- properties:
					}
				} else {
					err = errors.New("unable to type assert electron properties channel")
				}
			} else {
				err = errors.Errorf("nil properties channel returned for electron [%s]", properties.ElectronID())
			}
		} else {
			err = errors.Errorf("unable to load properties channel from sync map for electron [%s]", properties.ElectronID())
		}
	}

	return err
}

func (pt *passthrough) Send(ctx context.Context, electron Electron) <-chan Properties {
	result := make(chan Properties)

	if validator.IsValid(electron) {
		go func(result chan Properties) {

			var e []byte
			var err error

			if e, err = json.Marshal(electron); err == nil {

				// Only kick off the electron for processing if there isn't already an
				// instance loaded in the system
				if _, loaded := pt.results.LoadOrStore(electron.ID(), result); !loaded {

					// Push the electron onto the input channel
					select {
					case <-ctx.Done():
						return
					case pt.input <- e:
						// setup a monitoring thread for /basepath/electronid
					}
				}
			}
		}(result)
	}

	return result
}

type printer struct{}

func (p *printer) ID() string { return "printer" }
func (p *printer) Process(ctx context.Context, electron Electron, outbound chan<- Electron) (result <-chan []byte) {

	if validator.IsValid(electron) {
		var payload printerdata
		var err error

		if err = json.Unmarshal(electron.Payload(), &payload); err == nil {

			fmt.Printf("message from electron [%s] is: %s\n", electron.ID(), payload.Message)
		} else {
			fmt.Println(err.Error())
		}
	}

	return result
}

type bench struct{}

func (b *bench) ID() string { return "bench" }
func (b *bench) Process(ctx context.Context, electron Electron, outbound chan<- Electron) (result <-chan []byte) {
	return result
}

type returner struct{}

func (b *returner) ID() string { return "returner" }
func (b *returner) Process(ctx context.Context, electron Electron, outbound chan<- Electron) <-chan []byte {
	result := make(chan []byte)

	go func(result chan<- []byte) {
		defer close(result)
		if validator.IsValid(electron) {
			var payload printerdata
			var err error

			if err = json.Unmarshal(electron.Payload(), &payload); err == nil {
				result <- []byte(payload.Message)
			} else {
				fmt.Println(err.Error())
			}
		}
	}(result)

	return result
}

type printerdata struct {
	Message string `json:"message"`
}

func newElectron(atomID string, payload []byte) (electron *ElectronBase, err error) {

	id := uuid.New()

	electron = &ElectronBase{
		ElectronID: id.String(),
		AtomID:     atomID,
		Load:       payload,
		Resp:       make(chan Properties),
	}

	return electron, err
}

// harness creates a valid atomizer that uses the passthrough conductor
func harness(ctx context.Context) (c Conductor, err error) {
	pass := &passthrough{
		input: make(chan []byte),
	}

	if validator.IsValid(pass) {

		// Register the conductor so it's picked up when the atomizer is initialized
		if err = Register(ctx, pass.ID(), pass); err == nil {

			// Register the atom so that it's there for processing requests from the conductor
			if err = Register(ctx, "printer", &printer{}); err == nil {

				// Register the benchmark atom for the benchmark tests
				if err = Register(ctx, "bench", &bench{}); err == nil {

					// Register the benchmark atom for the benchmark tests
					if err = Register(ctx, "returner", &returner{}); err == nil {

						// Initialize the atomizer
						mizer := Atomize(ctx)

						// Start the execution threads
						if err = mizer.Exec(); err == nil {
							c = pass
						}
					}
				}
			}
		}
	}

	return c, err
}

func TestAtomizer_Exec_Returner(t *testing.T) {

	Clean()

	// Setup a cancellation context for the test so that it has a limited time
	var ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	if conductor, err := harness(ctx); err == nil {
		msg := randomdata.SillyName()
		message := fmt.Sprintf("{\"message\":\"%s\"}", msg)
		payload := []byte(message)
		sent := time.Now()

		var e *ElectronBase
		if e, err = newElectron("returner", payload); err == nil {

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

						if len(result.Results()) > 0 {
							if string(result.Results()[0]) == msg {
								t.Logf("EID [%s] - MATCH", result.ElectronID())
							} else {
								t.Errorf("%s != %s", msg, result.Results()[0])
							}
						} else {
							t.Error("Expected Results")
						}

						t.Log(fmt.Sprintf("Electron Elapsed Processing Time %s\n", result.EndTime().Sub(result.StartTime()).String()))
					} else {
						// TODO: Errors returned from atom
						t.Errorf("Error returned from atom: [%s]\n", result.Errors()[0])
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

func BenchmarkAtomizer_Exec_Single(b *testing.B) {
	Clean()

	// Setup a cancellation context for the test so that it has a limited time
	var ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	if conductor, err := harness(ctx); err == nil {

		// cleanup the benchmark timer to get correct measurements
		b.ResetTimer()

		for n := 0; n < b.N; n++ {
			if e, err := newElectron("bench", nil); err == nil {
				// Send the electron onto the conductor
				resp := conductor.Send(ctx, e)

				select {
				case <-ctx.Done():
					b.Error("context closed, test failed")
					return
				case result, ok := <-resp:
					if ok {
						if result != nil && len(result.Errors()) == 0 {
							// DO NOTHING
						} else {
							b.Error("invalid benchmark")
						}
					} else {
						b.Error("result channel closed, test failed")
					}
				}
			} else {
				b.Errorf("electron creation failure [%s]", err.Error())
			}
		}
	} else {
		b.Errorf("test harness failed [%s]", err.Error())
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
