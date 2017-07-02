package filter

import (
	"context"
	"strings"

	"github.com/peco/peco"
	"github.com/peco/peco/line"
)

// Filter is a filter to select query executions.
type Filter interface {
	SetInput(input string)
	Run(ctx context.Context) error
	Selection() *peco.Selection
	CurrentLineBuffer() Buffer
	Location() Location
}

type pecoFilter struct {
	*peco.Peco
}

// New creates a new Filter to filter input.
func New() Filter {
	p := peco.New()
	p.Argv = []string{}
	return &pecoFilter{Peco: p}
}

func (f *pecoFilter) SetInput(input string) {
	f.Stdin = strings.NewReader(input)
}

func (f *pecoFilter) CurrentLineBuffer() Buffer {
	return f.Peco.CurrentLineBuffer()
}

func (f *pecoFilter) Location() Location {
	return f.Peco.Location()
}

// Buffer interface is used for containers for lines to be processed.
// peco.Buffer interface implements this.
type Buffer interface {
	LineAt(n int) (line.Line, error)
}

// Location interface represents a location in lines.
// *peco.Location struct implements this.
type Location interface {
	LineNumber() int
}
