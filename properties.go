package atomizer

import (
	"encoding/json"
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
		Error      error           `json:"errors,omitempty"`
		Result     json.RawMessage `json:"result"`
	}{}

	err := json.Unmarshal(data, &jsonP)
	if err != nil {
		return err
	}

	p.ElectronID = jsonP.ElectronID
	p.AtomID = jsonP.AtomID
	p.Start = jsonP.Start
	p.End = jsonP.End
	p.Error = jsonP.Error
	p.Result = []byte(jsonP.Result)

	return nil
}

// MarshalJSON implements the custom json marshaler for properties
func (p *Properties) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ElectronID string          `json:"electronId"`
		AtomID     string          `json:"atomId"`
		Start      time.Time       `json:"starttime"`
		End        time.Time       `json:"endtime"`
		Error      error           `json:"errors,omitempty"`
		Result     json.RawMessage `json:"result"`
	}{
		ElectronID: p.ElectronID,
		AtomID:     p.AtomID,
		Start:      p.Start,
		End:        p.End,
		Error:      p.Error,
		Result:     json.RawMessage(p.Result),
	})
}
