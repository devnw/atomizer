package atomizer

import (
	"context"
	"time"
)

// TODO: Setup quarks and callbacks
// TODO: Add timeouts to the payload coming from the message source *time.Duration, if nil then it's ignored and the context uses a standard cancel context
//  otherwise it uses the timeout context with the duration from the payload
//  also include other types of payload identifiers like "long running" continuous, etc...

// TODO: setup a sub package that can be added to add logging metrics to the atomizer so that it will show performance of each atom and which atom is running, etc...
//  possibly could even include the option to turn on the profiler

// getmockconductor is a method for creating a mocked source
func getmockconductor(atoms int, delay *time.Duration, process func(ctx context.Context, payload []byte) (err error)) (source mockconductor) {
	return mockconductor{
		atoms,
		delay,
		process,
	}
}

// mockconductor is a struct implementation for mocking the source for unit testing
type mockconductor struct {
	atoms   int
	delay   *time.Duration
	process func(ctx context.Context, payload []byte) (err error)
}

// GetAtoms is a method for mocking a source of atoms
// TODO: Atoms are not what is returned from a source, the sources return electrons which are what allow
//  the atom to do it's work this is in the form of a []byte which allows for atomizer to be serialization agnostic since
//  the deserialization will occur through the implementation of the atom itself rather than in this atomizer library
func (conductor mockconductor) GetAtoms() <-chan mockatom {
	var atomStream = make(chan mockatom)

	// Push off the atom stream to a go routine and loop through the expected
	// atoms for mocking
	go func(aStream chan<- mockatom) {
		for i := 0; i < conductor.atoms; i++ {
			//var mAtom = mockatom{
			//	id: strconv.Itoa(i),
			//	status: 1,
			//}

			//aStream <- mAtom
			//
			//// Sleep if there is a delay in the mock data source
			//if conductor.delay != nil {
			//	time.Sleep(*conductor.delay)
			//}
		}
	}(atomStream)

	return atomStream
}
