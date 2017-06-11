package athenai

import (
	"os"
	"testing"

	"github.com/pkg/errors"
	"github.com/skatsuta/athenai/internal/testhelper"
	"github.com/stretchr/testify/assert"
)

func TestLoadConfigFile(t *testing.T) {
	section := "test"
	want := &Config{
		Debug:    true,
		Silent:   true,
		Section:  section,
		Profile:  "test",
		Region:   "us-west-1",
		Database: "testdb",
		Location: "s3://testloadfilebucket/prefix",
	}

	_, file, cleanup, err := testhelper.CreateConfigFile("TestLoadConfigFile", want)
	defer cleanup()
	assert.NoError(t, err)

	got := &Config{Section: section}
	err = LoadConfigFile(got, file.Name())
	got.iniCfg = nil // ignore iniCfg field

	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestLoadConfigFileFromHomeDir(t *testing.T) {
	section := "default"
	want := &Config{
		Section:  section,
		Profile:  "default",
		Region:   "us-west-1",
		Database: "sampledb",
		Location: "s3://testloadfilebucket/prefix",
	}

	homeDir, _, cleanup, err := testhelper.CreateConfigFile("TestLoadConfigFileFromHomeDir", want)
	defer cleanup()
	assert.NoError(t, err)

	// Set temporary home directory to $HOME
	defaultHomeDir := os.Getenv("HOME")
	err = os.Setenv("HOME", homeDir)
	assert.NoError(t, err)

	got := &Config{Section: section}
	err = LoadConfigFile(got, "") // if empty path is given we read config file under home dir
	got.iniCfg = nil              // ignore iniCfg field

	assert.NoError(t, err)
	assert.Equal(t, want, got)

	// Restore $HOME
	err = os.Setenv("HOME", defaultHomeDir)
	assert.NoError(t, err)
}

func TestLoadConfigFileError(t *testing.T) {
	err := LoadConfigFile(nil, "")
	assert.Error(t, err, "config is not nil")

	err = LoadConfigFile(&Config{}, "")
	assert.Error(t, err, "section name is not empty")

	path := "/no_existent_file"
	err = LoadConfigFile(&Config{Section: "default"}, path)
	assert.Error(t, err, "config file '"+path+"' exists unexpectedly")
	assert.IsType(t, &os.PathError{}, errors.Cause(err))

	section := "no_section"
	_, file, cleanup, err := testhelper.CreateConfigFile("TestLoadConfigFileError", &Config{Section: "default"})
	err = LoadConfigFile(&Config{Section: section}, file.Name())
	defer cleanup()
	e, ok := err.(*SectionError)
	assert.True(t, ok)
	assert.Equal(t, file.Name(), e.Path)
	assert.Equal(t, section, e.Section)
	assert.Contains(t, e.Cause.Error(), "does not exist")
}
