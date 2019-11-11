package atomizer

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/benjivesterby/validator"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type tresult struct {
	result   string
	electron *ElectronBase
	res      chan *Properties
	err      bool
	panic    bool
}

type invalidconductor struct{}

type validconductor struct {
	id    string
	echan <-chan []byte
	valid bool
}

func (cond *validconductor) ID() string {
	return cond.id
}

func (cond *validconductor) Receive(ctx context.Context) <-chan []byte {
	return cond.echan
}

func (cond *validconductor) Send(ctx context.Context, electron Electron) (err error) {
	return err
}

func (cond *validconductor) Validate() (valid bool) {
	return cond.valid && cond.echan != nil
}

func (cond *validconductor) Complete(ctx context.Context, properties *Properties) (err error) {
	return err
}

func (cond *validconductor) Close() {}

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

func (pt *passthrough) Complete(ctx context.Context, properties *Properties) (err error) {
	if validator.IsValid(properties) {
		// for rabbit mq drop properties onto the /basepath/electronid message path
		if value, ok := pt.results.Load(properties.ElectronID); ok {
			if value != nil {
				if resultChan, ok := value.(chan *Properties); ok {
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
				err = errors.Errorf("nil properties channel returned for electron [%s]", properties.ElectronID)
			}
		} else {
			err = errors.Errorf("unable to load properties channel from sync map for electron [%s]", properties.ElectronID)
		}
	} else {
		err = errors.Errorf("invalid properties returned for electron [%s]", properties.ElectronID)
	}

	return err
}

func (pt *passthrough) Send(ctx context.Context, electron Electron) (err error) {
	result := make(chan *Properties)

	if validator.IsValid(electron) {
		go func(result chan *Properties) {

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
				} else {
					defer close(result)
					p := &Properties{}
					p.Error = errors.Errorf("duplicate electron registration for EID [%s]", electron.ID())

					result <- p
				}
			}
		}(result)
	}

	return err
}

func (pt *passthrough) Close() {
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

func newElectron(atomID string, payload []byte) (electron *ElectronBase, res chan (*Properties), err error) {

	id := uuid.New()
	res = make(chan *Properties)

	electron = &ElectronBase{
		ElectronID: id.String(),
		AtomID:     atomID,
		Load:       payload,
		Resp:       res,
	}

	return electron, res, err
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
