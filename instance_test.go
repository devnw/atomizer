package atomizer

import (
	"testing"
)

func Test_instance_bond(t *testing.T) {

	ctx, cancel := _ctx(nil)

	tests := []struct {
		name string
		inst instance
		atom Atom
		err  bool
	}{
		{
			"valid instance",
			instance{
				electron:   noopelectron,
				conductor:  &noopconductor{},
				properties: &Properties{},
				ctx:        ctx,
				cancel:     cancel,
			},
			&noopatom{},
			false,
		},
		{
			"invalid instance / missing electron",
			instance{
				conductor:  &noopconductor{},
				properties: &Properties{},
				ctx:        ctx,
				cancel:     cancel,
			},
			&noopatom{},
			true,
		},
		{
			"invalid instance / missing conductor",
			instance{
				electron:   noopelectron,
				properties: &Properties{},
				ctx:        ctx,
				cancel:     cancel,
			},
			&noopatom{},
			true,
		},
		{
			"invalid instance / nil atom",
			instance{
				electron:   noopelectron,
				conductor:  &noopconductor{},
				properties: &Properties{},
				ctx:        ctx,
				cancel:     cancel,
			},
			nil,
			true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.inst.bond(test.atom)
			if err != nil && !test.err {
				t.Errorf(
					"expected success, got error %s",
					err,
				)
			}
		})
	}

}

func Test_instance_complete(t *testing.T) {

	ctx, cancel := _ctx(nil)

	tests := []struct {
		name string
		inst instance
		atom Atom
		err  bool
	}{
		{
			"valid instance",
			instance{
				electron:   noopelectron,
				conductor:  &noopconductor{},
				properties: &Properties{},
				ctx:        ctx,
				cancel:     cancel,
			},
			&noopatom{},
			false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.inst.bond(test.atom)
			if err != nil && !test.err {
				t.Errorf(
					"expected success, got error %s",
					err,
				)
			}
		})
	}

}

func Test_instance_execute(t *testing.T) {

	ctx, _ := _ctx(nil)

	tests := []struct {
		name string
		inst instance
		err  bool
	}{
		{
			"valid instance",
			instance{
				electron:   noopelectron,
				conductor:  &noopconductor{},
				properties: &Properties{},
				atom:       &noopatom{},
			},
			false,
		},
		{
			"invalid instance - no electron",
			instance{
				conductor:  &noopconductor{},
				properties: &Properties{},
				atom:       &noopatom{},
			},
			true,
		},
		{
			"invalid instance - no conductor",
			instance{
				electron:   noopelectron,
				properties: &Properties{},
				atom:       &noopatom{},
			},
			true,
		},
		{
			"invalid instance - no atom",
			instance{
				electron:   noopelectron,
				conductor:  &noopconductor{},
				properties: &Properties{},
			},
			true,
		},
		{
			"invalid instance - panic atom",
			instance{
				electron:   noopelectron,
				conductor:  &noopconductor{},
				properties: &Properties{},
				atom:       &panicatom{},
			},
			true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.inst.execute(ctx)
			if err != nil && !test.err {
				t.Errorf(
					"expected success, got error %s",
					err,
				)
			}
		})
	}

}

func Test_instance_Validate(t *testing.T) {

	tests := []struct {
		name  string
		inst  instance
		valid bool
	}{
		{
			"valid instance",
			instance{
				electron:  noopelectron,
				conductor: &noopconductor{},
				atom:      &noopatom{},
			},
			true,
		},
		{
			"invalid instance / nil atom",
			instance{
				electron:  noopelectron,
				conductor: &noopconductor{},
			},
			false,
		},
		{
			"invalid instance / invalid electron",
			instance{
				electron: Electron{
					SenderID: "empty",
					ID:       "empty",
				},
				conductor: &noopconductor{},
				atom:      &noopatom{},
			},
			false,
		},
		{
			"invalid instance / invalid electron",
			instance{
				electron:  Electron{},
				conductor: &noopconductor{},
				atom:      &noopatom{},
			},
			false,
		},
		{
			"invalid instance / invalid conductor",
			instance{
				electron: noopelectron,
				atom:     &noopatom{},
			},
			false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			valid := test.inst.Validate()
			if valid != test.valid {
				t.Errorf(
					"valid mismatch, expected %v got %v",
					valid,
					test.inst.Validate(),
				)
			}
		})
	}

}
