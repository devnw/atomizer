// Copyright © 2019 Developer Network, LLC
//
// This file is subject to the terms and conditions defined in
// file 'LICENSE', which is part of this source code package.

package engine

import (
	"context"
	"encoding/base64"
	"encoding/gob"
	"encoding/json"
	"strings"
	"time"
)

func init() {
	gob.Register(Request{})
}

// Request is the base electron that MUST parse from the payload
// from the conductor
type Request struct {
	// Origin is the unique identifier for the node that sent the electron
	Origin string

	// ID is the unique identifier of this electron
	ID string

	// AtomID is the unique identifier of the atom to execute, this is
	// the `go` type of atom to execute
	ProcessorID string

	// CopyState lets atomizer know if it should copy the state of the
	// original atom registration to the new atom instance when processing
	// a newly received electron
	//
	// NOTE: Copying the state of an Atom as registered requires that ALL
	// fields that are to be copied are **EXPORTED** otherwise they are
	// skipped
	CopyState bool

	// Payload is to be used by the registered atom to properly unmarshal
	// the []byte for the actual atom instance. RawMessage is used to
	// delay unmarshal of the payload information so the atom can do it
	// internally
	Payload []byte

	// m is the maker for instantiating the new atom
	m *maker

	// timeout is the maximum time duration that should be allowed
	// for this instance to process. After the duration is exceeded
	// the context should be canceled and the processing released
	// and a failure sent back to the conductor
	timeout *time.Duration
}

func (e *Request) Context(parent context.Context) (
	context.Context,
	context.CancelFunc,
) {
	var ctx context.Context
	var cancel context.CancelFunc

	if e.timeout != nil {
		ctx, cancel = context.WithTimeout(parent, *e.timeout)
	} else {
		ctx, cancel = context.WithCancel(parent)
	}

	// Set values on the context for the atom to use
	ctx = context.WithValue(ctx, ORIGIN, e.Origin)
	ctx = context.WithValue(ctx, PROCESSORID, e.ProcessorID)
	ctx = context.WithValue(ctx, REQUESTID, e.ID)

	return ctx, cancel
}

// UnmarshalJSON reads in a []byte of JSON data and maps it to the Electron
// struct properly for use throughout Atomizer
func (e *Request) UnmarshalJSON(data []byte) error {
	jsonE := struct {
		Origin    string          `json:"origin"`
		Atom      string          `json:"atom"`
		ID        string          `json:"id"`
		Timeout   *time.Duration  `json:"timeout,omitempty"`
		CopyState bool            `json:"copystate,omitempty"`
		Payload   json.RawMessage `json:"payload,omitempty"`
	}{}

	err := json.Unmarshal(data, &jsonE)
	if err != nil {
		return err
	}

	m, err := Get[*maker](jsonE.Atom)
	if err != nil {
		return err
	}

	e.Origin = jsonE.Origin
	e.ID = jsonE.ID
	e.ProcessorID = jsonE.Atom
	e.m = m
	e.timeout = jsonE.Timeout

	if jsonE.Payload != nil {
		pay := strings.Trim(string(jsonE.Payload), "\"")
		e.Payload, err = base64.StdEncoding.DecodeString(pay)
		if err != nil {
			e.Payload = jsonE.Payload
		}
	}

	return nil
}

// MarshalJSON implements the custom json marshaler for electron
func (e *Request) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Origin    string          `json:"origin"`
		ID        string          `json:"id"`
		AtomID    string          `json:"atom"`
		Timeout   *time.Duration  `json:"timeout,omitempty"`
		CopyState bool            `json:"copystate,omitempty"`
		Payload   json.RawMessage `json:"payload,omitempty"`
	}{
		Origin:  e.Origin,
		ID:      e.ID,
		AtomID:  e.ProcessorID,
		Timeout: e.timeout,
		Payload: json.RawMessage(e.Payload),
	})
}

// Validate ensures that the electron information is intact for proper
// execution
func (e *Request) Validate() (valid bool) {
	if e != nil &&
		e.Origin != "" &&
		e.ID != "" &&
		e.ProcessorID != "" {
		valid = true
	}

	return valid
}
