package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"text/template"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

const configFileTmpl = `
[{{.Section}}]
debug = {{.Debug}}
silent = {{.Silent}}
profile = {{.Profile}}
region = {{.Region}}
database = {{.Database}}
location = {{.Location}}
`

func createConfigFile(dir, name string, cfg *Config) (file *os.File, cleanup func(), err error) {
	file, err = ioutil.TempFile(dir, name)
	if err != nil {
		return nil, nil, err
	}

	err = template.Must(template.New(name).Parse(configFileTmpl)).Execute(file, cfg)
	cleanup = func() {
		file.Close()
		os.Remove(file.Name())
	}
	return file, cleanup, err
}

func TestLoadFile(t *testing.T) {
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

	file, cleanup, err := createConfigFile("", "TestLoadFile", want)
	defer cleanup()
	assert.NoError(t, err)

	got := &Config{Section: section}
	err = LoadFile(got, file.Name())
	got.iniCfg = nil // ignore iniCfg field

	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestLoadFileFromHomeDir(t *testing.T) {
	// Set temporary home directory to $HOME
	tmpHomedir := os.TempDir()
	defaultHomeDir := os.Getenv("HOME")
	err := os.Setenv("HOME", tmpHomedir)
	assert.NoError(t, err)
	dir := filepath.Join(tmpHomedir, defaultConfigDir)
	err = os.MkdirAll(dir, 0755)
	assert.NoError(t, err)
	fileName := filepath.Join(dir, defaultConfigFile)
	file, err := os.Create(fileName)
	assert.NoError(t, err)

	section := "default"
	want := &Config{
		Section:  section,
		Profile:  "default",
		Region:   "us-west-1",
		Database: "sampledb",
		Location: "s3://testloadfilebucket/prefix",
	}

	err = template.Must(template.New("TestLoadFileFromHomeDir").Parse(configFileTmpl)).Execute(file, want)
	defer func() {
		file.Close()
		os.Remove(file.Name())
	}()
	assert.NoError(t, err)

	got := &Config{Section: section}
	err = LoadFile(got, "") // if empty path is given we read config file under home dir
	got.iniCfg = nil        // ignore iniCfg field

	assert.NoError(t, err)
	assert.Equal(t, want, got)

	// Restore $HOME
	err = os.Setenv("HOME", defaultHomeDir)
	assert.NoError(t, err)
}

func TestLoadFileError(t *testing.T) {
	err := LoadFile(nil, "")
	assert.Error(t, err, "config is not nil")

	err = LoadFile(&Config{}, "")
	assert.Error(t, err, "section name is not empty")

	path := "/no_existent_file"
	err = LoadFile(&Config{Section: "default"}, path)
	assert.Error(t, err, "config file '"+path+"' exists unexpectedly")
	assert.IsType(t, &os.PathError{}, errors.Cause(err))

	section := "no_section"
	file, cleanup, err := createConfigFile("", "TestLoadFileError", &Config{Section: "default"})
	err = LoadFile(&Config{Section: section}, file.Name())
	defer cleanup()
	assert.Contains(t, err.Error(), "failed to get section '"+section+"' in config file")
}
