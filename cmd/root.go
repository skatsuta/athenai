package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/athena"
	"github.com/pkg/errors"
	"github.com/skatsuta/athenai/athenai"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	config  = &athenai.Config{}
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "athenai",
	Short: "Athenai is a simple command line tool that accesses Amazon Athena.",
	// TODO
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Enable debug mode
		if !config.Debug {
			log.SetOutput(ioutil.Discard)
		}
	},
}

// ValidationError represents an error that validation before command execution failed.
type ValidationError struct {
	// Cmd is a command name where validation failed.
	Cmd string
	// Name is a configuration name that is not valid. For exapmle, "location" for run command.
	Name string
	// Msg is a message shown to users.
	Msg string
}

func (ve *ValidationError) Error() string {
	return ve.Msg
}

func (ve *ValidationError) String() string {
	return ve.Msg
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fatal(err)
	}
}

func init() {
	// Define global flags
	f := RootCmd.PersistentFlags()
	f.StringVar(&cfgFile, "config", "", "Config file path (default is $HOME/.athenai/config)")
	f.BoolVar(&config.Debug, "debug", false, "Turn on debug logging")
	f.BoolVar(&config.Silent, "silent", false, "Do not show informational messages")
	f.StringVarP(&config.Section, "section", "s", "default", "The section in config file to use")
	f.StringVarP(&config.Profile, "profile", "p", "default", "Use a specific profile from your credential file")
	f.StringVarP(&config.Region, "region", "r", "us-east-1", "The AWS region to use")
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "ERROR:", err)
	os.Exit(1)
}

func printConfigFileWarning(err error) {
	cause := errors.Cause(err)
	switch e := cause.(type) {
	case *os.PathError:
		log.Println("No config file found:", e)
		fmt.Fprintf(os.Stderr, "No config file found on %s. Using only command line flags\n", e.Path)
	case *athenai.SectionError:
		log.Println("Error:", e)
		fmt.Fprintf(os.Stderr, "Section '%s' not found in %s. Please check if the '%s' section exists "+
			"in your config file and add it if it does not exist. Using only command line flags this time\n",
			e.Section, e.Path, e.Section)
	default:
		log.Println("Error loading config file:", e)
		fmt.Fprintln(os.Stderr, "Error loading config file. Use --debug flag for more details. Using only command line flags this time")
	}
}

// initConfig loads configurations from the config file and then override them by parsing flags.
// rawArgs should be os.Args.
func initConfig(cfg *athenai.Config, cmd *cobra.Command, rawArgs []string) error {
	if err := athenai.LoadConfigFile(cfg, cfgFile); err != nil && !cfg.Silent {
		// Config file is optional so just print the error and not return it.
		printConfigFileWarning(err)
	}
	// Parse flags again to override configs in config file.
	return cmd.ParseFlags(rawArgs)
}

// newClient creates a new Athena client.
func newClient(cfg *athenai.Config) *athena.Athena {
	c := aws.NewConfig().WithRegion(cfg.Region)
	if cfg.Debug {
		c = c.WithLogLevel(aws.LogDebugWithHTTPBody | aws.LogDebugWithRequestErrors)
	}
	return athena.New(session.Must(session.NewSessionWithOptions(session.Options{
		Config:  *c,
		Profile: cfg.Profile,
	})))
}
