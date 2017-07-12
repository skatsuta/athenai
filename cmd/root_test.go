package cmd

import (
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/skatsuta/athenai/athenai"
	"github.com/skatsuta/athenai/internal/bytes"
	"github.com/skatsuta/athenai/internal/testhelper"
	"github.com/stretchr/testify/assert"
)

// Run TestMain(m) to run init()
func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestPersistentPreRun(t *testing.T) {
	oldArgs := os.Args
	oldStdout := stdout
	defer func() {
		os.Args = oldArgs
		stdout = oldStdout
	}()

	os.Args = []string{
		"athenai", "run",
		"--profile", "TestPersistentPreRunEProfile",
		"--region", "us-west-2",
		"--location", "s3://TestPersistentPreRunEBucket/",
	}
	args := []string{}
	err := runCmd.Parent().PersistentPreRunE(runCmd, args)

	assert.NoError(t, err)
	assert.Equal(t, "TestPersistentPreRunEProfile", config.Profile)
	assert.Equal(t, "us-west-2", config.Region)
	assert.Equal(t, "s3://TestPersistentPreRunEBucket/", config.Location)
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
			},
			want: &athenai.Config{
				Section:  section,
				Profile:  "testprofile",
				Region:   "us-east-2",
				Location: "s3://samplebucket/",
			},
		},
	}

	for _, tt := range tests {
		config.Section = section
		initConfig(config, tt.cfgFile, runCmd, tt.rawArgs)

		assert.Equal(t, tt.want.Profile, config.Profile, "cfgFile: %#v, rawArgs: %#v", tt.cfgFile, tt.rawArgs)
		assert.Equal(t, tt.want.Region, config.Region, "cfgFile: %#v, rawArgs: %#v", tt.cfgFile, tt.rawArgs)
		assert.Equal(t, tt.want.Location, config.Location, "cfgFile: %#v, rawArgs: %#v", tt.cfgFile, tt.rawArgs)
	}
}

func TestInitConfigNoSection(t *testing.T) {
	cfg := &athenai.Config{
		Section:  "default",
		Profile:  "TestInitConfigNoSectionProfile",
		Location: "s3://samplebucket-2/",
	}

	_, file, cleanup, err := testhelper.CreateConfigFile("TestInitConfigNoSection", cfg)
	defer cleanup()
	assert.NoError(t, err)

	section := "no_section"
	rawArgs := []string{
		"run",
		"--section", section,
		"--profile", "TestInitConfigNoSectionProfile",
		"--location", "s3://samplebucket-2/",
	}
	config.Section = section
	initConfig(config, file.Name(), runCmd, rawArgs)

	assert.Equal(t, cfg.Profile, config.Profile)
	assert.Equal(t, cfg.Location, config.Location)
}

func TestInitConfigConfigFileNoArgs(t *testing.T) {
	cfg := &athenai.Config{
		Section:  "default",
		Profile:  "TestInitConfigConfigFileNoArgsProfile",
		Region:   "eu-central-1",
		Location: "s3://TestInitConfigConfigFileNoArgsBucket/",
	}

	_, file, cleanup, err := testhelper.CreateConfigFile("TestInitConfigConfigFileNoArgs", cfg)
	defer cleanup()
	assert.NoError(t, err)

	rawArgs := []string{"run"}
	config.Section = "default"
	initConfig(config, file.Name(), runCmd, rawArgs)

	assert.Equal(t, cfg.Profile, config.Profile)
	assert.Equal(t, cfg.Region, config.Region)
	assert.Equal(t, cfg.Location, config.Location)
}

func TestInitConfigConfigFileAndArgs(t *testing.T) {
	cfg := &athenai.Config{
		Section:  "test",
		Profile:  "TestInitConfigConfigFileAndArgs",
		Region:   "eu-west-1",
		Location: "s3://TestInitConfigConfigFileAndArgsBucket/folder/",
	}

	_, file, cleanup, err := testhelper.CreateConfigFile("TestInitConfigConfigFileAndArgs", cfg)
	defer cleanup()
	assert.NoError(t, err)

	rawArgs := []string{
		"run",
		"--section", "test",
		"--profile", "TestInitConfigConfigFileNoArgsProfile2",
		"--location", "TestInitConfigConfigFileAndArgsBucket2",
	}
	config.Section = "test"
	initConfig(config, file.Name(), runCmd, rawArgs)

	assert.Equal(t, "TestInitConfigConfigFileNoArgsProfile2", config.Profile)
	assert.Equal(t, cfg.Region, config.Region)
	assert.Equal(t, "TestInitConfigConfigFileAndArgsBucket2", config.Location)
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

func TestRunShowVersion(t *testing.T) {
	showVersion = true
	oldStdout := stdout
	defer func() {
		showVersion = false
		stdout = oldStdout
	}()

	var out bytes.Buffer
	stdout = &out
	RootCmd.Run(RootCmd, []string{})
	got := out.String()

	assert.Equal(t, commandVersion+"\n", got)
}
