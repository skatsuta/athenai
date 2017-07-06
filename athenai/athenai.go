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
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/athena"
	"github.com/aws/aws-sdk-go/service/athena/athenaiface"
	"github.com/chzyer/readline"
	"github.com/google/btree"
	"github.com/peco/peco/line"
	"github.com/pkg/errors"
	"github.com/skatsuta/athenai/exec"
	"github.com/skatsuta/athenai/filter"
	"github.com/skatsuta/athenai/print"
	"github.com/skatsuta/spinner"
)

const (
	refreshInterval = 100 * time.Millisecond

	filePrefix = "file://"

	noStmtFound = "No SQL statements found to execute"

	runningQueryMsg    = "Running query..."
	loadingHistoryMsg  = "Loading history..."
	fetchingResultsMsg = "Fetching results..."
	cancelingMsg       = "Canceling..."
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
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer

	rl      readlineCloser
	f       filter.Filter
	printer print.Printer

	client athenaiface.AthenaAPI
	cfg    *Config

	refreshInterval time.Duration
	waitInterval    time.Duration

	mu       sync.RWMutex
	signalCh chan os.Signal
}

// New creates a new Athena.
func New(client athenaiface.AthenaAPI, cfg *Config, out io.Writer) *Athenai {
	out = &safeWriter{w: out}
	a := &Athenai{
		stdin:           os.Stdin,
		stdout:          out,
		stderr:          &safeWriter{w: os.Stderr},
		printer:         createPrinter(out, cfg),
		cfg:             cfg,
		client:          client,
		refreshInterval: refreshInterval,
		waitInterval:    exec.DefaultWaitInterval,
		signalCh:        make(chan os.Signal, 1),
	}
	return a
}

// WithStderr sets stderr to a.
func (a *Athenai) WithStderr(stderr io.Writer) *Athenai {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.stderr = &safeWriter{w: stderr}
	return a
}

// WithWaitInterval sets wait interval to a.
func (a *Athenai) WithWaitInterval(interval time.Duration) *Athenai {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.waitInterval = interval
	return a
}

func (a *Athenai) print(x ...interface{}) {
	fmt.Fprint(a.stdout, x...)
}

func (a *Athenai) println(x ...interface{}) {
	fmt.Fprintln(a.stdout, x...)
}

// showProgressMsg shows a given progress message until a context is canceled.
func (a *Athenai) showProgressMsg(ctx context.Context, msg string) {
	s := spinner.New(spinnerChars, a.refreshInterval)
	s.Writer = a.stdout
	s.Suffix = " " + msg
	s.Start()
	<-ctx.Done() // Wait until ctx is done
	s.Stop()
}

// runSingleQuery runs a single query. `query` must be a single SQL statement.
func (a *Athenai) runSingleQuery(ctx context.Context, query string, rcCh chan *ResultContainer) {
	// Run a query, and send results or an error
	log.Printf("Start running %q\n", query)
	q := exec.NewQuery(a.client, a.cfg.QueryConfig(), query).WithWaitInterval(a.waitInterval)
	r, err := q.Run(ctx)
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
		a.printErr(err, "query execution failed")
	}
}

// RunQuery runs the given queries.
// It splits each statement by semicolons and run them concurrently.
// It skips empty statements.
func (a *Athenai) RunQuery(queries ...string) {
	// Trap SIGINT signal
	signal.Notify(a.signalCh, os.Interrupt)
	// Context to propagate cancellation initiated by user
	userCancelCtx, userCancelFunc := context.WithCancel(context.Background())
	// Context to notify cancellation process is complete
	cancelingCtx, cancelingFunc := context.WithCancel(context.Background())
	defer func() {
		userCancelFunc()
		cancelingFunc()
	}()

	canceledCh := make(chan struct{})

	// Watcher goroutine to cancel query executions
	go func() {
		select {
		case <-a.signalCh: // User has canceled query executions
			log.Println("Starting cancellation initiated by user")
			userCancelFunc()
			a.print("\n")
			if !a.cfg.Silent {
				go a.showProgressMsg(cancelingCtx, cancelingMsg)
			}
			canceledCh <- struct{}{}
		case <-userCancelCtx.Done(): // Exit normally
		}
	}()

	// Split SQL statements
	stmts := a.splitStmts(queries)
	l := len(stmts)
	log.Printf("%d SQL statements to execute: %#v\n", l, stmts)
	if l == 0 {
		a.println(noStmtFound)
		return
	}

	// Run each statement concurrently
	rcChs := make([]chan *ResultContainer, l)
	var wg sync.WaitGroup
	wg.Add(l)
	for i, stmt := range stmts {
		rcCh := make(chan *ResultContainer, 1)
		rcChs[i] = rcCh
		go func(query string) {
			a.runSingleQuery(userCancelCtx, query, rcCh)
			wg.Done()
		}(stmt) // Capture stmt locally in order to use it in goroutines
	}

	// Print progress messages
	if !a.cfg.Silent {
		go a.showProgressMsg(userCancelCtx, runningQueryMsg)
	}

	go func() {
		wg.Wait()
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
		Stdin:             a.stdin,
		Stdout:            a.stdout,
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
				a.printErr(err, "error reading line")
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

// fetchQueryExecutions fetches query executions and returns them being sorted by submission date
// in the descending order.
func (a *Athenai) fetchQueryExecutions(ctx context.Context) ([]*athena.QueryExecution, error) {
	lqx, err := a.client.ListQueryExecutionsWithContext(ctx, &athena.ListQueryExecutionsInput{})
	if err != nil {
		return nil, errors.Wrap(err, "ListQueryExecutions API error")
	}

	bgqx, err := a.client.BatchGetQueryExecutionWithContext(ctx, &athena.BatchGetQueryExecutionInput{
		QueryExecutionIds: lqx.QueryExecutionIds,
	})
	if err != nil {
		return nil, errors.Wrap(err, "BatchGetQueryExecution API error")
	}

	qxs := bgqx.QueryExecutions
	sort.Slice(qxs, func(i, j int) bool {
		// Sort by SubmissionDateTime in descending order
		return qxs[i].Status.SubmissionDateTime.After(*qxs[j].Status.SubmissionDateTime)
	})

	return qxs, nil
}

func (a *Athenai) filterQueryExecutions(qxs []*athena.QueryExecution) ([]*athena.QueryExecution, error) {
	entryMap := make(map[string]*athena.QueryExecution, len(qxs))
	entries := make([]string, 0, len(qxs))
	for _, qx := range qxs {
		if aws.StringValue(qx.Status.State) != athena.QueryExecutionStateSucceeded {
			// Skip if not succeeded
			continue
		}
		entry := generateEntry(qx)
		entryMap[entry] = qx
		entries = append(entries, entry)
	}

	history := strings.Join(entries, "\n")
	a.f.SetInput(history)

	err := a.f.Run(context.Background())
	if err != nil && !strings.Contains(err.Error(), "collect results") {
		return nil, errors.Wrap(err, "error filtering query executions")
	}

	s := a.f.Selection()
	if s.Len() == 0 {
		if l, err := a.f.CurrentLineBuffer().LineAt(a.f.Location().LineNumber()); err == nil {
			s.Add(l)
		}
	}

	selectedQxs := make([]*athena.QueryExecution, 0, s.Len())
	s.Ascend(func(it btree.Item) bool {
		if entry, ok := entryMap[it.(line.Line).Output()]; ok {
			selectedQxs = append(selectedQxs, entry)
		}
		return true
	})
	return selectedQxs, nil
}

func (a *Athenai) selectQueryExecutions(ctx context.Context) ([]*athena.QueryExecution, error) {
	a.mu.Lock()
	if a.f == nil {
		a.f = filter.New()
	}
	a.mu.Unlock()

	loadingCtx, cancel := context.WithCancel(ctx)
	defer cancel() // Ensure to cancel

	// Print loading messages
	if !a.cfg.Silent {
		go a.showProgressMsg(loadingCtx, loadingHistoryMsg)
	}

	qxs, err := a.fetchQueryExecutions(loadingCtx)
	if err != nil {
		return nil, errors.Wrap(err, "error fetching query executions")
	}

	// Stop printing loading messages
	cancel()

	selectedQxs, err := a.filterQueryExecutions(qxs)
	if err != nil && !strings.Contains(err.Error(), "canceled") { // Ignore user-canceled error
		return nil, errors.Wrap(err, "error selecting query executions")
	}

	return selectedQxs, nil
}

// fetchQueryResults fetches query results of qx and send them to rcCh.
func (a *Athenai) fetchQueryResults(ctx context.Context, qx *athena.QueryExecution, rcCh chan *ResultContainer) {
	log.Printf("Start fetching query results of QueryExecutionId %s\n", aws.StringValue(qx.QueryExecutionId))
	q := exec.NewQueryFromQx(a.client, a.cfg.QueryConfig(), qx).WithWaitInterval(a.waitInterval)
	if err := q.GetResults(ctx); err != nil {
		rcCh <- &ResultContainer{Err: err}
	} else {
		rcCh <- &ResultContainer{Result: q.Result}
	}
}

// ShowResults shows results of completed query executions.
func (a *Athenai) ShowResults() {
	// Trap SIGINT signal
	signal.Notify(a.signalCh, os.Interrupt)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	canceledCh := make(chan struct{})

	// Watch user-initiated cancellation
	go func() {
		select {
		case <-a.signalCh: // User has canceled query executions
			log.Println("Starting cancellation initiated by user")
			cancel()
			canceledCh <- struct{}{}
		case <-ctx.Done(): // Exit normally
		}
	}()

	qxs, err := a.selectQueryExecutions(ctx)
	if err != nil {
		a.print("\n")
		if !strings.Contains(err.Error(), "canceled") { // Ignore user-canceled error
			a.printErr(err, "error selecting query executions")
		}
		return
	}

	// Print messages while fetching query results
	if !a.cfg.Silent {
		a.print("\n")
		go a.showProgressMsg(ctx, fetchingResultsMsg)
	}

	// Get each query result concurrently
	l := len(qxs)
	rcChs := make([]chan *ResultContainer, l)
	var wg sync.WaitGroup
	wg.Add(l)
	for i, qx := range qxs {
		rcCh := make(chan *ResultContainer, 1)
		rcChs[i] = rcCh
		go func(qx *athena.QueryExecution) {
			a.fetchQueryResults(ctx, qx, rcCh)
			wg.Done()
		}(qx) // Capture locally in order to use it in goroutines
	}

	go func() {
		wg.Wait()
		cancel() // All results have been fetched; Stop showing the progress messages
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

	log.Println("Fetched all query results")
}

func (a *Athenai) printErr(err error, message string) {
	fmt.Fprintf(a.stderr, "Error: %s: %s\n", message, err)
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
func (a *Athenai) splitStmts(args []string) []string {
	stmts := make([]string, 0, len(args))

	for _, arg := range args {
		arg := arg // Capture locally
		if strings.HasPrefix(arg, filePrefix) {
			log.Printf("%q prefix found in %q, reading its contents from file\n", filePrefix, arg)
			var err error
			arg, err = readFile(arg)
			if err != nil {
				a.printErr(err, "failed to read file")
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

func createPrinter(out io.Writer, cfg *Config) print.Printer {
	switch cfg.Format {
	case "csv":
		return print.NewCSV(out)
	default:
		return print.NewTable(out)
	}
}

func generateEntry(qx *athena.QueryExecution) string {
	query := aws.StringValue(qx.Query)
	if strings.Contains(query, "\n") {
		// Serialize a multi-line single query
		query = strings.Join(strings.Split(query, "\n"), " ")
	}

	entry := fmt.Sprintf("%s\t%s\t%s\t%.2f seconds\t%s",
		qx.Status.SubmissionDateTime,
		query,
		aws.StringValue(qx.Status.State),
		float64(aws.Int64Value(qx.Statistics.EngineExecutionTimeInMillis))/1000,
		print.FormatBytes(aws.Int64Value(qx.Statistics.DataScannedInBytes)),
	)
	return entry
}
