package atomizer

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
)

var nooppropNob64ErrJSON = `{"electronId":"test","atomId":"test","starttime":"0001-01-01T00:00:00Z","endtime":"0001-01-01T00:00:00Z","error":"no matchy","result":{"result":"test"}}`

var nooppropJSON = `{"electronId":"test","atomId":"test","starttime":"0001-01-01T00:00:00Z","endtime":"0001-01-01T00:00:00Z","result":{"result":"test"}}`

var noopprop = Properties{
	ElectronID: "test",
	AtomID:     "test",
	Start:      time.Time{},
	End:        time.Time{},
	Error:      nil,
	Result:     []byte(`{"result":"test"}`),
}

var nooppropErrJSON = `{"electronId":"test","atomId":"test","starttime":"0001-01-01T00:00:00Z","endtime":"0001-01-01T00:00:00Z","error":"eyJldmVudCI6eyJtZXNzYWdlIjoidGVzdCIsImVsZWN0cm9uSUQiOiIiLCJhdG9tSUQiOiIiLCJjb25kdWN0b3JJRCI6IiJ9LCJpbnRlcm5hbCI6bnVsbH0=","result":{"result":"test"}}`

var nooppropNoMatchErrJSON = `{"electronId":"test","atomId":"test","starttime":"0001-01-01T00:00:00Z","endtime":"0001-01-01T00:00:00Z","error":"eyJub21hdGNoIjoibm9tYXRjaCJ9","result":{"result":"test"}}`

var nooppropErr = Properties{
	ElectronID: "test",
	AtomID:     "test",
	Start:      time.Time{},
	End:        time.Time{},
	Error:      simple("test", nil),
	Result:     []byte(`{"result":"test"}`),
}

var nooppropNonAtomErrJSON = `{"electronId":"test","atomId":"test","starttime":"0001-01-01T00:00:00Z","endtime":"0001-01-01T00:00:00Z","error":"dGVzdA==","result":{"result":"test"}}`

var nooppropNonAtomNoMatchErrJSON = `{"electronId":"test","atomId":"test","starttime":"0001-01-01T00:00:00Z","endtime":"0001-01-01T00:00:00Z","error":"eyJub21hdGNoIjoibm9tYXRjaCJ9","result":{"result":"test"}}`

var nooppropNonAtomErr = Properties{
	ElectronID: "test",
	AtomID:     "test",
	Start:      time.Time{},
	End:        time.Time{},
	Error:      errors.New("test"),
	Result:     []byte(`{"result":"test"}`),
}

func TestProperties_MarshalJSON(t *testing.T) {

	tests := []struct {
		name string
		p    Properties
		json string
		err  bool
	}{
		{
			"valid Properties",
			noopprop,
			nooppropJSON,
			false,
		},
		{
			"valid Properties w/ error",
			nooppropErr,
			nooppropErrJSON,
			false,
		},
		{
			"valid Properties w/ non-Atom error",
			nooppropNonAtomErr,
			nooppropNonAtomErrJSON,
			false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res, err := json.Marshal(test.p)
			if err != nil && !test.err {
				t.Errorf("expected success, got error | %s", err.Error())
				return
			}

			if err == nil && test.err {
				t.Error("expected error")
				return
			}

			if strings.Compare(string(res), test.json) != 0 {
				t.Errorf(
					"mismatch: e[%s] != r[%s]",
					test.json,
					string(res),
				)
			}
		})
	}
}

func TestProperties_UnmarshalJSON(t *testing.T) {

	tests := []struct {
		name  string
		p     Properties
		json  string
		equal bool
		err   bool
	}{
		{
			"valid Properties",
			noopprop,
			nooppropJSON,
			true,
			false,
		},
		{
			"valid Properties w/ error",
			nooppropErr,
			nooppropErrJSON,
			true,
			false,
		},
		{
			"valid Properties w/ non-matching Atom error",
			nooppropErr,
			nooppropNoMatchErrJSON,
			false,
			false,
		},
		{
			"valid Properties w/ non-Atom error",
			nooppropNonAtomErr,
			nooppropNonAtomErrJSON,
			true,
			false,
		},
		{
			"valid Properties w/ non-matching non-Atom error",
			nooppropNonAtomErr,
			nooppropNonAtomNoMatchErrJSON,
			false,
			false,
		},
		{
			"invalid Properties missing base64 encoding for err",
			nooppropErr,
			nooppropNob64ErrJSON,
			false,
			true,
		},
		{
			"invalid json blob",
			Properties{},
			`{"empty"}`,
			false,
			true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p := Properties{}
			err := json.Unmarshal([]byte(test.json), &p)

			if err != nil && !test.err {
				t.Errorf("expected success, got error | %s", err.Error())
				return
			}

			if err == nil && test.err {
				t.Error("expected error")
				return
			}

			if !test.err {
				if cmp.Equal(test.p, p) != test.equal {
					t.Errorf(
						"expected equality e[%s] != r[%s]",
						spew.Sdump(test.p),
						spew.Sdump(p),
					)
				}
			}
		})
	}
}
