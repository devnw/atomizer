# Definitions

## Atom

An Atom is a piece of business logic that implements the Process method of the
Atom interface. Atoms are the primary workhorse of the Atomizer. Without their
payload, known as Electrons, Atoms cannot be processed. A bonded Atom (i.e.,
instance) is an Atom that has been initialized with a valid Electron.

## Conductor

A Conductor is the “driver” that connects to message queues (or to in-memory
caches or other communication frameworks) for passing processing payloads
(i.e., Electrons) throughout the cluster. This driver implements the Conductor
interface. For example, an Atom that connects to another Atom directly for
processing requests could be a form of Conductor.

## Electron

An Electron is the payload that is retrieved from the message queue or other
communication processor. It contains data in the form of a byte slice (could
be json, protobuf, or gob-encoded, etc...) that is then used to properly
initialize an Atom. Essentially, the Electron is the configuration for the
business logic that is internal to the Atom.
