package testhelper

import (
	"testing"

	"github.com/aws/aws-sdk-go/service/athena"
	"github.com/stretchr/testify/assert"
)

func TestCreateRows(t *testing.T) {
	tests := []struct {
		rawRows [][]string
		want    []*athena.Row
	}{
		{[][]string{}, []*athena.Row{}},
		{[][]string{{}}, []*athena.Row{{Data: []*athena.Datum{}}}},
		{[][]string{{}, {}}, []*athena.Row{{Data: []*athena.Datum{}}, {Data: []*athena.Datum{}}}},
		{
			rawRows: [][]string{{"foo"}},
			want:    []*athena.Row{{Data: []*athena.Datum{new(athena.Datum).SetVarCharValue("foo")}}},
		},
		{
			rawRows: [][]string{{"foo", "bar"}},
			want: []*athena.Row{
				{
					Data: []*athena.Datum{
						new(athena.Datum).SetVarCharValue("foo"),
						new(athena.Datum).SetVarCharValue("bar"),
					},
				},
			},
		},
		{
			rawRows: [][]string{{"foo"}, {"bar"}},
			want: []*athena.Row{
				{Data: []*athena.Datum{new(athena.Datum).SetVarCharValue("foo")}},
				{Data: []*athena.Datum{new(athena.Datum).SetVarCharValue("bar")}},
			},
		},
		{
			rawRows: [][]string{{"foo", "bar"}, {"baz", "foobar"}},
			want: []*athena.Row{
				{
					Data: []*athena.Datum{
						new(athena.Datum).SetVarCharValue("foo"),
						new(athena.Datum).SetVarCharValue("bar"),
					},
				},
				{
					Data: []*athena.Datum{
						new(athena.Datum).SetVarCharValue("baz"),
						new(athena.Datum).SetVarCharValue("foobar"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		got := CreateRows(tt.rawRows)

		assert.Equal(t, tt.want, got, "Raw rows: %#v", tt.rawRows)
	}
}
