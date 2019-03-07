package testing

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

// GetMockConductor is a method for creating a mocked source
func GetMockConductor(atoms int, delay *time.Duration, process func(ctx context.Context, payload []byte) (err error)) (source MockConductor) {
	return MockConductor{
		atoms,
		delay,
		process,
	}
}

// MockConductor is a struct implementation for mocking the source for unit testing
type MockConductor struct {
	atoms   int
	delay   *time.Duration
	process func(ctx context.Context, payload []byte) (err error)
}

// GetAtoms is a method for mocking a source of atoms
// TODO: Atoms are not what is returned from a source, the sources return electrons which are what allow
//  the atom to do it's work this is in the form of a []byte which allows for atomizer to be serialization agnostic since
//  the deserialization will occur through the implementation of the atom itself rather than in this atomizer library
func (conductor MockConductor) GetAtoms() <-chan MockAtom {
	var atomStream = make(chan MockAtom)

	// Push off the atom stream to a go routine and loop through the expected
	// atoms for mocking
	go func(aStream chan<- MockAtom) {
		for i := 0; i < conductor.atoms; i++ {
			//var mAtom = MockAtom{
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
