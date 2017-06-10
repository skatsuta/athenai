package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/athena"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/skatsuta/athenai/athenai"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
)

var config = &athenai.Config{}

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
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	f := RootCmd.PersistentFlags()
	f.StringVar(&cfgFile, "config", "", "Config file (default is $HOME/.athenai/config.yml)")
	f.BoolVar(&config.Debug, "debug", false, "Turn on debug logging")
	f.StringVar(&config.Section, "section", "s", "The section in config file to use")
	f.StringVarP(&config.Profile, "profile", "p", "default", "Use a specific profile from your credential file")
	f.StringVarP(&config.Region, "region", "r", "", "The AWS region to use")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fatal(err)
		}

		// Search config in home directory with name ".athenai" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".athenai")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "ERROR:", err)
	os.Exit(1)
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
