package config

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/thesatellite-ai/fetchr/pkg/curl"
)

//go:embed default_config.jsonc
var defaultConfigContent []byte

// Config is the top-level configuration.
type Config struct {
	Defaults curl.RequestOptions        `json:"defaults"`
	Logging  LoggingConfig              `json:"logging"`
	Profiles map[string]curl.RequestOptions `json:"profiles"`
}

// LoggingConfig configures request logging.
type LoggingConfig struct {
	Enabled bool        `json:"enabled"`
	File    string      `json:"file"`
	Webhook string      `json:"webhook"`
	OTEL    *OTELConfig `json:"otel,omitempty"`
}

// OTELConfig configures OpenTelemetry export.
type OTELConfig struct {
	Endpoint string `json:"endpoint"`
	Service  string `json:"service"`
}

// DefaultConfigContent returns the embedded default config.
func DefaultConfigContent() []byte {
	return defaultConfigContent
}

// LoadConfig loads and parses a config file at the given path.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	return parseConfig(data)
}

// LoadConfigAuto finds, loads, and returns config. Creates default if none found.
func LoadConfigAuto() (*Config, error) {
	path, err := FindConfigFile()
	if err == nil {
		return LoadConfig(path)
	}

	// No config found — create default
	path, err = EnsureDefaultConfig()
	if err != nil {
		// If we can't create a config, return empty defaults
		return &Config{}, nil
	}
	return LoadConfig(path)
}

// EnsureDefaultConfig creates the default config file if it doesn't exist.
// Returns the path to the config file.
func EnsureDefaultConfig() (string, error) {
	path := DefaultConfigPath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create config dir: %w", err)
	}
	if err := os.WriteFile(path, defaultConfigContent, 0644); err != nil {
		return "", fmt.Errorf("write default config: %w", err)
	}
	return path, nil
}

// ResolveProfile returns the merged options for a named profile.
// Profile options are merged on top of defaults.
func (c *Config) ResolveProfile(name string) (curl.RequestOptions, error) {
	profile, ok := c.Profiles[name]
	if !ok {
		return curl.RequestOptions{}, fmt.Errorf("profile %q not found", name)
	}
	return curl.Merge(c.Defaults, profile), nil
}

// BuildLogger creates a Logger from the logging config.
func (c *Config) BuildLogger() curl.Logger {
	if !c.Logging.Enabled {
		return nil
	}

	var loggers []curl.Logger

	if c.Logging.File != "" {
		path := expandHome(c.Logging.File)
		loggers = append(loggers, curl.NewFileLogger(path))
	}

	if c.Logging.Webhook != "" {
		loggers = append(loggers, curl.NewWebhookLogger(c.Logging.Webhook))
	}

	if len(loggers) == 0 {
		return nil
	}
	if len(loggers) == 1 {
		return loggers[0]
	}
	return curl.NewMultiLogger(loggers...)
}

func parseConfig(data []byte) (*Config, error) {
	cleaned := stripJSONC(string(data))
	var cfg Config
	if err := json.Unmarshal([]byte(cleaned), &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}

// stripJSONC removes // line comments and /* */ block comments from JSONC.
func stripJSONC(s string) string {
	var result strings.Builder
	i := 0
	inString := false

	for i < len(s) {
		// Handle string literals (don't strip inside strings)
		if s[i] == '"' && (i == 0 || s[i-1] != '\\') {
			inString = !inString
			result.WriteByte(s[i])
			i++
			continue
		}

		if inString {
			result.WriteByte(s[i])
			i++
			continue
		}

		// Line comment
		if i+1 < len(s) && s[i] == '/' && s[i+1] == '/' {
			for i < len(s) && s[i] != '\n' {
				i++
			}
			continue
		}

		// Block comment
		if i+1 < len(s) && s[i] == '/' && s[i+1] == '*' {
			i += 2
			for i+1 < len(s) && !(s[i] == '*' && s[i+1] == '/') {
				i++
			}
			if i+1 < len(s) {
				i += 2
			}
			continue
		}

		result.WriteByte(s[i])
		i++
	}

	return result.String()
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}
