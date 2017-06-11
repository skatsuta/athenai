package stub

import (
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/athena"
	"github.com/aws/aws-sdk-go/service/athena/athenaiface"
	"github.com/pkg/errors"
	"github.com/skatsuta/athenai/internal/testhelper"
)

// StartQueryExecutionStub simulates StartQueryExecution API.
type StartQueryExecutionStub struct {
	athenaiface.AthenaAPI
	ID string
}

// StartQueryExecution runs the SQL query statements contained in the Query string.
func (s *StartQueryExecutionStub) StartQueryExecution(input *athena.StartQueryExecutionInput) (*athena.StartQueryExecutionOutput, error) {
	query := aws.StringValue(input.QueryString)
	for _, kwd := range []string{"SELECT", "SHOW", "DESCRIBE"} {
		if strings.HasPrefix(query, kwd) {
			resp := &athena.StartQueryExecutionOutput{
				QueryExecutionId: &s.ID,
			}
			return resp, nil
		}
	}
	return nil, errors.Errorf("InvalidRequestException: %q", query)
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
	athena.QueryExecution
	ErrMsg         string
	queryStateFlow []string
	stateCnt       int
}

// NewGetQueryExecutionStub creates a new GetQueryExecution which returns successful query states in order.
func NewGetQueryExecutionStub() *GetQueryExecutionStub {
	return &GetQueryExecutionStub{
		queryStateFlow: successfulQueryStateFlow,
	}
}

// NewGetFailedQueryExecution creates a new GetQueryExecutionStub which returns failed query states in order.
func NewGetFailedQueryExecution() *GetQueryExecutionStub {
	return &GetQueryExecutionStub{
		queryStateFlow: failedQueryStateFlow,
	}
}

// GetQueryExecution returns information about a single execution of a query.
func (s *GetQueryExecutionStub) GetQueryExecution(input *athena.GetQueryExecutionInput) (*athena.GetQueryExecutionOutput, error) {
	if s.ErrMsg != "" {
		return nil, errors.New(s.ErrMsg)
	}

	l := len(s.queryStateFlow)
	state := s.queryStateFlow[l-1]
	if s.stateCnt < l {
		state = s.queryStateFlow[s.stateCnt]
	}

	s.stateCnt++

	if s.QueryExecution.Status == nil {
		s.QueryExecution.SetStatus(&athena.QueryExecutionStatus{})
	}
	s.QueryExecution.Status.SetState(state)
	resp := &athena.GetQueryExecutionOutput{
		QueryExecution: &s.QueryExecution,
	}
	return resp, nil
}

// GetQueryResultsStub simulates GetQueryResults and GetQueryResultsPages API.
type GetQueryResultsStub struct {
	athenaiface.AthenaAPI
	athena.ResultSet
	ErrMsg   string
	MaxPages int
	page     int
}

// GetQueryResults returns the results of a single query execution specified by QueryExecutionId.
func (s *GetQueryResultsStub) GetQueryResults(input *athena.GetQueryResultsInput) (*athena.GetQueryResultsOutput, error) {
	if s.ErrMsg != "" {
		return nil, errors.New(s.ErrMsg)
	}

	s.page++
	resp := &athena.GetQueryResultsOutput{
		ResultSet: &s.ResultSet,
	}
	if s.page < s.MaxPages {
		resp.SetNextToken("next")
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

// Client is a mock of Athena client.
type Client struct {
	athenaiface.AthenaAPI
	StartQueryExecutionStub
	GetQueryExecutionStub
	GetQueryResultsStub
}

// NewClient returns a new MockedClient.
func NewClient(id string) *Client {
	return &Client{
		StartQueryExecutionStub: StartQueryExecutionStub{ID: id},
		GetQueryExecutionStub:   GetQueryExecutionStub{queryStateFlow: successfulQueryStateFlow},
		GetQueryResultsStub:     GetQueryResultsStub{MaxPages: 1},
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

// WithResultSet sets rs to s.
func (s *Client) WithResultSet(rs athena.ResultSet) *Client {
	s.ResultSet = rs
	return s
}

// WithStats sets statistics data to s.
func (s *Client) WithStats(execTime, scannedBytes int64) *Client {
	stats := testhelper.CreateStats(execTime, scannedBytes)
	s.QueryExecution.SetStatistics(stats)
	return s
}

// WithQuery sets query to s.
func (s *Client) WithQuery(query string) *Client {
	s.QueryExecution.SetQuery(strings.TrimSuffix(query, ";"))
	return s
}
