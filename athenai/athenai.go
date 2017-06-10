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

	"github.com/aws/aws-sdk-go/service/athena/athenaiface"
	"github.com/chzyer/readline"
	"github.com/pkg/errors"
	"github.com/skatsuta/athenai/exec"
	"github.com/skatsuta/athenai/print"
)

const (
	tickInterval = 1000 * time.Millisecond

	filePrefix = "file://"

	noStmtFound = "No SQL statements found to run"
)

// readlineCloser is an interface to read every line in REPL and then close it.
type readlineCloser interface {
	Readline() (string, error)
	Close() error
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
	mu       sync.Mutex
}

// New creates a new Athena.
func New(client athenaiface.AthenaAPI, out io.Writer, cfg *Config) *Athenai {
	a := &Athenai{
		in:       os.Stdin,
		out:      out,
		cfg:      cfg,
		client:   client,
		interval: tickInterval,
		resultCh: make(chan *exec.Result, 1),
		errCh:    make(chan error, 1),
		doneCh:   make(chan struct{}),
	}
	return a
}

func (a *Athenai) print(x ...interface{}) {
	a.mu.Lock()
	defer a.mu.Unlock()
	fmt.Fprint(a.out, x...)
}

func (a *Athenai) println(x ...interface{}) {
	a.print(x...)
	a.print("\n")
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
			log.Println("Stopped showing progress messages")
			return
		}
	}
}

// runSingleQuery runs a single query. `query` must be a single SQL statement.
func (a *Athenai) runSingleQuery(query string) {
	// Run a query, and send results or an error
	log.Printf("Start running %q\n", query)
	r, err := exec.NewQuery(a.client, query, a.cfg.QueryConfig()).Run()
	if err != nil {
		a.errCh <- err
	} else {
		a.resultCh <- r
	}
}

// RunQuery runs the given queries.
// It splits each statement by semicolons and run them concurrently.
// It skips empty statements.
func (a *Athenai) RunQuery(queries []string) {
	// Split statements
	stmts, errs := splitStmts(queries)
	if len(errs) > 0 {
		for _, err := range errs {
			printErr(err, "error splitting SQL statements")
		}
	}

	l := len(stmts)
	log.Printf("%d SQL statements found: %#v\n", l, stmts)
	if l == 0 {
		a.println(noStmtFound)
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
	a.wg.Add(len(stmts))
	for _, stmt := range stmts {
		go func(query string) {
			a.runSingleQuery(query)
			a.wg.Done()
		}(stmt) // Capture stmt locally in order to use it in goroutines
	}

	// Monitoring goroutine to wait for the completion of all query executions
	go func() {
		a.wg.Wait()
		a.doneCh <- struct{}{}
	}()

	// Receive results or errors until done
	for {
		select {
		case r := <-a.resultCh:
			a.print("\n")
			print.NewTable(a.out).Print(r)
		case e := <-a.errCh:
			a.print("\n")
			printErr(e, "query execution failed")
		case <-a.doneCh:
			log.Println("All query executions have been completed")
			return
		}
	}
}

func (a *Athenai) setupREPL() error {
	// rl is already set, no need to be setup again
	if a.rl != nil {
		log.Printf("REPL setup has been done already: %#v\n", a.rl)
		return nil
	}

	historyFile := filepath.Join(os.TempDir(), ".athenai_history")
	rl, err := readline.NewEx(&readline.Config{
		Prompt:            "athenai> ",
		HistoryFile:       historyFile,
		HistorySearchFold: true,
		Stdin:             a.in,
		Stdout:            a.out,
	})
	if err != nil {
		return err
	}

	log.Printf("Queries given will be saved to %s\n", historyFile)

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
					log.Println("Ctrl-C is pressed on empty line, exitting REPL")
					return nil
				}
				log.Println("Ctrl-C is pressed on non-empty line, continue to run REPL")
				a.println("To exit, press Ctrl-C again or Ctrl-D")
				continue
			case io.EOF:
				log.Println("Ctrl-D is pressed, exitting REPL")
				return nil
			default:
				printErr(err, "error reading line")
			}
		}

		// Ignore empty input
		if query == "" {
			continue
		}

		// Run the query
		log.Printf("Input given: %q\n", query)
		a.RunQuery([]string{query})
	}
}

func printErr(err error, message string) {
	fmt.Fprintf(os.Stderr, "ERROR: %s: %s\n", message, err)
}

// readFile reads the content of a file whose path has `file://` prefix.
func readFile(arg string) (string, error) {
	filename := strings.TrimPrefix(arg, filePrefix)
	log.Println("Given file name:", filename)
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", errors.Wrap(err, "failed to read file")
	}
	c := string(content)
	log.Printf(`Content of %s:
--------------------
%s
--------------------
`, filename, c)
	return c, nil
}

// splitStmts splits SQL statements in the queries by semicolons and flatten them.
// It drops empty statements.
//
// If an argument has `file://` prefix, splitStmts reads the file content
// and splits each statement as well.
// If it enconters errors while reading files, it returns the errors as the second return value.
func splitStmts(args []string) ([]string, []error) {
	stmts := make([]string, 0, len(args))
	var errs []error

	for _, arg := range args {
		if strings.HasPrefix(arg, filePrefix) {
			log.Printf("%q prefix found in %q, reading its contents from file\n", filePrefix, arg)
			content, err := readFile(arg)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			arg = content
		}

		splitted := strings.Split(arg, ";")
		for _, s := range splitted {
			stmt := strings.TrimSpace(s)
			if stmt != "" {
				// Select non-empty statements
				stmts = append(stmts, stmt)
			}
		}
	}

	return stmts, errs
}
