package cmd

import (
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/skatsuta/athenai/athenai"
	"github.com/skatsuta/athenai/internal/testhelper"
	"github.com/stretchr/testify/assert"
)

// Run TestMain(m) to run init()
func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestInitConfigNoConfigFile(t *testing.T) {
	section := "default"

	tests := []struct {
		cfgFile string
		rawArgs []string
		want    *athenai.Config
	}{
		{
			cfgFile: "/no_existent_config",
			rawArgs: []string{
				"run",
				"--profile", "testprofile",
				"--region", "us-east-2",
				"--location", "s3://samplebucket/",
				"--database", "testdb",
			},
			want: &athenai.Config{
				Section:  section,
				Profile:  "testprofile",
				Region:   "us-east-2",
				Location: "s3://samplebucket/",
				Database: "testdb",
			},
		},
	}

	for _, tt := range tests {
		config.Section = section
		err := initConfig(config, tt.cfgFile, runCmd, tt.rawArgs)

		assert.NoError(t, err, "cfgFile: %#v, rawArgs: %#v", tt.cfgFile, tt.rawArgs)
		assert.Equal(t, tt.want.Profile, config.Profile, "cfgFile: %#v, rawArgs: %#v", tt.cfgFile, tt.rawArgs)
		assert.Equal(t, tt.want.Region, config.Region, "cfgFile: %#v, rawArgs: %#v", tt.cfgFile, tt.rawArgs)
		assert.Equal(t, tt.want.Location, config.Location, "cfgFile: %#v, rawArgs: %#v", tt.cfgFile, tt.rawArgs)
	}
}

func TestInitConfigNoSection(t *testing.T) {
	cfg := &athenai.Config{
		Section:  "default",
		Profile:  "TestInitConfigNoSectionProfile",
		Region:   "us-west-1",
		Location: "s3://samplebucket-2/",
		Database: "testdb2",
	}

	section := "no_section"
	rawArgs := []string{
		"run",
		"--section", section,
		"--profile", "TestInitConfigNoSectionProfile",
		"--region", "us-west-1",
		"--location", "s3://samplebucket-2/",
		"--database", "testdb2",
	}

	_, file, cleanup, err := testhelper.CreateConfigFile("TestInitConfigNoSection", cfg)
	defer cleanup()
	assert.NoError(t, err)

	config.Section = section
	err = initConfig(config, file.Name(), runCmd, rawArgs)

	assert.NoError(t, err)
	assert.Equal(t, cfg.Profile, config.Profile)
	assert.Equal(t, cfg.Location, config.Location)
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		cfg      *athenai.Config
		logLevel aws.LogLevelType
	}{
		{
			cfg: &athenai.Config{
				Debug:   true,
				Profile: "TestNewClientProfile",
				Region:  "us-east-1",
			},
			logLevel: aws.LogDebugWithHTTPBody | aws.LogDebugWithRequestErrors,
		},
	}

	for _, tt := range tests {
		client := newClient(tt.cfg)

		assert.Equal(t, tt.cfg.Region, *client.Client.Config.Region)
		assert.Equal(t, tt.logLevel, *client.Client.Config.LogLevel)
	}
}
