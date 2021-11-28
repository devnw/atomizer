package engine

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Pallinder/go-randomdata"
	"github.com/google/uuid"
	"go.devnw.com/event"
	"go.devnw.com/validator"
)

func printEvents(
	ctx context.Context,
	t *testing.T,
	events event.EventStream,
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

func TestAtomizer_Atomize_Register_Fail(t *testing.T) {
	// Register invalid atom
	_, err := Atomize(
		context.TODO(),
		nil,
		&struct{}{},
	)
	if err == nil {
		t.Fatalf("expected error | %s", err)
	}
}

func TestAtomizer_Exec(t *testing.T) {
	d := time.Second * 30
	// Setup a cancellation context for the test
	ctx, cancel := _ctxT(context.TODO(), &d)
	defer cancel()

	// Execute clean at beginning and end
	reset(ctx, t)
	t.Cleanup(func() {
		reset(context.TODO(), t)
	})

	t.Log("setting up harness")
	conductor, events, err := harness(ctx, 1000)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("setting up printing of atomizer events")
	go printEvents(ctx, t, events)

	t.Log("creating test electron")
	msg := randomdata.SillyName()
	e := newElectron(
		ID(returner{}),
		[]byte(
			fmt.Sprintf("{\"message\":%q}", msg),
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
		t.Fatal(err)
	}

	t.Log("read result from passthrough conductor")
	check(ctx, t, test, e, result)

	t.Logf(
		"Processing Time Through Atomizer %s\n",
		time.Since(sent).String(),
	)
}

func check(
	ctx context.Context,
	t *testing.T,
	test *tresult,
	e *Electron,
	result <-chan *Properties,
) {
	// Block until a result is returned from the instance
	select {
	case <-ctx.Done():
		t.Fatal("context closed, test failed")
		return
	case result, ok := <-result:
		if !ok {
			t.Fatal("result channel closed, test failed")
		}

		if result.Error != nil {
			t.Fatal("Errors returned from atom", e)
		}

		if len(result.Result) == 0 {
			t.Fatal("results length is not 1")
		}

		res := string(result.Result)
		if res != test.result {
			t.Fatalf("%s != %s", test.result, res)
		}

		t.Logf(
			"EID [%s] | Time [%s] - MATCH",
			result.ElectronID,
			result.End.Sub(result.Start).String(),
		)
	}
}

func TestAtomizer_initReg_Exec(t *testing.T) {
	d := time.Second * 30
	// Setup a cancellation context for the test
	ctx, cancel := _ctxT(context.TODO(), &d)
	defer cancel()

	// Execute clean at beginning and end
	reset(ctx, t)
	t.Cleanup(func() {
		reset(context.TODO(), t)
	})

	t.Log("setting up harness")
	conductor, events, err := harness(ctx, 1000)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("setting up printing of atomizer events")
	go printEvents(ctx, t, events)

	t.Log("creating test electron")
	msg := randomdata.SillyName()
	e := newElectron(
		ID(returner{}),
		[]byte(
			fmt.Sprintf("{\"message\":%q}", msg),
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
		t.Fatal(err)
	}

	t.Log("read result from passthrough conductor")
	check(ctx, t, test, e, result)

	t.Logf(
		"Processing Time Through Atomizer %s\n",
		time.Since(sent).String(),
	)
}

func TestAtomizer_Copy_State(t *testing.T) {
	d := time.Second * 30
	// Setup a cancellation context for the test
	ctx, cancel := _ctxT(context.TODO(), &d)
	defer cancel()

	// Execute clean at beginning and end
	reset(ctx, t)
	t.Cleanup(func() {
		reset(context.TODO(), t)
	})

	stateid := uuid.New().String()

	t.Log("setting up harness")
	conductor, events, err := harness(ctx, 1000, &state{stateid})
	if err != nil {
		t.Fatal(err)
	}

	t.Log("setting up printing of atomizer events")
	go printEvents(ctx, t, events)

	t.Log("creating test electron")
	e := &Electron{
		SenderID:  uuid.New().String(),
		ID:        uuid.New().String(),
		AtomID:    ID(state{}),
		CopyState: true,
	}

	var sent = time.Now()

	t.Log("sending electron through conductor")
	// Send the electron onto the conductor
	result, err := conductor.Send(ctx, e)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("read result from passthrough conductor")
	// Block until a result is returned from the instance
	select {
	case <-ctx.Done():
		t.Fatal("context closed, test failed")
		return
	case result, ok := <-result:
		if !ok {
			t.Fatal("result channel closed, test failed")
		}

		if result.Error != nil {
			t.Fatal("Errors returned from atom", e)
		}

		res := string(result.Result)
		if res != stateid {
			t.Fatalf("[%s] != [%s]", res, stateid)
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

func TestAtomizer_Copy_State_Disabled(t *testing.T) {
	d := time.Second * 30
	// Setup a cancellation context for the test
	ctx, cancel := _ctxT(context.TODO(), &d)
	defer cancel()

	// Execute clean at beginning and end
	reset(ctx, t)
	t.Cleanup(func() {
		reset(context.TODO(), t)
	})

	stateid := uuid.New().String()

	t.Log("setting up harness")
	conductor, events, err := harness(ctx, 1000, &state{stateid})
	if err != nil {
		t.Fatal(err)
	}

	t.Log("setting up printing of atomizer events")
	go printEvents(ctx, t, events)

	t.Log("creating test electron")
	e := &Electron{
		SenderID:  uuid.New().String(),
		ID:        uuid.New().String(),
		AtomID:    ID(state{}),
		CopyState: false,
	}

	var sent = time.Now()

	t.Log("sending electron through conductor")
	// Send the electron onto the conductor
	result, err := conductor.Send(ctx, e)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("read result from passthrough conductor")
	// Block until a result is returned from the instance
	select {
	case <-ctx.Done():
		t.Fatal("context closed, test failed")
		return
	case result, ok := <-result:
		if !ok {
			t.Fatal("result channel closed, test failed")
		}

		if result.Error != nil {
			t.Fatal("Errors returned from atom", e)
		}

		res := string(result.Result)
		if res == stateid {
			t.Fatalf("Expected mismatch [%s] == [%s]", res, stateid)
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
	ctx, cancel := _ctx(context.TODO())
	defer cancel()

	reset(ctx, t)
	t.Cleanup(func() {
		reset(context.TODO(), t)
	})

	t.Log("Initializing Test Harness")

	conductor, _, err := harness(ctx, -1)
	if err != nil {
		t.Fatalf("error while executing harness | %s", err)
	}

	t.Log("Harness Successfully Created")

	var sent = time.Now()
	wg := sync.WaitGroup{}

	tests := spawnReturner(50)

	t.Logf("[%v] tests loaded", len(tests))

	results := make(chan Properties)

	go func() {
		defer cancel()

		for _, test := range tests {
			wg.Add(1)
			go func(test *tresult) {
				defer wg.Done()

				sentAndEval(
					ctx,
					t,
					conductor,
					test,
				)
			}(test)
		}

		wg.Wait()
		close(results)
	}()

	<-ctx.Done()
	t.Logf(
		"Processing Time Through Atomizer %s\n",
		time.Since(sent).String(),
	)
}

func sentAndEval(
	ctx context.Context,
	t *testing.T,
	c Conductor,
	test *tresult,
) {
	// Send the electron onto the conductor
	result, err := c.Send(ctx, test.electron)
	if err != nil {
		t.Fatalf("Error sending electron %s", test.electron.ID)
	}

	select {
	case <-ctx.Done():
		return
	case result, ok := <-result:
		if !ok {
			t.Fatal("result channel closed, test failed")
		}

		if result.Error != nil {
			t.Fatal(
				"Error returned from atom",
				result.Error,
			)
		}

		if !validator.Valid(result.Result) {
			t.Fatal("results length is not 1")
		}

		res := string(result.Result)
		if res != test.result {
			t.Fatalf("%s != %s", test.result, res)
		}
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
			&validconductor{make(chan *Electron), true},
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

	ctx, cancel := _ctx(context.TODO())
	defer cancel()

	for _, test := range tests {
		t.Run(test.key, func(t *testing.T) {
			reset(ctx, t)
			t.Cleanup(func() {
				reset(context.TODO(), t)
			})

			// Store the test conductor
			if test.err || (!test.err && test.value != nil) {
				// TODO: should the error be ignored here?
				// Store invalid conductor
				_ = Register(test.value)
			}

			a, err := Atomize(ctx)
			if err != nil {
				t.Fatal(err)
			}

			err = a.Exec()
			if err != nil {
				t.Fatal(err)
			}

			if !validator.Valid(a) {
				t.Fatalf("atomizer was expected to be valid but was returned invalid")
			}
		})
	}
}

func TestAtomizer_AddConductor(t *testing.T) {
	ctx, cancel := _ctx(context.TODO())
	defer cancel()

	tests := []struct {
		key   string
		value interface{}
		err   bool
	}{
		{
			"ValidTestEmptyConductor",
			&validconductor{make(chan *Electron), true},
			false,
		},
		{
			"InvalidTestConductor",
			&validconductor{make(chan *Electron), false},
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

	for _, test := range tests {
		t.Run(test.key, func(t *testing.T) {
			// Reset sync map for this test
			reset(ctx, t)
			t.Cleanup(func() {
				reset(context.TODO(), t)
			})

			a, err := Atomize(ctx)
			if err != nil {
				t.Fatal(err)
			}

			err = a.Exec()
			if err != nil {
				t.Fatal(err)
			}

			// Add the conductor
			err = a.Register(test.value)
			if err != nil && !test.err {
				t.Fatalf("expected success, received error")
			}
		})
	}
}

func TestAtomizer_register_Errs(t *testing.T) {
	ctx, cancel := _ctx(context.TODO())
	defer cancel()

	tests := []struct {
		key   string
		a     *atomizer
		value interface{}
	}{
		{
			"invalid conductor test",
			&atomizer{
				ctx:       ctx,
				publisher: event.NewPublisher(ctx),
			},
			&validconductor{},
		},
		{
			"Invalid Struct Type",
			&atomizer{
				ctx:       ctx,
				publisher: event.NewPublisher(ctx),
			},
			&struct{}{},
		},
	}

	for _, test := range tests {
		t.Run(test.key, func(t *testing.T) {
			errors := test.a.Errors(1)

			test.a.register(test.value)

			out, ok := <-errors
			if !ok {
				t.Fatal("channel closed")
			}

			t.Log(out)
		})
	}
}

func TestAtomizer_Register_Errs(t *testing.T) {
	ctx, cancel := _ctx(context.TODO())
	cancel()

	tests := []struct {
		key   string
		a     *atomizer
		value interface{}
	}{
		{
			"panic test, nil channels",
			&atomizer{publisher: event.NewPublisher(ctx)},
			&validconductor{make(chan *Electron), true},
		},
		{
			"close context test",
			&atomizer{ctx: ctx, publisher: event.NewPublisher(ctx)},
			&validconductor{make(chan *Electron), true},
		},
		{
			"Invalid Struct Type",
			&atomizer{publisher: event.NewPublisher(ctx)},
			&struct{}{},
		},
	}

	for _, test := range tests {
		t.Run(test.key, func(t *testing.T) {
			// Add the conductor
			err := test.a.Register(test.value)
			if err == nil {
				t.Fatalf("expected error, received success")
			}
		})
	}
}

func TestAtomizer_receive_panic(t *testing.T) {
	a := &atomizer{
		ctx:       context.Background(),
		publisher: event.NewPublisher(context.Background()),
	}

	errors := a.Errors(1)

	defer func() {
		r := recover()
		if r != nil {
			t.Fatal("unexpected panic")
		}
	}()

	a.receive()

	err, ok := <-errors
	if !ok || err == nil {
		t.Fatal("channel closed")
	}
}

func TestAtomizer_receive_nilreg(t *testing.T) {
	a := &atomizer{
		ctx:       context.Background(),
		publisher: event.NewPublisher(context.Background()),
	}

	errors := a.Errors(1)

	a.receive()

	_, ok := <-errors
	if !ok {
		t.Fatal("channel closed")
	}
}

func TestAtomizer_receive_closedReg(t *testing.T) {
	a := &atomizer{
		ctx:           context.Background(),
		registrations: make(chan interface{}),
		publisher:     event.NewPublisher(context.Background()),
	}

	close(a.registrations)

	errors := a.Errors(1)

	a.receive()

	_, ok := <-errors
	if !ok {
		t.Fatal("channel closed")
	}
}

func TestAtomizer_receiveAtom_invalid(t *testing.T) {
	a := &atomizer{publisher: event.NewPublisher(context.Background())}

	err := a.receiveAtom(&invalidatom{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAtomizer_conduct_closedreceiver(t *testing.T) {
	c := &validconductor{echan: make(chan *Electron)}
	close(c.echan)

	a := &atomizer{
		ctx:       context.Background(),
		publisher: event.NewPublisher(context.Background()),
	}

	errors := a.Errors(1)

	a.conduct(context.Background(), c)

	_, ok := <-errors
	if !ok {
		t.Fatal("channel closed")
	}
}

func TestAtomizer_conduct_panic(t *testing.T) {
	c := &validconductor{echan: make(chan *Electron)}
	close(c.echan)

	a := &atomizer{publisher: event.NewPublisher(context.Background())}

	errors := a.Errors(2)

	defer func() {
		r := recover()
		if r != nil {
			t.Fatal("unexpected panic")
		}
	}()

	a.conduct(context.Background(), c)

	err, ok := <-errors
	if !ok || err == nil {
		t.Fatal("expected error")
	}
}

func TestAtomizer_conduct_invalidE(t *testing.T) {
	c := &passthrough{input: make(chan *Electron)}
	a := &atomizer{
		ctx:       context.Background(),
		publisher: event.NewPublisher(context.Background()),
	}
	go a.conduct(context.Background(), c)

	t.Log("sending")
	results, err := c.Send(context.Background(), noopinvalidelectron)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("waiting on results")
	res, ok := <-results
	if !ok {
		t.Fatal("unexpected closed channel")
	}

	if res.Error == nil {
		t.Fatal("expected error result")
	}
}

func TestAtomizer_split_closedEchan(t *testing.T) {
	a := &atomizer{
		ctx:       context.Background(),
		publisher: event.NewPublisher(context.Background()),
	}

	errors := a.Errors(1)

	echan := make(chan instance)
	close(echan)

	a._split(nil, echan)

	_, ok := <-errors
	if !ok {
		t.Fatal("channel closed")
	}
}

func TestAtomizer_Wait(t *testing.T) {
	ctx, cancel := _ctx(context.TODO())
	a := &atomizer{
		ctx:       ctx,
		cancel:    cancel,
		publisher: event.NewPublisher(ctx),
	}

	cancel()
	a.Wait()
}

func TestAtomizer_distribute_closedEchan(t *testing.T) {
	ctx, cancel := _ctx(context.TODO())
	a := &atomizer{
		ctx:       ctx,
		cancel:    cancel,
		electrons: make(chan instance),
		publisher: event.NewPublisher(ctx),
	}
	close(a.electrons)

	errors := a.Errors(1)

	a.distribute()

	_, ok := <-errors
	if !ok {
		t.Fatal("channel closed")
	}
}

func TestAtomizer_exec_ERR(t *testing.T) {
	ctx, cancel := _ctx(context.TODO())
	a := &atomizer{
		ctx:       ctx,
		cancel:    cancel,
		publisher: event.NewPublisher(ctx),
	}

	errors := a.Errors(1)

	i := instance{ctx: ctx, cancel: cancel}

	a.exec(i, nil)

	_, ok := <-errors
	if !ok {
		t.Fatal("channel closed")
	}
}

func unexpHarness(t *testing.T) (context.Context, context.CancelFunc, *atomizer) {
	ctx, cancel := _ctx(context.TODO())
	mizer, err := Atomize(ctx)
	if err != nil {
		t.Fatal(err)
	}

	a, ok := mizer.(*atomizer)
	if !ok {
		t.Fatal("unable to cast atomizer")
	}

	return ctx, cancel, a
}

func TestAtomizer_distribute_unregistered(t *testing.T) {
	ctx, cancel, a := unexpHarness(t)

	errors := a.Errors(1)

	i := instance{
		ctx:      ctx,
		cancel:   cancel,
		electron: &Electron{AtomID: "nopey.nope"},
	}

	go a.distribute()
	go func() { a.electrons <- i }()

	_, ok := <-errors
	if !ok {
		t.Fatal("channel closed")
	}
}

func TestAtomizer_exec_inst_err(t *testing.T) {
	ctx, cancel, a := unexpHarness(t)

	errors := a.Errors(1)

	i := instance{
		ctx:       ctx,
		cancel:    cancel,
		electron:  noopelectron,
		conductor: &noopconductor{},
	}

	go a.exec(i, &panicatom{})

	_, ok := <-errors
	if !ok {
		t.Fatal("channel closed")
	}
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
				publisher: event.NewPublisher(context.Background()),
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
		t.Run(test.key, func(t *testing.T) {
			ok := validator.Valid(test.value)
			if !ok && !test.err {
				t.Fatalf("expected success, got error")
			}

			if ok && test.err {
				t.Fatalf("expected error")
			}
		})
	}
}

// ********************************
// BENCHMARKS
// ********************************

func BenchmarkAtomizer_Exec_Single(b *testing.B) {
	resetB()
	b.Cleanup(func() {
		resetB()
	})

	ctx, cancel := _ctx(context.TODO())
	defer cancel()

	conductor, _, err := harness(ctx, -1)
	if err != nil {
		b.Fatalf("test harness failed [%s]", err.Error())
	}

	// cleanup the benchmark timer to get correct measurements
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		e := newElectron(ID(noopatom{}), nil)

		// Send the electron onto the conductor
		result, err := conductor.Send(ctx, e)
		if err != nil {
			b.Fatal(err)
		}

		select {
		case <-ctx.Done():
			b.Fatal("context closed, test failed")
		case result, ok := <-result:
			if !ok {
				b.Fatal("result channel closed, test failed")
			}

			if result.Error != nil {
				b.Fatal("Error returned from atom", result.Error)
			}
		}
	}
}
