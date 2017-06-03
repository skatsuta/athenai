package athenai

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestShowProgressMsg(t *testing.T) {
	want := "Running query.." // Message shown during 2 ticks

	var out bytes.Buffer
	a := &Athenai{
		out:      &out,
		cfg:      &Config{},
		interval: 10 * time.Millisecond,
	}
	a.ShowProgressMsg()
	<-time.After(29 * time.Millisecond) // Wait for less than 30 ms (3 ticks)

	assert.Equal(t, want, out.String())
}
