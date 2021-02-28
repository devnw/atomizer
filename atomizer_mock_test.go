package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"devnw.com/alog"
	"devnw.com/validator"
	"github.com/Pallinder/go-randomdata"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type tresult struct {
	result   string
	electron Electron
	// err      bool
	// panic    bool
}

var noopelectron = Electron{
	SenderID: "empty",
	ID:       "empty",
	AtomID:   "empty",
}

var noopinvalidelectron = Electron{}

type invalidconductor struct{}

type noopconductor struct{}

func (*noopconductor) Receive(ctx context.Context) <-chan Electron {
	return nil
}

func (*noopconductor) Send(
	ctx context.Context,
	electron Electron,
) (<-chan Properties, error) {
	return nil, nil
}

func (*noopconductor) Close() {}

func (*noopconductor) Complete(
	ctx context.Context,
	properties Properties,
) error {
	return nil
}

type noopatom struct{}

func (*noopatom) Process(
	ctx context.Context,
	conductor Conductor,
	electron Electron,
) ([]byte, error) {
	return nil, nil
}

type panicatom struct{}

func (*panicatom) Process(
	ctx context.Context,
	conductor Conductor,
	electron Electron,
) ([]byte, error) {
	panic("test panic")
}

type invalidatom struct{}

func (*invalidatom) Process(
	ctx context.Context,
	conductor Conductor,
	electron Electron,
) ([]byte, error) {
	return nil, nil
}

func (*invalidatom) Validate() bool {
	return false
}

type validconductor struct {
	echan chan Electron
	valid bool
}

func (cond *validconductor) Receive(ctx context.Context) <-chan Electron {
	return cond.echan
}

func (cond *validconductor) Send(ctx context.Context, electron Electron) (response <-chan Properties, err error) {

	return response, err
}

func (cond *validconductor) Validate() (valid bool) {
	return cond.valid && cond.echan != nil
}

func (cond *validconductor) Complete(ctx context.Context, properties Properties) (err error) {
	return err
}

func (cond *validconductor) Close() {}

// TODO: Move passthrough as a conductor implementation for in-node processing
type passthrough struct {
	input   chan Electron
	results sync.Map
}

func (pt *passthrough) Receive(ctx context.Context) <-chan Electron {
	return pt.input
}

func (pt *passthrough) Validate() bool { return pt.input != nil }

func (pt *passthrough) Complete(ctx context.Context, properties Properties) error {
	if !validator.Valid(properties) {
		return errors.Errorf("invalid properties returned for electron [%s]", properties.ElectronID)
	}

	// for rabbit mq drop properties onto the /basepath/electronid message path
	value, ok := pt.results.Load(properties.ElectronID)
	if !ok {
		return errors.Errorf("unable to load properties channel from sync map for electron [%s]", properties.ElectronID)
	}

	if value == nil {
		return errors.Errorf("nil properties channel returned for electron [%s]", properties.ElectronID)
	}

	resultChan, ok := value.(chan Properties)
	if !ok {
		return errors.New("unable to type assert electron properties channel")
	}

	defer close(resultChan)

	// Push the properties of the instance onto the channel
	select {
	case <-ctx.Done():
		return nil
	case resultChan <- properties:
	}
	return nil
}

func (pt *passthrough) Send(ctx context.Context, electron Electron) (<-chan Properties, error) {
	var err error
	result := make(chan Properties)

	go func(result chan Properties) {

		// Only kick off the electron for processing if there isn't already an
		// instance loaded in the system
		if _, loaded := pt.results.LoadOrStore(electron.ID, result); !loaded {

			// Push the electron onto the input channel
			select {
			case <-ctx.Done():
				return
			case pt.input <- electron:
				// setup a monitoring thread for /basepath/electronid
			}
		} else {
			defer close(result)
			p := Properties{}
			alog.Errorf(nil, "duplicate electron registration for EID [%s]", electron.ID)

			result <- p
		}
	}(result)

	return result, err
}

func (pt *passthrough) Close() {}

type printer struct{}

type state struct{ ID string }

func (s *state) Process(ctx context.Context, conductor Conductor, electron Electron) (result []byte, err error) {
	return []byte(s.ID), nil
}

func (p *printer) Process(ctx context.Context, conductor Conductor, electron Electron) (result []byte, err error) {

	if validator.Valid(electron) {
		var payload printerdata

		if err = json.Unmarshal(electron.Payload, &payload); err == nil {

			fmt.Printf("message from electron [%s] is: %s\n", electron.ID, payload.Message)
		}
	}

	return result, err
}

type returner struct{}

func (b *returner) Process(ctx context.Context, conductor Conductor, electron Electron) (result []byte, err error) {

	if !validator.Valid(electron) {
		return nil, errors.New("invalid electron")
	}

	var payload = &printerdata{}
	err = json.Unmarshal(electron.Payload, payload)
	if err != nil {
		return nil, err
	}

	result = []byte(payload.Message)
	alog.Println("returning payload")

	return result, err
}

func spawnReturner(size int) (tests []*tresult) {

	tests = make([]*tresult, 0, size)

	for i := 0; i < size; i++ {
		msg := randomdata.SillyName()

		e := newElectron(
			ID(returner{}),
			[]byte(fmt.Sprintf("{\"message\":\"%s\"}", msg)),
		)

		tests = append(tests, &tresult{
			result:   msg,
			electron: e,
		})
	}

	return tests
}

type printerdata struct {
	Message string `json:"message"`
}

func newElectron(atomID string, payload []byte) Electron {
	return Electron{
		SenderID: uuid.New().String(),
		ID:       uuid.New().String(),
		AtomID:   atomID,
		Payload:  payload,
	}
}

// harness creates a valid atomizer that uses the passthrough conductor
func harness(
	ctx context.Context,
	events chan interface{},
) (Conductor, error) {

	pass := &passthrough{
		input: make(chan Electron, 1),
	}

	// Register the conductor so it's picked up
	// when the atomizer is initialized
	if err := Register(pass); err != nil {
		return nil, err
	}

	// Test Atom registrations

	if err := Register(&printer{}); err != nil {
		return nil, err
	}

	if err := Register(&noopatom{}); err != nil {
		return nil, err
	}

	if err := Register(&returner{}); err != nil {
		return nil, err
	}

	// Initialize the atomizer
	mizer, err := Atomize(ctx, events)
	if err != nil {
		return nil, fmt.Errorf("error creating atomizer | %s", err)
	}

	a, _ := mizer.(*atomizer)

	// Start the execution threads
	return pass, a.Exec()
}
