package print

import "github.com/aws/aws-sdk-go/service/athena"

// Result represents an interface that holds information of a query execution and its results.
type Result interface {
	Info() *athena.QueryExecution
	Rows() [][]string
}

// mockedResult is a mock struct which implements Result interface for testing.
type mockedResult struct {
	info *athena.QueryExecution
	data [][]string
}

func (m *mockedResult) Info() *athena.QueryExecution {
	return m.info
}

func (m *mockedResult) Rows() [][]string {
	return m.data
}
