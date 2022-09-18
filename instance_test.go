package engine

import (
	"context"
	"testing"
)

func Test_instance_bond(t *testing.T) {
	ctx, cancel := _ctx(context.TODO())

	tests := []struct {
		name string
		inst instance
		atom Processor
		err  bool
	}{
		{
			"valid instance",
			instance{
				req:    noopelectron,
				trans:  &noopconductor{},
				prop:   &Response{},
				ctx:    ctx,
				cancel: cancel,
			},
			&noopatom{},
			false,
		},
		{
			"invalid instance / missing electron",
			instance{
				trans:  &noopconductor{},
				prop:   &Response{},
				ctx:    ctx,
				cancel: cancel,
			},
			&noopatom{},
			true,
		},
		{
			"invalid instance / missing conductor",
			instance{
				req:    noopelectron,
				prop:   &Response{},
				ctx:    ctx,
				cancel: cancel,
			},
			&noopatom{},
			true,
		},
		{
			"invalid instance / nil atom",
			instance{
				req:    noopelectron,
				trans:  &noopconductor{},
				prop:   &Response{},
				ctx:    ctx,
				cancel: cancel,
			},
			nil,
			true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.inst.bond(test.atom)
			if err != nil && !test.err {
				t.Fatalf(
					"expected success, got error %s",
					err,
				)
			}
		})
	}
}

func Test_instance_complete(t *testing.T) {
	ctx, cancel := _ctx(context.TODO())

	tests := []struct {
		name string
		inst instance
		atom Processor
		err  bool
	}{
		{
			"valid instance",
			instance{
				req:    noopelectron,
				trans:  &noopconductor{},
				prop:   &Response{},
				ctx:    ctx,
				cancel: cancel,
			},
			&noopatom{},
			false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.inst.bond(test.atom)
			if err != nil && !test.err {
				t.Fatalf(
					"expected success, got error %s",
					err,
				)
			}
		})
	}
}

func Test_instance_execute(t *testing.T) {
	ctx, _ := _ctx(context.TODO())

	tests := []struct {
		name string
		inst instance
		err  bool
	}{
		{
			"valid instance",
			instance{
				req:   noopelectron,
				trans: &noopconductor{},
				prop:  &Response{},
				proc:  &noopatom{},
			},
			false,
		},
		{
			"invalid instance - no electron",
			instance{
				trans: &noopconductor{},
				prop:  &Response{},
				proc:  &noopatom{},
			},
			true,
		},
		{
			"invalid instance - no conductor",
			instance{
				req:  noopelectron,
				prop: &Response{},
				proc: &noopatom{},
			},
			true,
		},
		{
			"invalid instance - no atom",
			instance{
				req:   noopelectron,
				trans: &noopconductor{},
				prop:  &Response{},
			},
			true,
		},
		{
			"invalid instance - panic atom",
			instance{
				req:   noopelectron,
				trans: &noopconductor{},
				prop:  &Response{},
				proc:  &panicatom{},
			},
			true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.inst.execute(ctx)
			if err != nil && !test.err {
				t.Fatalf(
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
				req:   noopelectron,
				trans: &noopconductor{},
				proc:  &noopatom{},
			},
			true,
		},
		{
			"invalid instance / nil atom",
			instance{
				req:   noopelectron,
				trans: &noopconductor{},
			},
			false,
		},
		{
			"invalid instance / invalid electron",
			instance{
				req: &Request{
					Origin: "empty",
					ID:     "empty",
				},
				trans: &noopconductor{},
				proc:  &noopatom{},
			},
			false,
		},
		{
			"invalid instance / invalid electron",
			instance{
				req:   &Request{},
				trans: &noopconductor{},
				proc:  &noopatom{},
			},
			false,
		},
		{
			"invalid instance / invalid conductor",
			instance{
				req:  noopelectron,
				proc: &noopatom{},
			},
			false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			valid := test.inst.Validate()
			if valid != test.valid {
				t.Fatalf(
					"valid mismatch, expected %v got %v",
					valid,
					test.inst.Validate(),
				)
			}
		})
	}
}
