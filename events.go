// Copyright © 2019 Developer Network, LLC
//
// This file is subject to the terms and conditions defined in
// file 'LICENSE', which is part of this source code package.

package engine

import (
	"encoding/gob"
	"fmt"
	"strings"

	"go.structs.dev/gen"
)

func init() {
	gob.Register(&Error{})
	gob.Register(&Event{})
	gob.Register(Metadata{})
}

const (
	ORIGIN      = "origin"
	REQUESTID   = "request_id"
	PROCESSORID = "processor_id"
	TRANSPORTID = "transport_id"
)

type Metadata gen.Map[string, any]

func (m Metadata) String() string {
	if len(m) == 0 {
		return ""
	}

	var meta []string
	for k, v := range m {
		meta = append(meta, fmt.Sprintf("%s: <%s>", k, v))
	}

	return strings.Join(meta, ", ")
}

// Event indicates an atomizer event has taken place
type Event struct {
	Msg  string   `json:"msg"`
	Meta Metadata `json:"meta"`
}

func (e *Event) Event() string {
	return e.String()
}

func (e *Event) String() string {
	out := e.Msg

	meta := e.Meta.String()
	if meta != "" {
		out = fmt.Sprintf("%s | %s", out, meta)
	}

	return out
}

// Error is an error type which provides specific
// atomizer information as part of an error
type Error struct {
	Msg   string
	Meta  Metadata
	Inner error
}

func (e *Error) Error() string {
	return e.String()
}

func (e *Error) String() string {
	out := e.Msg

	if e.Inner != nil {
		out = fmt.Sprintf("%s: %s", out, e.Inner.Error())
	}

	meta := e.Meta.String()
	if meta != "" {
		out = fmt.Sprintf("%s | %s", out, meta)
	}

	return out
}

type wrapped interface {
	Unwrap() error
}

// Unwrap unwraps the error to the deepest error and returns that one
func (e *Error) Unwrap() (err error) {
	// Unwrap all errors until we get to the deepest one
	if w, ok := e.Inner.(wrapped); ok {
		err = w.Unwrap()
	}

	return err
}
