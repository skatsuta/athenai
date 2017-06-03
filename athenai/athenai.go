package athenai

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/athena"
	"github.com/skatsuta/athenai/exec"
)

const (
	tickInterval = 1000 * time.Millisecond
)

// Config is a configuration information for Athenai.
type Config struct {
	exec.QueryConfig
	Debug  bool
	Region string
	Silent bool
}

// Athenai is a main struct to run this app.
type Athenai struct {
	out      io.Writer
	cfg      *Config
	client   *athena.Athena
	interval time.Duration
}

// New creates a new Athena.
func New(out io.Writer, cfg *Config) *Athenai {
	if cfg == nil {
		cfg = &Config{}
	}

	a := &Athenai{
		out:      out,
		cfg:      cfg,
		client:   newClient(cfg),
		interval: tickInterval,
	}
	return a
}

func (a *Athenai) print(x ...interface{}) {
	fmt.Fprint(a.out, x...)
}

// ShowProgressMsg shows progress messages while queries are being executed.
func (a *Athenai) ShowProgressMsg() {
	// Do nothing if it's run in silent mode
	if a.cfg.Silent {
		return
	}

	// Start a new goroutine to show progress messages regularly
	go func() {
		a.print("Running query")
		tick := time.Tick(a.interval)
		for {
			select {
			case <-tick:
				a.print(".")
			}
		}
	}()
}

// newClient creates a new Athena client.
func newClient(cfg *Config) *athena.Athena {
	// Create a service configuration
	c := aws.NewConfig().WithRegion(cfg.Region)

	// Set log level
	if cfg.Debug {
		c = c.WithLogLevel(aws.LogDebugWithHTTPBody | aws.LogDebugWithRequestErrors)
	} else {
		// Surpress log outputs
		log.SetOutput(ioutil.Discard)
	}

	return athena.New(session.Must(session.NewSession(c)))
}
