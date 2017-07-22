package filter

import (
	"context"
	"log"
	"strings"

	"github.com/google/btree"
	"github.com/peco/peco"
	"github.com/peco/peco/line"
	"github.com/pkg/errors"
)

const (
	collectResultsErr = "collect results"
)

// Filter is a filter to select entries.
type Filter interface {
	SetInput(input string)
	Run(ctx context.Context) error
	Len() int
	Each(fn func(item string) bool)
}

type pecoFilter struct {
	p *peco.Peco
}

// New creates a new Filter to filter input.
func New() Filter {
	p := peco.New()
	p.Argv = []string{}
	return &pecoFilter{p: p}
}

// SetInput sets input to f.
func (f *pecoFilter) SetInput(input string) {
	f.p.Stdin = strings.NewReader(input)
}

// Run performs filtering.
func (f *pecoFilter) Run(ctx context.Context) error {
	err := f.p.Run(ctx)
	if err != nil && !strings.Contains(err.Error(), collectResultsErr) {
		return errors.Wrap(err, "error filtering entries")
	}

	s := f.p.Selection()
	if s.Len() == 0 {
		n := f.p.Location().LineNumber()
		if line, err := f.p.CurrentLineBuffer().LineAt(n); err == nil {
			log.Printf("No line is selected. Adding the current line %d\n", n)
			s.Add(line)
		}
	}
	return nil
}

// Len returns the length of selected items filtered by f.
func (f *pecoFilter) Len() int {
	return f.p.Selection().Len()
}

// Each iterates over selected items and call fn with them.
func (f *pecoFilter) Each(fn func(item string) bool) {
	f.p.Selection().Ascend(func(it btree.Item) bool {
		return fn(it.(line.Line).Output())
	})
}
