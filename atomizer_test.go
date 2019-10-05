package atomizer

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"

	"github.com/benjivesterby/validator"
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
					// TODO:
				}
			} else {
				// TODO:
			}
		} else {
			// TODO:
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
	hk :=
		`                              .ZMMMMMN.     ..:+=.
	,NMNZ~..             .MMZZZZZMM,.ZMMMMMMMM?.
  .+MM+=ZMMMMN......,++I7MMZZZZZZZMMMM~......+M~
  ~M8.......~MMMMMMMMMMMMMZZZZZZZZZM8.........MM
  MM...................~MZZZZZMMMMZMM8,.......MM.
 .MM...................NMZZZZZMZZMM8ZZMMMNMMMMMM.
 .MN...................M8ZZZZZMMMMZZZZZZMMZZZZZMM,
 .MM...................IMZZZZZZZMZZZZZZZMMZZZZZZMM
  MMMM..................NMM8Z8MMM8ZZZZZZMMMMZZZZMM..
  ,MM.....................+ZZ?,.?M8ZZZ8MMZMMZZZZMZ,,
 .MM..............................+MMMMZZ8ZZZZZMN.MMZ+~,
 +M7.................................~MZZZZZZMMMM.MMMMMM8I~,
.MN...................................:MMMMMMN.,MIMMMMMMMMMN
?M~.............................................MMMMMMMMMMMM
NM..............................................MM.?ZMMMMMMM
MM...........................................~NMMMMMMMMN8MMM
..:+IMMZI:........................................?I=MM,......~78
NN+,:MM,........~MMN.....................MMM+........MM
,MI........ZMMM.....................MMMZ.......,MM
.MM:,,......MN..........ZDZ:........+MZ........7MMMMMM
+MMMMM,..................MIIIM+..................MM
..  IMZ..................8MDMM................ZIMM,
   ,MMMM=...................................,MMMMN
 .IMMMMM...................................~MM,..7MM,
.NM?.  ,NMM7..............................ZMM+
...      .,MMMMNI.....................IMMMM,
	   .:NMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMM,
	 .NMM+ZZZMMZZZMM~~,,,,++??,M8ZZZMMZZ~7MMN,
   .MMM.ZZ$,MMZZZZZM7~,,,,,,,,MMZZZZZMM.ZZ+.MMN
  ~MM+MM~.ZMMZZZZZZZMMD:,,,~MM8ZZZZZZZMM+.ZMM+MM
 ,M8...:MMMMZZMMMMZZZZZ8MMM8ZZZZZMMM8ZZMMMM:...8M~
,MZ......+MZZ8+..~MZZZZZZZZZZZZZM~..M8ZZM7......MM
+M.......MMZZZM~,M8ZZZZZZZZZZZZZMM.~MZZZMM......?M
,M~.....ZMZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ8M......MM
.NM~....MMZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZMI....+MM
  8MMMMMMMZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZMMMMMMM,
	...:M8ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ8M+,,
	   .M8ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ8M:M8
	   ,M8ZZZZZZZZZZZZZZZMZZZZZZZZZZZZZZZZM,MN
	   .MMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMM:M7
	   ,MI..............7M...............+M:N,
	   .MD..............7M...............ZM.+
		MM..............?M,.............,MZ.
		,MM.............MM7.............MM.,
		 .NMMMMD?7IIZNMMM7MMMNZII7ZZMMMM7.+
			..IZMNMMZ7,.....+7NMMMMZ+..MM8,
								  ..?MMMN+
								  .:NMMM8
								 ..IMMMM+
								..ZMMMM~
								.,NMMMD.
								..ZMMMM~     -Brandon Adamson
								.,NMMMD

------------------------------------------------
Thank you for visiting https://asciiart.website/
This ASCII pic can be found at
https://asciiart.website/index.php?art=logos%20and%20insignias/hello%20kitty
`
	fmt.Println(hk)
	return result
}

type printerdata struct {
	Message string `json:"message"`
}

func TestAtomizer(t *testing.T) {

	Clean()

	var err error
	pass := &passthrough{
		input: make(chan []byte),
	}

	// Setup a cancellation context for the test so that it has a limited time
	var ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	if validator.IsValid(pass) {

		// Register the conductor so it's picked up when the atomizer is initialized
		if err = Register(pass.ID(), pass); err == nil {

			// Register the atom so that it's there for processing requests from the conductor
			if err = Register("printer", &printer{}); err == nil {

				var payload = []byte("{\"message\":\"ruh roh\"}")

				// Initialize the atomizer
				mizer := Atomize(ctx)

				// Start the execution threads
				if err = mizer.Exec(); err == nil {

					// Send the electron onto the conductor
					resp := pass.Send(ctx, &ElectronBase{
						ElectronID: "Yabba Dabba Doo!",
						AtomID:     "printer",
						Load:       payload,
						Resp:       make(chan Properties),
					})

					// Block until a result is returned from the instance
					select {
					case <-ctx.Done():
						t.Error("context closed, test failed")
					case result, ok := <-resp:
						if ok {
							spew.Dump(result)

							fmt.Printf("Elapsed Time %s\n", result.EndTime().Sub(result.StartTime()).String())
						} else {
							t.Error("result channel closed, test failed")
						}
					}

				} else {
					t.Error(err.Error())
				}
			} else {
				t.Error(err.Error())
			}
		} else {
			t.Error(err.Error())
		}
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
			Register(test.key, test.value)
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
						if err = Register(test.key, test.value); err == nil {

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
