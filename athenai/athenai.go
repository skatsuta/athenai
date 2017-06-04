package athenai

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/athena"
	"github.com/aws/aws-sdk-go/service/athena/athenaiface"
	"github.com/chzyer/readline"
	"github.com/pkg/errors"
	"github.com/skatsuta/athenai/exec"
	"github.com/skatsuta/athenai/print"
)

const (
	tickInterval = 1000 * time.Millisecond
)

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

// splitStmts splits SQL statements in the query by semicolons.
// It drops empty statements.
func splitStmts(query string) []string {
	splitted := strings.Split(query, ";")

	// Filtering without allocating: https://github.com/golang/go/wiki/SliceTricks#filtering-without-allocating
	stmts := splitted[:0]
	for _, s := range splitted {
		// Select non-empty statements
		if strings.TrimSpace(s) != "" {
			stmts = append(stmts, s)
		}
	}

	return stmts
}

// readlineCloser is an interface to read every line in REPL and then close it.
type readlineCloser interface {
	Readline() (string, error)
	Close() error
}

// Config is a configuration information for Athenai.
type Config struct {
	exec.QueryConfig
	Debug  bool
	Region string
	Silent bool
}

// Athenai is a main struct to run this app.
type Athenai struct {
	in  io.Reader
	out io.Writer
	rl  readlineCloser

	cfg    *Config
	client athenaiface.AthenaAPI
	// tick interval
	interval time.Duration

	resultCh chan *exec.Result
	errCh    chan error
	doneCh   chan struct{}
	wg       sync.WaitGroup
}

// New creates a new Athena.
func New(out io.Writer, cfg *Config) *Athenai {
	a := &Athenai{
		in:  os.Stdin,
		out: out,
		cfg: cfg,
		// TODO: client should be passed via function arguments for easy testing
		client:   newClient(cfg),
		interval: tickInterval,
		resultCh: make(chan *exec.Result, 1),
		errCh:    make(chan error, 1),
		doneCh:   make(chan struct{}),
	}
	return a
}

func (a *Athenai) print(x ...interface{}) {
	fmt.Fprint(a.out, x...)
}

func (a *Athenai) println(x ...interface{}) {
	fmt.Fprintln(a.out, x...)
}

// showProgressMsg shows progress messages while queries are being executed.
func (a *Athenai) showProgressMsg(ctx context.Context) {
	a.print("Running query")
	tick := time.Tick(a.interval)
	for {
		select {
		case <-tick:
			a.print(".")
		case <-ctx.Done():
			return
		}
	}
}

// runSingleQuery runs a single query. `query` must be a single SQL statement.
func (a *Athenai) runSingleQuery(query string) {
	// Run a query, and send results or an error
	r, err := exec.NewQuery(a.client, query, &a.cfg.QueryConfig).Run()
	if err != nil {
		a.errCh <- err
	} else {
		a.resultCh <- r
	}
}

// monitorComplete waits for all query executions and notifies `doneCh` when they are all complete.
func (a *Athenai) monitorComplete() {
	a.wg.Wait()
	a.doneCh <- struct{}{}
}

// RunQuery runs the given query.
// It splits the query by semicolons and run each statement concurrently.
// It skips empty statements.
func (a *Athenai) RunQuery(query string) {
	// Split statements
	stmts := splitStmts(query)
	if len(stmts) == 0 {
		a.println("Nothing executed")
		return
	}

	// Print progress messages
	if !a.cfg.Silent {
		ctx, cancel := context.WithCancel(context.Background())
		// Stop printing when this method finishes
		defer cancel()
		go a.showProgressMsg(ctx)
	}

	// Run each statement concurrently
	var wg sync.WaitGroup
	wg.Add(len(stmts))
	for _, stmt := range stmts {
		go func(query string) {
			a.runSingleQuery(query)
			wg.Done()
		}(stmt) // Copy stmt to use in goroutines
	}

	doneCh := make(chan struct{})

	// Monitoring goroutine to wait for all query executions and notifies `doneCh` when they are all complete.
	go func() {
		wg.Wait()
		doneCh <- struct{}{}
	}()

	for {
		select {
		case r := <-a.resultCh:
			a.print("\n")
			print.NewTable(a.out).Print(r)
		case e := <-a.errCh:
			a.print("\n")
			fmt.Fprintln(os.Stderr, e)
		case <-doneCh:
			return
		}
	}
}

func (a *Athenai) setupREPL() error {
	// rl already exists, no need to setup
	if a.rl != nil {
		return nil
	}

	rl, err := readline.NewEx(&readline.Config{
		Prompt:            "athenai> ",
		HistoryFile:       filepath.Join(os.TempDir(), ".athenai_history"),
		HistorySearchFold: true,
		Stdin:             a.in,
		Stdout:            a.out,
	})
	if err != nil {
		return err
	}

	a.rl = rl
	return nil
}

// RunREPL runs REPL mode (interactive mode).
func (a *Athenai) RunREPL() error {
	if err := a.setupREPL(); err != nil {
		return errors.Wrap(err, "failed to setup REPL")
	}
	defer a.rl.Close()

	for {
		// Read a line from stdin
		query, err := a.rl.Readline()
		if err != nil {
			switch err {
			case readline.ErrInterrupt:
				if query == "" {
					// Exit REPL if ^C is pressed at empty line
					return nil
				}
				// Show guide message to exit and continue
				a.println("To exit, press Ctrl-C again or Ctrl-D")
				continue
			case io.EOF:
				// Exit if ^D is pressed
				return nil
			default:
				fmt.Fprintln(os.Stderr, "Error reading line:", err)
			}
		}

		// Ignore empty input
		if query == "" {
			continue
		}

		// Run the query
		a.RunQuery(query)
	}
}
