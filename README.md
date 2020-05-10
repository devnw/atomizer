# Atomizer - Massively Parallel Distributed Computing

[![CI](https://github.com/devnw/atomizer/workflows/CI/badge.svg)](https://github.com/devnw/atomizer/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/devnw/atomizer)](https://goreportcard.com/report/github.com/devnw/atomizer)
[![codecov](https://codecov.io/gh/devnw/atomizer/branch/master/graph/badge.svg)](https://codecov.io/gh/devnw/atomizer)
[![GoDoc](https://godoc.org/github.com/devnw/atomizer?status.svg)](https://pkg.go.dev/github.com/devnw/atomizer)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](http://makeapullrequest.com)

Created to facilitate simplified construction of distributed systems, the
Atomizer library was built with simplicity in mind by exposing an API which
allows users to create "Atoms" ([def](docs/definitions.md#atom)) which
contain the atomic elements of business logic for an application which when
paired with an "Electron"([def](docs/definitions.md#electron)) payload are
executed in the atomizer runtime.

## Index

* [Getting Started](#getting-started)
* [Test App](#test-app)
* [Atomizer Architecture](docs/design-methodologies.md)
* [Terminology](docs/definitions.md)
* [Conductor Creation](#conductor-creation)
* [Atom Creation](#atom-creation)
* [Electron Creation](#electron-creation)
* [Properties - Atom Results](#properties---atom-results)
* [Events](#events)
* [Element Registration](#element-registration)
  * [Init Registration](#init-registration)
  * [Direct Registration](#direct-registration)

## Getting Started

To use Atomizer you will first need to pull a copy down.

```go
go get -u github.com/devnw/atomizer
```

I highly recommend checking out the [Test App](#test-app) for a functional
example of the Atomizer framework in action.

To create an instance of Atomizer in your app you will need a Conductor
([def](docs/definitions.md#conductor)). Currently there is an AMQP conductor
that was built for the creation of Atomizer which can be found
at ([github.com/devnw/amqp](https://github.com/devnw/amqp)), or you can create
your own conductor by following the
[Conductor creation instructions](#conductor-creation).

Once you have registered your Conductor using one of the
[Element Registration](#element-registration) methods then you must create
and register your Atom implementations following the
[Atom Creation](#atom-creation) instructions.

After registering at least one Conductor and one Atom in your Atomizer
instance you can begin accepting requests. Here is an example of initializing
the framework and registering an Atom and Conductor to begin processing.

```go
a := Atomize(ctx, nil)

// Start the Atomizer processing system
err := a.Exec()

if err != nil {
   ...
}

// Register your Conductor
err = a.Register(&MyConductor{})
if err != nil {
   ...
}

// Register your Atom
err = a.Register(&MyAtom{})
if err != nil {
   ...
}
```

Now that the framework is initialized you can push processing requests
through your Conductor for your registered Atoms and the Atomizer framework
will execute the Electron([def](docs/definitions.md#electron)) payload by
bonding it with the correct registered Atom and returning the resulting
`Properties` over the Conductor back to the sender.

## Test App

Since the concepts here are new to people seeing this for the first time a
capstone team of students from the University of Illinois Springfield put
together a test app to showcase a working implementation of the Atomizer
framework.

This Test App implements a MonteCarlo *pi* simulation using the Atomizer
framework. The Atom implementation can be found here
[MonteCarlo *pi*](https://github.com/devnw/montecarlopi). This implementation
uses two Atoms. The first Atom "MonteCarlo" is a
[Spawner](docs/design-methodologies.md#atom-types) which creates payloads for
the Toss Atom which is an [Atomic](docs/design-methodologies.md#atom-types)
Atom.

To run a simulation the capstone team created a Web UI built on NodeJS which
creates the initial Electron JSON payload which is sent into a RabbitMQ AMQP
conductor. This electron causes the MonteCarlo Atom to spawn Electron
payloads for the Toss atom based on the number of tosses passed in through
the UI. Once the Tosses are all complete the MonteCarlo Atom calculates the
*pi* estimation and returns that the the original sender (i.e. the UI)

For the more technically inclined there is also now a console test
application which can be found here.

[Atomizer Test Console](https://github.com/devnw/atomizer-test-console)

This test application will allow you to run the same MonteCarlo *pi*
simulation from command line without needing to setup a NodeJS app or pull
down the corresponding Docker container.

Thank you to

* Matthew N Meyer ([@MatthewNormanMeyer](https://github.com/MatthewNormanMeyer))
* Megan Pugliese ([@mugatupotamus](https://github.com/mugatupotamus))
* Nick Heiting ([@nheiting](https://github.com/nheiting))
* Benji Vesterby ([@benjivesterby](https://github.com/benjivesterby))

## Conductor Creation

Creation of a conductor involves implementing the Conductor interface as
described in the Atomizer library which can be seen below.

```go
// Conductor is the interface that should be implemented for passing
// electrons to the atomizer that need processing. This should generally be
// registered with the atomizer in an initialization script
type Conductor interface {

    // Receive gets the atoms from the source
    // that are available to atomize
    Receive(ctx context.Context) <-chan Electron

    // Complete mark the completion of an electron instance
    // with applicable statistics
    Complete(ctx context.Context, properties Properties) error

    // Send sends electrons back out through the conductor for
    // additional processing
    Send(ctx context.Context, e Electron) (<-chan Properties, error)

    // Close cleans up the conductor
    Close()
}
```

Once you have created your Conductor you must then register it into the
framework using one of the [Element Registration](#element-registration)
methods for Atomizer.

## Atom Creation

The Atomizer library is the framework on which you can build your distributed
system. To do this you need to create "Atoms"([def](docs/definitions.md#atom))
which implement your specific business logic. To do this you must implement
the Atom interface seen below.

```go
type Atom interface {
    Process(ctx context.Context,c Conductor,e Electron,) ([]byte, error)
}
```

Once you have created your Atom you must then register it into the framework
using one of the [Element Registration](#element-registration) methods for
Atomizer.

## Electron Creation

Electrons([def](docs/definitions.md#atom)) are one of the most important
elements of the Atomizer framework because they supply the `data` necessary
for the framework to properly execute an Atom since the Atom implementation
is pure business logic.

```go
// Electron is the base electron that MUST parse from the payload
// from the conductor
type Electron struct {
    // SenderID is the unique identifier for the node that sent the electron
    SenderID string

    // ID is the unique identifier of this electron
    ID string

    // AtomID is the identifier of the atom for this electron instance
    // this is generally `package.Type`. Use the atomizer.ID() method
    // if unsure of the type for an Atom.
    AtomID string

    // Timeout is the maximum time duration that should be allowed
    // for this instance to process. After the duration is exceeded
    // the context should be cancelled and the processing released
    // and a failure sent back to the conductor
    Timeout *time.Duration

    // Payload is to be used by the registered atom to properly unmarshal
    // the []byte for the actual atom instance. RawMessage is used to
    // delay unmarshal of the payload information so the atom can do it
    // internally
    Payload []byte
}
```

The most important part for an Atom is the `Payload`. This `[]byte` holds
encoded data which can be decoded in your Atom implementation. This is how
the Atom receives state information for which it uses to process. Decoding
the `Payload` is the responsibility of the Atom implementation. The other
fields of the Electron are used by the Atomizer internally, but are available
to an Atom as part of the Process method if necessary.

Electrons are provided to the Atomizer framework through a registered
Conductor, generally a Message Queue.

## Properties - Atom Results

The results of Atom processing are contained in the `Properties` struct in
the Atomizer library. This struct contains metadata about the processing that
took place as well as the results or errors of the specific Atom which was
executed from a request.

```go
// Properties is the struct for storing properties information after the
// processing of an atom has completed so that it can be sent to the
// original requestor
type Properties struct {
    ElectronID string
    AtomID     string
    Start      time.Time
    End        time.Time
    Error      error
    Result     []byte
}
```

The Result field is the `[]byte` that is returned from the Atom that was
executed, and in general the Error property contains the `error` which was
returned from the Atom as well, however if there are errors in processing
an Atom that are not related to the internal processing of the Atom that
error will be returned on the `Properties` struct in place of the Atom error
as it is unlikely the Atom executed.

## Events

Atomizer exports a method called `Events` which returns a
`<- chan interface{}`. When you call this method an internal channel in
Atomizer is created which then **MUST** be monitored for events or it will
block processing.

The purpose of this method is allow implementations to monitor the events
occurring inside an instance of the Atomizer for informational or error events.
These should be monitored inside a `go` routine in the client application.

Along with this Atomizer exports two important structs. `atomizer.Error` and
`atomizer.Event`. These contain information such as the `AtomID`,
`ElectronID` or `ConductorID` that the event applies to as well as any
message or error that may be part of the event.

Both `atomizer.Error` and `atomizer.Event` implement the `fmt.Stringer`
interface to make for easy logging.

`atomizer.Error` also implements the `error` interface as well as the
`Unwrap` method for nested errors.

```go
// Event indicates an atomizer event has taken
// place that is not categorized as an error
// Event implements the stringer interface but
// does NOT implement the error interface
type Event struct {
    // Message from atomizer about this error
    Message string `json:"message"`

    // ElectronID is the associated electron instance
    // where the error occurred. Empty ElectronID indicates
    // the error was not part of a running electron instance.
    ElectronID string `json:"electronID"`

    // AtomID is the atom which was processing when
    // the error occurred. Empty AtomID indicates
    // the error was not part of a running atom.
    AtomID string `json:"atomID"`

    // ConductorID is the conductor which was being
    // used for receiving instructions
    ConductorID string `json:"conductorID"`
}

// Error is an error type which provides specific
// atomizer information as part of an error
type Error struct {

    // Event is the event that took place to create
    // the error and contains metadata relevant to the error
    Event Event `json:"event"`

    // Internal is the internal error
    Internal error `json:"internal"`
}
```

## Element Registration

There are two primary elements in Atomizer that can be registered into the
system currently. Atoms and Conductors. These two pieces are essential to
how the Atomizer framework is able to execute processing requests. These
elements can be registered into the Atomizer framework through two different
methods. [Init Registration](#init-registration), and
[Direct Registration](#direct-registration).

### Init Registration

Atoms & Conductors can be registered in an `init` method in your code as seen
below.

```go
func init() {
    err := atomizer.Register(&MonteCarlo{})
    if err != nil {
        ...
    }
}
```

### Direct Registration

Direct registration happens when you use the `Register` method directly on an
Atomizer instance as seen below.

```go
a := Atomize(ctx, nil)

// Start the Atomizer processing system
err := a.Exec()

if err != nil {
   ...
}

// Register the Atom
err = a.Register(&MonteCarlo{})
if err != nil {
   ...
}
```

