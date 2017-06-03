package athenai

import (
	"io/ioutil"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/athena"
	"github.com/skatsuta/athenai/exec"
)

// Config is a configuration information for Athenai.
type Config struct {
	exec.QueryConfig
	Debug  bool
	Region string
	Silent bool
}

// Athenai is a main struct to run this app.
type Athenai struct {
	cfg    *Config
	client *athena.Athena
}

// New creates a new Athena.
func New(cfg *Config) *Athenai {
	if cfg == nil {
		cfg = &Config{}
	}

	a := &Athenai{
		cfg:    cfg,
		client: newClient(cfg),
	}
	return a
}

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
