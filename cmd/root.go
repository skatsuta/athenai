package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/athena"
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
	f.StringVar(&config.Section, "section", "s", "The section in config file to use")
	f.StringVarP(&config.Profile, "profile", "p", "default", "Use a specific profile from your credential file")
	f.StringVarP(&config.Region, "region", "r", "", "The AWS region to use")
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "ERROR:", err)
	os.Exit(1)
}

func printConfigFileWarning(err error) {
	switch e := err.(type) {
	case *os.PathError:
		log.Println("No config file found:", e)
		fmt.Fprintf(os.Stderr, "No config file found on %s. Using only command line flags\n", e.Path)
	case *athenai.SectionError:
		log.Println("Error:", e)
		fmt.Fprintf(os.Stderr, "Section '%s' not found in %s. Please check if the '%s' section exists in the config file and add it if it does not. Using only command line flags now\n",
			e.Section, e.Section, e.Path)
	default:
		log.Println("Error loading config file:", e)
		fmt.Fprintln(os.Stderr, "Error loading config file. Use --debug flag for more details. Using only command line flags now")
	}
}

// initConfig loads configurations from the config file and then override them by parsing flags.
// rawArgs should be os.Args.
func initConfig(cfg *athenai.Config, cmd *cobra.Command, rawArgs []string) error {
	if err := athenai.LoadConfigFile(cfg, cfgFile); err != nil && !cfg.Silent {
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
