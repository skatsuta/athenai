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
	"github.com/skatsuta/athenai/core"
	"github.com/spf13/cobra"
)

var (
	showVersion bool
	cfgFile     string

	stdout = os.Stdout

	config = &core.Config{}
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "athenai",
	Short: "Athenai is a simple and easy-to-use command line tool that runs SQL statements on Amazon Athena.",
	// TODO
	Long: `Athenai is a simple and easy-to-use command line tool that runs SQL statements on Amazon Athena.
With Athenai you can easily run multiple queries at a time on Amazon Athena and see the results
in table or CSV format once the executions are complete.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)
		if !config.Debug {
			// Disable debug log messages
			log.SetOutput(ioutil.Discard)
		}
		initConfig(config, cfgFile, cmd, os.Args[1:])
		log.Printf("Initialized Config: %#v\n", config)

		if config.Output != "" {
			file, err := os.Create(config.Output)
			if err != nil {
				return errors.Wrap(err, "failed to open file to write")
			}
			log.Printf("Setting output to %s\n", file.Name())
			stdout = file
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if showVersion {
			fmt.Fprintln(stdout, commandVersion)
		} else {
			cmd.Help()
		}
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		return stdout.Close()
	},
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		os.Exit(1)
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
	f.StringVarP(&config.Output, "output", "o", "", "Output query results to a given file path instead of stdout")

	// Define local flags
	RootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "Show version")
}

func printConfigFileWarning(err error) {
	cause := errors.Cause(err)
	switch e := cause.(type) {
	case *os.PathError:
		log.Println("No config file found:", e)
		fmt.Fprintf(os.Stderr, "No config file found on %s. Using only command line flags\n", e.Path)
	case *core.SectionError:
		log.Println("Error on section:", e)
		fmt.Fprintf(os.Stderr, "Section '%s' not found in %s. Please check if the '%s' section exists "+
			"in your config file and add it if it does not exist. Using only command line flags this time\n",
			e.Section, e.Path, e.Section)
	default:
		log.Println("Error loading config file:", e)
		fmt.Fprintln(os.Stderr, "Error loading config file. Use --debug flag for more details. Using only command line flags this time")
	}
}

// initConfig loads configurations from the config file and then override them by parsing flags.
// rawArgs should be os.Args[1:].
func initConfig(cfg *core.Config, cfgFile string, cmd *cobra.Command, rawArgs []string) {
	log.Printf("Primitive config: %#v\n", cfg)
	if err := core.LoadConfigFile(cfg, cfgFile); err != nil && !cfg.Silent {
		// Config file is optional so just print the error and not return it.
		printConfigFileWarning(err)
	}
	// Parse flags again to override configs in config file.
	log.Printf("Raw args: %#v\n", rawArgs)
	cmd.ParseFlags(rawArgs)
}

// newClient creates a new Athena client.
func newClient(cfg *core.Config) *athena.Athena {
	log.Printf("Creating Athena client: region = %s, profile = %s\n", cfg.Region, cfg.Profile)
	c := aws.NewConfig().WithRegion(cfg.Region)
	if cfg.Debug {
		log.Println("Debug mode is enabled. Setting log level for AWS SDK to debug")
		c = c.WithLogLevel(aws.LogDebugWithHTTPBody | aws.LogDebugWithRequestErrors)
	}
	return athena.New(session.Must(session.NewSessionWithOptions(session.Options{
		Config:  *c,
		Profile: cfg.Profile,
	})))
}
