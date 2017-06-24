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

	noStmtFound = "No SQL statements found to execute"

	runningQueryMsg   = "Running query..."
	cancelingQueryMsg = "Canceling query..."
)

var spinnerChars = []string{"⠋", "⠙", "⠚", "⠞", "⠖", "⠦", "⠴", "⠲", "⠳", "⠓"}

type safeWriter struct {
	mu sync.Mutex
	w  io.Writer
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

// ResultContainer is a container which has a query result or an error.
type ResultContainer struct {
	Result print.Result
	Err    error
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

	mu       sync.RWMutex
	wg       sync.WaitGroup
	signalCh chan os.Signal
}

// New creates a new Athena.
func New(client athenaiface.AthenaAPI, out io.Writer, cfg *Config) *Athenai {
	out = &safeWriter{w: out}
	a := &Athenai{
		in:       os.Stdin,
		out:      out,
		printer:  print.NewTable(out),
		cfg:      cfg,
		client:   client,
		interval: refreshInterval,
		signalCh: make(chan os.Signal, 1),
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

// showProgressMsg shows a given progress message until a context is canceled.
func (a *Athenai) showProgressMsg(ctx context.Context, msg string) {
	s := spinner.New(spinnerChars, a.interval)
	s.Writer = a.out
	s.Suffix = " " + msg
	s.Start()
	<-ctx.Done() // Wait until ctx is done
	s.Stop()
}

// runSingleQuery runs a single query. `query` must be a single SQL statement.
func (a *Athenai) runSingleQuery(ctx context.Context, query string, rcCh chan *ResultContainer) {
	// Run a query, and send results or an error
	log.Printf("Start running %q\n", query)
	r, err := exec.NewQuery(a.client, a.cfg.QueryConfig(), query).Run(ctx)
	if err != nil {
		rcCh <- &ResultContainer{Err: err}
	} else {
		rcCh <- &ResultContainer{Result: r}
	}
}

func (a *Athenai) printResultOrErr(rc *ResultContainer) {
	if r := rc.Result; r != nil {
		a.print("\n")
		a.printer.Print(r)
		return
	}

	err := rc.Err
	cause := errors.Cause(err)
	switch e := cause.(type) {
	case *exec.CanceledError:
		log.Println(e) // Just log the error
	default:
		printErr(err, "query execution failed")
	}
}

// RunQuery runs the given queries.
// It splits each statement by semicolons and run them concurrently.
// It skips empty statements.
func (a *Athenai) RunQuery(queries ...string) {
	// Trap SIGINT signal and prepare a context
	signal.Notify(a.signalCh, os.Interrupt)
	// Context to propagate cancellation initiated by user
	userCancelCtx, userCancelFunc := context.WithCancel(context.Background())
	// Context to notify cancellation process is complete
	cancelingCtx, cancelingFunc := context.WithCancel(context.Background())
	defer cancelingFunc()

	canceledCh := make(chan struct{})

	// Watcher goroutine to cancel query executions
	go func() {
		select {
		case <-a.signalCh: // User has canceled query executions
			log.Println("Starting cancellation initiated by user")
			userCancelFunc()
			a.print("\n")
			if !a.cfg.Silent {
				go a.showProgressMsg(cancelingCtx, cancelingQueryMsg)
			}
			canceledCh <- struct{}{}
		case <-userCancelCtx.Done(): // Exit normally
		}
	}()

	// Split SQL statements
	stmts := splitStmts(queries)
	l := len(stmts)
	log.Printf("%d SQL statements to execute: %#v\n", l, stmts)
	if l == 0 {
		a.println(noStmtFound)
		return
	}

	// Run each statement concurrently
	rcChs := make([]chan *ResultContainer, l)
	a.wg.Add(l)
	for i, stmt := range stmts {
		rcCh := make(chan *ResultContainer, 1)

		go func(query string) {
			a.runSingleQuery(userCancelCtx, query, rcCh)
			a.wg.Done()
		}(stmt) // Capture stmt locally in order to use it in goroutines

		rcChs[i] = rcCh
	}

	// Print progress messages
	if !a.cfg.Silent {
		go a.showProgressMsg(userCancelCtx, runningQueryMsg)
	}

	go func() {
		a.wg.Wait()
		userCancelFunc() // All executions have been completed; Stop showing the progress messages
		signal.Stop(a.signalCh)
	}()

	for _, rcCh := range rcChs {
		select {
		case <-canceledCh: // Stop showing results if canceled
			a.print("\n")
			return
		default:
			a.printResultOrErr(<-rcCh)
		}
	}

	log.Println("All query executions have been completed")
	a.print("\n")
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
		log.Printf("Given input: %q\n", query)
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
		return "", err
	}
	c := string(content)
	log.Printf(`Content of %s:
--------------------
%s
--------------------
`, filename, c)
	return c, nil
}

// splitStmts splits SQL statements contained in args by semicolons and flattens them.
// It drops empty statements.
//
// If an argument has `file://` prefix, splitStmts reads the file content
// and splits each statement as well.
// If it encounters errors while reading files, it just prints the errors on stderr and ignores them.
func splitStmts(args []string) []string {
	stmts := make([]string, 0, len(args))

	for _, arg := range args {
		arg := arg // Capture locally
		if strings.HasPrefix(arg, filePrefix) {
			log.Printf("%q prefix found in %q, reading its contents from file\n", filePrefix, arg)
			var err error
			arg, err = readFile(arg)
			if err != nil {
				printErr(err, "failed to read file")
				continue
			}
		}

		splitted := strings.Split(arg, ";")
		for _, s := range splitted {
			stmt := strings.TrimSpace(s)
			if stmt != "" {
				stmts = append(stmts, stmt)
			}
		}
	}

	return stmts
}
