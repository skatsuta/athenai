package stub

import (
	"fmt"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/athena"
	"github.com/aws/aws-sdk-go/service/athena/athenaiface"
	"github.com/pkg/errors"
	"github.com/skatsuta/athenai/internal/testhelper"
)

// Result represents a stub result of a query execution.
type Result struct {
	ID           string
	Query        string
	ExecTime     int64
	ScannedBytes int64
	athena.ResultSet
	ErrMsg string
}

// StartQueryExecutionStub simulates StartQueryExecution API.
type StartQueryExecutionStub struct {
	athenaiface.AthenaAPI
	results map[string]*Result // map[query]*Result
}

// NewStartQueryExecutionStub creates a new StartQueryExecutionStub which returns stub responses
// based on rs.
func NewStartQueryExecutionStub(rs ...*Result) *StartQueryExecutionStub {
	results := make(map[string]*Result, len(rs))
	for _, r := range rs {
		results[r.Query] = r
	}
	return &StartQueryExecutionStub{results: results}
}

// StartQueryExecution runs the SQL query statements contained in the Query string.
// It returns an error if a query other than SELECT, SHOW or DESCRIBE statement is given.
func (s *StartQueryExecutionStub) StartQueryExecution(input *athena.StartQueryExecutionInput) (*athena.StartQueryExecutionOutput, error) {
	query := aws.StringValue(input.QueryString)
	r, ok := s.results[query]
	if !ok {
		return nil, errors.Errorf("InvalidRequestException: %q is an unexpected query", query)
	}
	for _, kwd := range []string{"SELECT", "SHOW", "DESCRIBE"} {
		if !strings.HasPrefix(query, kwd) {
			continue
		}
		resp := &athena.StartQueryExecutionOutput{QueryExecutionId: &r.ID}
		return resp, nil
	}
	return nil, errors.Errorf("InvalidRequestException: %q is not an allowed statement", query)
}

var (
	successfulQueryStateFlow = []string{
		athena.QueryExecutionStateQueued,
		athena.QueryExecutionStateRunning,
		athena.QueryExecutionStateSucceeded,
	}

	failedQueryStateFlow = []string{
		athena.QueryExecutionStateQueued,
		athena.QueryExecutionStateRunning,
		athena.QueryExecutionStateFailed,
	}
)

// GetQueryExecutionStub simulates GetQueryExecution API.
type GetQueryExecutionStub struct {
	athenaiface.AthenaAPI
	mu             sync.RWMutex
	queryStateFlow []string
	results        map[string]*Result // map[id]*Result
	stateCnts      map[string]int     // map[id]count
}

// newGetQueryExecutionStub creates a new GetQueryExecutionStub which returns stub responses
// based on rs with queryStateFlow states.
func newGetQueryExecutionStub(queryStateFlow []string, rs ...*Result) *GetQueryExecutionStub {
	l := len(rs)
	results := make(map[string]*Result, l)
	stateCnts := make(map[string]int, l)
	for _, r := range rs {
		results[r.ID] = r
		stateCnts[r.ID] = 0
	}
	return &GetQueryExecutionStub{
		queryStateFlow: queryStateFlow,
		results:        results,
		stateCnts:      stateCnts,
	}
}

// NewGetQueryExecutionStub creates a new GetQueryExecutionStub which returns stub responses
// based on rs with successful query states in order.
func NewGetQueryExecutionStub(rs ...*Result) *GetQueryExecutionStub {
	return newGetQueryExecutionStub(successfulQueryStateFlow, rs...)
}

// NewGetFailedQueryExecutionStub creates a new GetQueryExecutionStub which returns stub responses
// based on rs with failed query states in order.
func NewGetFailedQueryExecutionStub(rs ...*Result) *GetQueryExecutionStub {
	return newGetQueryExecutionStub(failedQueryStateFlow, rs...)
}

// GetQueryExecution returns information about a single execution of a query.
func (s *GetQueryExecutionStub) GetQueryExecution(input *athena.GetQueryExecutionInput) (*athena.GetQueryExecutionOutput, error) {
	id := aws.StringValue(input.QueryExecutionId)
	r, ok := s.results[id]
	if !ok {
		return nil, errors.Errorf("InvalidRequestException: QueryExecution %s was not found", id)
	}
	if r.ErrMsg != "" {
		return nil, errors.New(r.ErrMsg)
	}

	s.mu.RLock()
	cnt := s.stateCnts[id]
	s.mu.RUnlock()

	l := len(s.queryStateFlow)
	state := s.queryStateFlow[l-1]
	if cnt < l {
		state = s.queryStateFlow[cnt]
	}

	s.mu.Lock()
	s.stateCnts[id]++
	s.mu.Unlock()

	resp := &athena.GetQueryExecutionOutput{
		QueryExecution: &athena.QueryExecution{
			QueryExecutionId: &r.ID,
			Query:            &r.Query,
			Statistics:       testhelper.CreateStats(r.ExecTime, r.ScannedBytes),
			Status:           &athena.QueryExecutionStatus{State: &state},
		},
	}
	return resp, nil
}

// GetQueryResultsStub simulates GetQueryResults and GetQueryResultsPages API.
type GetQueryResultsStub struct {
	athenaiface.AthenaAPI
	MaxPages int
	mu       sync.Mutex
	pages    map[string]int     // map[id]page
	results  map[string]*Result // map[id]*Result
}

// NewGetQueryResultsStub creates a new GetQueryResultsStub which returns stub responses
// based on rs.
func NewGetQueryResultsStub(rs ...*Result) *GetQueryResultsStub {
	l := len(rs)
	results := make(map[string]*Result, l)
	pages := make(map[string]int, l)
	for _, r := range rs {
		results[r.ID] = r
		pages[r.ID] = 0
	}
	return &GetQueryResultsStub{
		MaxPages: 1,
		results:  results,
		pages:    pages,
	}
}

// GetQueryResults returns the results of a single query execution specified by QueryExecutionId.
func (s *GetQueryResultsStub) GetQueryResults(input *athena.GetQueryResultsInput) (*athena.GetQueryResultsOutput, error) {
	id := aws.StringValue(input.QueryExecutionId)
	r, ok := s.results[id]
	if !ok {
		return nil, errors.Errorf("InvalidRequestException: QueryExecution %s was not found", id)
	}
	if r.ErrMsg != "" {
		return nil, errors.New(r.ErrMsg)
	}

	s.mu.Lock()
	s.pages[id]++
	page := s.pages[id]
	s.mu.Unlock()

	resp := &athena.GetQueryResultsOutput{ResultSet: &r.ResultSet}
	if page < s.MaxPages {
		resp.SetNextToken(fmt.Sprintf("NextToken%d", page))
	}
	return resp, nil
}

// GetQueryResultsPages iterates over the pages of a GetQueryResults operation, calling the callback function with the response data for each page.
func (s *GetQueryResultsStub) GetQueryResultsPages(input *athena.GetQueryResultsInput, callback func(*athena.GetQueryResultsOutput, bool) bool) error {
	cont := true
	for cont {
		qr, err := s.GetQueryResults(input)
		if err != nil {
			return err
		}
		lastPage := qr.NextToken == nil
		cont = callback(qr, lastPage)
		cont = cont && !lastPage
	}
	return nil
}

// Client is a stub of Athena client.
type Client struct {
	athenaiface.AthenaAPI
	*StartQueryExecutionStub
	*GetQueryExecutionStub
	*GetQueryResultsStub
}

// NewClient returns a new Athena client which returns stub API responses based on rs.
func NewClient(rs ...*Result) *Client {
	return &Client{
		StartQueryExecutionStub: NewStartQueryExecutionStub(rs...),
		GetQueryExecutionStub:   NewGetQueryExecutionStub(rs...),
		GetQueryResultsStub:     NewGetQueryResultsStub(rs...),
	}
}

// StartQueryExecution runs the SQL query statements contained in the Query string.
func (s *Client) StartQueryExecution(input *athena.StartQueryExecutionInput) (*athena.StartQueryExecutionOutput, error) {
	return s.StartQueryExecutionStub.StartQueryExecution(input)
}

// GetQueryExecution returns information about a single execution of a query.
func (s *Client) GetQueryExecution(input *athena.GetQueryExecutionInput) (*athena.GetQueryExecutionOutput, error) {
	return s.GetQueryExecutionStub.GetQueryExecution(input)
}

// GetQueryResults returns the results of a single query execution specified by QueryExecutionId.
func (s *Client) GetQueryResults(input *athena.GetQueryResultsInput) (*athena.GetQueryResultsOutput, error) {
	return s.GetQueryResultsStub.GetQueryResults(input)
}

// GetQueryResultsPages iterates over the pages of a GetQueryResults operation, calling the callback function with the response data for each page.
func (s *Client) GetQueryResultsPages(input *athena.GetQueryResultsInput, callback func(*athena.GetQueryResultsOutput, bool) bool) error {
	return s.GetQueryResultsStub.GetQueryResultsPages(input, callback)
}
