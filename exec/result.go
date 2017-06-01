package exec

import (
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/athena"
)

// Result represents results of a query execution.
// This struct must implement print.Result interface.
type Result struct {
	mu   sync.RWMutex
	info *athena.QueryExecution
	rs   *athena.ResultSet
}

// Info returns information of a query execution.
func (r *Result) Info() *athena.QueryExecution {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.info
}

func (r *Result) sendRows(ch chan []string) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, rw := range r.rs.Rows {
		row := make([]string, 0, len(rw.Data))
		for _, d := range rw.Data {
			row = append(row, aws.StringValue(d.VarCharValue))
		}
		ch <- row
	}
	close(ch)
}

// Rows returns a receive-only channel which receives each row of query results.
func (r *Result) Rows() <-chan []string {
	ch := make(chan []string)
	go r.sendRows(ch)
	return ch
}
