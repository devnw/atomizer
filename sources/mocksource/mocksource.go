package mocksource

import (
	"context"
	"strconv"
	"time"
)

// TODO: Setup quarks and callbacks

func GetMockSource(atoms int, delay *time.Duration, process func(ctx context.Context, payload []byte) (err error)) (source mocksource) {
	return mocksource{
		atoms,
		delay,
		process,
	}
}

type mocksource struct {
	atoms int
	delay *time.Duration
	process func(ctx context.Context, payload []byte) (err error)
}

func (this mocksource) GetAtoms() <- chan mockatom {
	var atomStream = make(chan mockatom)

	// Push off the atom stream to a go routine and loop through the expected
	// atoms for mocking
	go func(aStream chan <- mockatom) {
		for i := 0; i < this.atoms; i++ {
			var mAtom = mockatom{
				id: strconv.Itoa(i),
				status: 1,
			}

			aStream <- mAtom

			// Sleep if there is a delay in the mock data source
			if this.delay != nil {
				time.Sleep(*this.delay)
			}
		}
	}(atomStream)

	return atomStream
}

type mockatom struct {
	id string
	status int
	process func(ctx context.Context, payload []byte) (err error)
}

func (this mockatom) GetId() (id string) {
	return this.id
}

func (this mockatom) GetStatus() (status int) {
	return this.status
}

func (this mockatom) Process(ctx context.Context, payload []byte) (err error) {

	// If this mock atom is mocked with a process function then execute the process function
	// Otherwise just exit
	if this.process != nil {
		err = this.process(ctx, payload)
	}

	return err
}

func (this mockatom) Panic(ctx context.Context) {
}

func (this mockatom) Complete(ctx context.Context) (err error) {
	return err
}