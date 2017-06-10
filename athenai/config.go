package athenai

import (
	"fmt"
	"path/filepath"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/skatsuta/athenai/exec"
	"gopkg.in/ini.v1"
)

const (
	defaultConfigDir  = ".athenai"
	defaultConfigFile = "config"
)

// Config is a configuration information.
type Config struct {
	Debug    bool   `ini:"debug"`
	Silent   bool   `ini:"silent"`
	Section  string `ini:"-"`
	Profile  string `ini:"profile"`
	Region   string `ini:"region"`
	Database string `ini:"database"`
	Location string `ini:"location"`

	iniCfg *ini.File `ini:"-"`
}

// QueryConfig creates an exec.QueryConfig struct based on c.
func (c *Config) QueryConfig() *exec.QueryConfig {
	return &exec.QueryConfig{
		Database: c.Database,
		Location: c.Location,
	}
}

// SectionError represents an error about section in config file.
type SectionError struct {
	Path    string
	Section string
	Err     error
}

func (se *SectionError) Error() string {
	return fmt.Sprintf("failed to get section '%s' in %s: %s", se.Section, se.Path, se.Err)
}

// Cause returns the underlying cause of the error.
func (se *SectionError) Cause() error {
	return se.Err
}

// LoadConfigFile loads configurations at `cfg.Section` section into `cfg` from `path`.
// If `path` is empty, `$HOME/.athenai/config` is used.
func LoadConfigFile(cfg *Config, path string) error {
	if cfg == nil {
		return errors.New("cfg is nil")
	}
	if cfg.Section == "" {
		return errors.New("section name is empty")
	}

	filePath, err := normalizeConfigPath(path)
	if err != nil {
		return errors.Wrap(err, "failed to find config file path")
	}

	iniCfg, err := ini.Load(filePath)
	if err != nil {
		return errors.Wrap(err, "failed to load config file")
	}
	cfg.iniCfg = iniCfg

	sec, err := iniCfg.GetSection(cfg.Section)
	if err != nil {
		return &SectionError{
			Path:    filePath,
			Section: cfg.Section,
			Err:     err,
		}
	}

	return sec.MapTo(cfg)
}

func normalizeConfigPath(path string) (string, error) {
	if path != "" {
		return homedir.Expand(path)
	}

	home, err := homedir.Dir()
	if err != nil {
		return "", errors.Wrap(err, "failed to find your home directory")
	}

	path = filepath.Join(home, defaultConfigDir, defaultConfigFile)
	return path, nil
}