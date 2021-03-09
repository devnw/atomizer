// Copyright © 2019 Developer Network, LLC
//
// This file is subject to the terms and conditions defined in
// file 'LICENSE', which is part of this source code package.

package engine

import (
	"encoding/json"
	"errors"
	"time"
)

// TODO: Set it up so that requests can be made to check the properties of
// a bonded electron / atom at runtime

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

// UnmarshalJSON reads in a []byte of JSON data and maps it to the Properties
// struct properly for use throughout Atomizer
func (p *Properties) UnmarshalJSON(data []byte) error {
	jsonP := struct {
		ElectronID string          `json:"electronId"`
		AtomID     string          `json:"atomId"`
		Start      time.Time       `json:"starttime"`
		End        time.Time       `json:"endtime"`
		Error      []byte          `json:"error,omitempty"`
		Result     json.RawMessage `json:"result"`
	}{}

	err := json.Unmarshal(data, &jsonP)
	if err != nil {
		return err
	}

	if jsonP.Error != nil {
		e := &Error{}
		err := json.Unmarshal(jsonP.Error, &e)
		if err == nil {
			p.Error = e
		} else {
			p.Error = errors.New(string(jsonP.Error))
		}
	}

	p.ElectronID = jsonP.ElectronID
	p.AtomID = jsonP.AtomID
	p.Start = jsonP.Start
	p.End = jsonP.End
	p.Result = []byte(jsonP.Result)

	return nil
}

// MarshalJSON implements the custom json marshaler for properties
func (p *Properties) MarshalJSON() ([]byte, error) {
	var eString []byte
	if p.Error != nil {
		_, ok := p.Error.(*Error)
		if ok {
			var err error
			eString, err = json.Marshal(p.Error)
			if err != nil {
				eString = []byte(p.Error.Error())
			}
		} else {
			eString = []byte(p.Error.Error())
		}
	}

	return json.Marshal(&struct {
		ElectronID string          `json:"electronId"`
		AtomID     string          `json:"atomId"`
		Start      time.Time       `json:"starttime"`
		End        time.Time       `json:"endtime"`
		Error      []byte          `json:"error,omitempty"`
		Result     json.RawMessage `json:"result"`
	}{
		ElectronID: p.ElectronID,
		AtomID:     p.AtomID,
		Start:      p.Start,
		End:        p.End,
		Error:      eString,
		Result:     json.RawMessage(p.Result),
	})
}

// Equal determines if two properties structs are equal to eachother
// TODO: Should this use reflect.DeepEqual?
func (p *Properties) Equal(p2 *Properties) bool {
	var eEquals bool
	if p.Error != nil {
		if p2.Error != nil {
			eEquals = p.Error.Error() == p2.Error.Error()
		}
	} else if p.Error == nil && p2.Error == nil {
		eEquals = true
	}

	return p.ElectronID == p2.ElectronID &&
		p.AtomID == p2.AtomID &&
		p.Start.Equal(p2.Start) &&
		p.End.Equal(p2.End) &&
		string(p.Result) == string(p2.Result) &&
		eEquals
}
