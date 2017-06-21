package athenai

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/service/athena/athenaiface"
	"github.com/chzyer/readline"
	"github.com/pkg/errors"
	"github.com/skatsuta/athenai/exec"
	"github.com/skatsuta/athenai/print"
	"github.com/skatsuta/spinner"
)

const (
	refreshInterval = 100 * time.Millisecond

	filePrefix = "file://"

	noStmtFound = "No SQL statements found to run"
)

var spinnerChars = []string{"⠋", "⠙", "⠚", "⠞", "⠖", "⠦", "⠴", "⠲", "⠳", "⠓"}

type safeWriter struct {
	w  io.Writer
	mu sync.Mutex
}

func (sw *safeWriter) Write(p []byte) (int, error) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	return sw.w.Write(p)
}

// readlineCloser is an interface to read every line in REPL and then close it.
type readlineCloser interface {
	Readline() (string, error)
	Close() error
}

// Athenai is a main struct to run this app.
type Athenai struct {
	in      io.Reader
	out     io.Writer
	rl      readlineCloser
	printer print.Printer

	cfg      *Config
	client   athenaiface.AthenaAPI
	interval time.Duration

	resultChs []chan print.Result
	errChs    []chan error
	doneCh    chan struct{}
	wg        sync.WaitGroup
	mu        sync.RWMutex
}

// New creates a new Athena.
func New(client athenaiface.AthenaAPI, out io.Writer, cfg *Config) *Athenai {
	out = &safeWriter{w: out}
	a := &Athenai{
		in:        os.Stdin,
		out:       out,
		printer:   print.NewTable(out),
		cfg:       cfg,
		client:    client,
		interval:  refreshInterval,
		resultChs: make([]chan print.Result, 0),
		errChs:    make([]chan error, 0),
		doneCh:    make(chan struct{}),
	}
	return a
}

func (a *Athenai) print(x ...interface{}) {
	fmt.Fprint(a.out, x...)
}

func (a *Athenai) println(x ...interface{}) {
	a.print(x...)
	a.print("\n")
}

func (a *Athenai) setupChannels(numStmts int) {
	l := numStmts
	if !a.cfg.Order {
		// Use single channel if arrangement is not needed
		l = 1
	}

	log.Printf("Arranging order: %v. Setting up %d channels\n", a.cfg.Order, l)

	a.mu.Lock()
	for i := 0; i < l; i++ {
		a.resultChs = append(a.resultChs, make(chan print.Result, 1))
		a.errChs = append(a.errChs, make(chan error, 1))
	}
	a.mu.Unlock()
}

// showProgressMsg shows progress messages while queries are being executed.
func (a *Athenai) showProgressMsg(ctx context.Context) {
	s := spinner.New(spinnerChars, a.interval)
	s.Writer = a.out
	s.Suffix = " Running query..."
	s.Start()
	<-ctx.Done()
	s.Stop()
}

// runSingleQuery runs a single query. `query` must be a single SQL statement.
func (a *Athenai) runSingleQuery(ctx context.Context, query string, resultCh chan print.Result, errCh chan error) {
	// Run a query, and send results or an error
	log.Printf("Start running %q\n", query)
	r, err := exec.NewQuery(a.client, a.cfg.QueryConfig(), query).Run(ctx)
	if err != nil {
		errCh <- err
	} else {
		resultCh <- r
	}
}

// RunQuery runs the given queries.
// It splits each statement by semicolons and run them concurrently.
// It skips empty statements.
func (a *Athenai) RunQuery(queries ...string) {
	// Split statements
	stmts, errs := splitStmts(queries)
	if len(errs) > 0 {
		for _, err := range errs {
			printErr(err, "error splitting SQL statements")
		}
	}

	l := len(stmts)
	log.Printf("%d SQL statements to run: %#v\n", l, stmts)
	if l == 0 {
		a.println(noStmtFound)
		return
	}

	a.setupChannels(l)

	// Prepare a context and trap SIGINT signal
	ctx, cancel := context.WithCancel(context.Background())
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt)
	defer func() {
		cancel()
		signal.Stop(signalCh)
	}()

	// Watcher to cancel query executions
	go func() {
		select {
		case <-signalCh:
			cancel()
		case <-ctx.Done(): // Just exit this goroutine
		}
	}()

	// Run each statement concurrently
	a.wg.Add(l)
	for i, stmt := range stmts {
		i := i // Copy locally
		if !a.cfg.Order {
			// Always use the first channel
			i = 0
		}

		go func(query string) {
			a.runSingleQuery(ctx, query, a.resultChs[i], a.errChs[i])
			a.wg.Done()
		}(stmt) // Capture stmt locally in order to use it in goroutines
	}

	// Print progress messages
	if !a.cfg.Silent {
		go a.showProgressMsg(ctx)
	}

	// Monitoring goroutine to wait for the completion of all query executions
	go func() {
		a.wg.Wait()
		a.doneCh <- struct{}{}
	}()

	// Receive results or errors until done
	n := len(a.resultChs)
	for i := 0; i < n; i++ {
	loop:
		for {
			select {
			case r := <-a.resultChs[i]:
				a.print("\n")
				a.printer.Print(r)
				if a.cfg.Order {
					// Go to the next channel
					break loop
				}
			case err := <-a.errChs[i]:
				cause := errors.Cause(err)
				a.print("\n")
				switch e := cause.(type) {
				case *exec.CanceledError:
					a.println(e) // Show as normal message
				default:
					printErr(err, "query execution failed")
				}
				if a.cfg.Order {
					// Go to the next channel
					break loop
				}
			case <-a.doneCh:
				log.Println("All query executions have been completed")
				if !a.cfg.Order {
					// Exit if no more results come into the single channel
					return
				}
			}
		}
	}
}

func (a *Athenai) setupREPL() error {
	// rl is already set, no need to be setup again
	a.mu.RLock()
	if a.rl != nil {
		defer a.mu.RUnlock()
		log.Printf("REPL setup has been done already: %#v\n", a.rl)
		return nil
	}
	a.mu.RUnlock()

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

	log.Printf("Query history will be saved to %s\n", historyFile)

	a.mu.Lock()
	a.rl = rl
	a.mu.Unlock()
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
		a.RunQuery(query)
	}
}

func printErr(err error, message string) {
	fmt.Fprintf(os.Stderr, "Error: %s: %s\n", message, err)
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
