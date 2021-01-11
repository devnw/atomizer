// Copyright © 2019 Developer Network, LLC
//
// This file is subject to the terms and conditions defined in
// file 'LICENSE', which is part of this source code package.

package atomizer

import (
	"encoding/base64"
	"encoding/gob"
	"encoding/json"
	"strings"
	"time"
)

func init() {
	gob.Register(Electron{})
}

// Electron is the base electron that MUST parse from the payload
// from the conductor
type Electron struct {
	// SenderID is the unique identifier for the node that sent the
	// electron
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
}

// UnmarshalJSON reads in a []byte of JSON data and maps it to the Electron
// struct properly for use throughout Atomizer
func (e *Electron) UnmarshalJSON(data []byte) error {
	jsonE := struct {
		SenderID string          `json:"senderid"`
		ID       string          `json:"id"`
		AtomID   string          `json:"atomid"`
		Timeout  *time.Duration  `json:"timeout,omitempty"`
		Payload  json.RawMessage `json:"payload,omitempty"`
	}{}

	err := json.Unmarshal(data, &jsonE)
	if err != nil {
		return err
	}

	e.SenderID = jsonE.SenderID
	e.ID = jsonE.ID
	e.AtomID = jsonE.AtomID
	e.Timeout = jsonE.Timeout

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
func (e Electron) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		SenderID string          `json:"senderid"`
		ID       string          `json:"id"`
		AtomID   string          `json:"atomid"`
		Timeout  *time.Duration  `json:"timeout,omitempty"`
		Payload  json.RawMessage `json:"payload,omitempty"`
	}{
		SenderID: e.SenderID,
		ID:       e.ID,
		AtomID:   e.AtomID,
		Timeout:  e.Timeout,
		Payload:  json.RawMessage(e.Payload),
	})
}

// Validate ensures that the electron information is intact for proper
// execution
func (e Electron) Validate() (valid bool) {
	if e.SenderID != "" &&
		e.ID != "" &&
		e.AtomID != "" {

		valid = true
	}

	return valid
}
