package config

import (
	"io/ioutil"
	"os"
	"testing"
	"text/template"

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

func createConfigFile(name string, cfg *Config) (file *os.File, cleanup func(), err error) {
	file, err = ioutil.TempFile("", name)
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

	file, cleanup, err := createConfigFile("TestLoadFile", want)
	defer cleanup()
	assert.NoError(t, err)

	got := &Config{Section: section}
	err = LoadFile(got, file.Name())
	got.iniCfg = nil // ignore iniCfg field

	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestLoadFileError(t *testing.T) {
	err := LoadFile(nil, "")
	assert.Error(t, err, "config is not nil")

	err = LoadFile(&Config{}, "")
	assert.Error(t, err, "section name is not empty")

	path := "/no_existent_file"
	err = LoadFile(&Config{Section: "default"}, path)
	assert.Error(t, err, "config file '"+path+"' exists unexpectedly")

	section := "no_section"
	file, cleanup, err := createConfigFile("TestLoadFileError", &Config{})
	err = LoadFile(&Config{Section: section}, file.Name())
	defer cleanup()
	assert.Error(t, err, "section '"+section+"' exists unexpectedly")
}
