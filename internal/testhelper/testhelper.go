package testhelper

import (
	"html/template"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/athena"
)

func init() {
	// Initialize random seed
	rand.Seed(time.Now().UnixNano())
}

// CreateRows creates an array of *athena.Row from an array of string arrays.
func CreateRows(rawRows [][]string) []*athena.Row {
	rows := make([]*athena.Row, len(rawRows))
	for i, row := range rawRows {
		r := &athena.Row{Data: make([]*athena.Datum, len(row))}
		for j, data := range row {
			r.Data[j] = new(athena.Datum).SetVarCharValue(data)
		}
		rows[i] = r
	}
	return rows
}

// CreateStats creates a new QueryExecutionStatistics.
func CreateStats(execTime, scannedBytes int64) *athena.QueryExecutionStatistics {
	return &athena.QueryExecutionStatistics{
		EngineExecutionTimeInMillis: aws.Int64(execTime),
		DataScannedInBytes:          aws.Int64(scannedBytes),
	}
}

const configFileTmpl = `
[{{.Section}}]
debug = {{.Debug}}
silent = {{.Silent}}
profile = {{.Profile}}
region = {{.Region}}
database = {{.Database}}
location = {{.Location}}
`

// CreateConfigFile creates a new config file in a tempporary directory based on cfg's data.
// The type of cfg is set to interface{} to avoid cyclic import, but it must be *athenai.Config.
func CreateConfigFile(name string, cfg interface{}) (homeDir string, file *os.File, cleanup func(), err error) {
	// Create a temporary directory for config file
	homeDir = filepath.Join(os.TempDir(), strconv.Itoa(rand.Int()))
	baseDir := filepath.Join(homeDir, ".athenai")
	if err = os.MkdirAll(baseDir, 0755); err != nil {
		return "", nil, nil, err
	}

	filePath := filepath.Join(baseDir, "config")
	file, err = os.Create(filePath)
	if err != nil {
		return homeDir, nil, nil, err
	}
	log.Println("Created a new config file:", filePath)

	err = template.Must(template.New(name).Parse(configFileTmpl)).Execute(file, cfg)
	cleanup = func() {
		file.Close()
		os.Remove(file.Name())
	}
	return homeDir, file, cleanup, err
}
