// Package config implements the template's stdlib-first configuration.
//
// Precedence (highest wins):
//
//  1. command-line flags      (parsed by the CLI layer)
//  2. environment variables   (this package)
//  3. built-in defaults       (this package)
//
// File support can be layered later without touching call sites: extend Load
// to overlay file contents between defaults and environment. Until a real
// need appears, that complexity stays out (YAGNI).
package config

import (
	"strings"

	"github.com/guilhermelinosp/hellnet-cli-template/internal/env"
)

// Config carries the operational settings the base template cares about.
type Config struct {
	// LogLevel is one of debug, info, warn, error (default: info).
	LogLevel string `json:"log_level"`
	// LogFormat is one of text or json (default: text).
	LogFormat string `json:"log_format"`
}

// Defaults returns the baseline configuration.
func Defaults() Config {
	return Config{LogLevel: "info", LogFormat: "text"}
}

// Load resolves configuration from the process environment. prefix selects
// the variable namespace, e.g. prefix "MYAPP" reads MYAPP_LOG_LEVEL and
// MYAPP_LOG_FORMAT. Invalid values fall back to defaults silently-by-design;
// prefer flag validation for user-facing feedback.
func Load(prefix string) Config {
	cfg := Defaults()
	if v := env.String(envKey(prefix, "LOG_LEVEL"), ""); v != "" {
		cfg.LogLevel = strings.ToLower(v)
	}
	if v := env.String(envKey(prefix, "LOG_FORMAT"), ""); v != "" {
		cfg.LogFormat = strings.ToLower(v)
	}
	return cfg
}

// envKey builds "<PREFIX>_<FIELD>" sanitized for environment naming rules.
func envKey(prefix, field string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.Trim(prefix, "_"), "-", "_")) + "_" + field
}
