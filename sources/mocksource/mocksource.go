package mocksource

import (
	"context"
	"strconv"
	"time"
)

// TODO: Setup quarks and callbacks
// TODO: Add timeouts to the payload coming from the message source *time.Duration, if nil then it's ignored and the context uses a standard cancel context
//  otherwise it uses the timeout context with the duration from the payload
//  also include other types of payload identifiers like "long running" continuous, etc...

// TODO: setup a sub package that can be added to add logging metrics to the atomizer so that it will show performance of each atom and which atom is running, etc...
//  possibly could even include the option to turn on the profiler

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

// TODO: Atoms are not what is returned from a source, the sources return electrons which are what allow
//  the atom to do it's work this is in the form of a []byte which allows for atomizer to be serialization agnostic since
//  the deserialization will occur through the implementation of the atom itself rather than in this atomizer library
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

type mockelectron struct {

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