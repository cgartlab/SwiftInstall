package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	Mirror          string `json:"mirror,omitempty"`
	Proxy           string `json:"proxy,omitempty"`
	ProxyAutoDetect *bool  `json:"proxy_auto_detect,omitempty"`
	LogLevel        string `json:"log_level,omitempty"`
	LogFile         string `json:"log_file,omitempty"`
	DefaultManifest string `json:"default_manifest,omitempty"`
	SkipExisting    *bool  `json:"skip_existing,omitempty"`
	SkipChecks      *bool  `json:"skip_checks,omitempty"`
	Color           string `json:"color,omitempty"`
	RetryCount      *int   `json:"retry_count,omitempty"`
	RetryDelaySec   *int   `json:"retry_delay_sec,omitempty"`
}

func defaults() *Config {
	rc := 2
	rd := 3
	return &Config{
		Mirror:          "",
		Proxy:           "",
		LogLevel:        "info",
		LogFile:         "",
		DefaultManifest: "",
		Color:           "auto",
		RetryCount:      &rc,
		RetryDelaySec:   &rd,
	}
}

func globalPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".sis", "config.json"), nil
}

func localPath() string {
	return ".sis.json"
}

func Load() (*Config, error) {
	cfg := defaults()

	gp, err := globalPath()
	if err != nil {
		return cfg, fmt.Errorf("home dir: %w", err)
	}

	if data, err := os.ReadFile(gp); err == nil {
		var gc Config
		if err := json.Unmarshal(data, &gc); err == nil {
			merge(cfg, &gc)
		}
	} else if !os.IsNotExist(err) {
		return cfg, fmt.Errorf("read global config: %w", err)
	}

	if data, err := os.ReadFile(localPath()); err == nil {
		var lc Config
		if err := json.Unmarshal(data, &lc); err == nil {
			merge(cfg, &lc)
		}
	} else if !os.IsNotExist(err) {
		return cfg, fmt.Errorf("read local config: %w", err)
	}

	sanitize(cfg)
	return cfg, nil
}

func merge(dst, src *Config) {
	if src.Mirror != "" {
		dst.Mirror = src.Mirror
	}
	if src.Proxy != "" {
		dst.Proxy = src.Proxy
	}
	if src.ProxyAutoDetect != nil {
		dst.ProxyAutoDetect = src.ProxyAutoDetect
	}
	if src.LogLevel != "" {
		dst.LogLevel = src.LogLevel
	}
	if src.LogFile != "" {
		dst.LogFile = src.LogFile
	}
	if src.DefaultManifest != "" {
		dst.DefaultManifest = src.DefaultManifest
	}
	if src.SkipExisting != nil {
		dst.SkipExisting = src.SkipExisting
	}
	if src.SkipChecks != nil {
		dst.SkipChecks = src.SkipChecks
	}
	if src.Color != "" {
		dst.Color = src.Color
	}
	if src.RetryCount != nil {
		dst.RetryCount = src.RetryCount
	}
	if src.RetryDelaySec != nil {
		dst.RetryDelaySec = src.RetryDelaySec
	}
}

func (c *Config) Validate() error {
	if c.LogLevel != "" {
		valid := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
		if !valid[strings.ToLower(c.LogLevel)] {
			return fmt.Errorf("log_level must be one of: debug, info, warn, error (got %q)", c.LogLevel)
		}
	}
	if c.Color != "" {
		valid := map[string]bool{"auto": true, "always": true, "never": true}
		if !valid[strings.ToLower(c.Color)] {
			return fmt.Errorf("color must be auto, always, or never (got %q)", c.Color)
		}
	}
	if c.RetryCount != nil && *c.RetryCount < 0 {
		return fmt.Errorf("retry_count must be non-negative (got %d)", *c.RetryCount)
	}
	if c.RetryDelaySec != nil && *c.RetryDelaySec < 1 {
		return fmt.Errorf("retry_delay_sec must be at least 1 (got %d)", *c.RetryDelaySec)
	}
	if c.Mirror != "" && c.Mirror != "ustc" && c.Mirror != "official" {
		return fmt.Errorf("mirror must be ustc or official (got %q)", c.Mirror)
	}
	if c.Proxy != "" {
		u, err := url.Parse(c.Proxy)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return fmt.Errorf("proxy must be a valid http(s) URL (got %q)", c.Proxy)
		}
	}
	return nil
}

func sanitize(cfg *Config) {
	if cfg.LogLevel != "" {
		valid := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
		if !valid[strings.ToLower(cfg.LogLevel)] {
			cfg.LogLevel = "info"
		} else {
			cfg.LogLevel = strings.ToLower(cfg.LogLevel)
		}
	}
	if cfg.Color != "" {
		valid := map[string]bool{"auto": true, "always": true, "never": true}
		if !valid[strings.ToLower(cfg.Color)] {
			cfg.Color = "auto"
		} else {
			cfg.Color = strings.ToLower(cfg.Color)
		}
	}
	if cfg.RetryCount != nil && *cfg.RetryCount < 0 {
		cfg.RetryCount = nil
	}
	if cfg.RetryDelaySec != nil && *cfg.RetryDelaySec < 1 {
		cfg.RetryDelaySec = nil
	}
	if cfg.Mirror != "" && cfg.Mirror != "ustc" && cfg.Mirror != "official" {
		cfg.Mirror = ""
	}
	if cfg.Proxy != "" {
		u, err := url.Parse(cfg.Proxy)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			cfg.Proxy = ""
		}
	}
}

type SaveOpts struct {
	Local bool
}

func Save(cfg *Config, opts SaveOpts) error {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	var path string
	if opts.Local {
		path = localPath()
	} else {
		var err error
		path, err = globalPath()
		if err != nil {
			return err
		}
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir %s: %w", dir, err)
	}

	if _, err := os.Stat(dir); err != nil {
		return fmt.Errorf("config dir not accessible %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("write temp config %s: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename config %s -> %s: %w", tmpPath, path, err)
	}
	return nil
}

var validKeys = map[string]bool{
	"mirror":           true,
	"proxy":            true,
	"proxy_auto_detect": true,
	"log_level":        true,
	"log_file":         true,
	"default_manifest": true,
	"skip_existing":    true,
	"skip_checks":      true,
	"color":            true,
	"retry_count":      true,
	"retry_delay_sec":  true,
}

func Get(cfg *Config, key string) (any, error) {
	switch key {
	case "mirror":
		return cfg.Mirror, nil
	case "proxy":
		return cfg.Proxy, nil
	case "proxy_auto_detect":
		return boolVal(cfg.ProxyAutoDetect), nil
	case "log_level":
		return cfg.LogLevel, nil
	case "log_file":
		return cfg.LogFile, nil
	case "default_manifest":
		return cfg.DefaultManifest, nil
	case "skip_existing":
		return boolVal(cfg.SkipExisting), nil
	case "skip_checks":
		return boolVal(cfg.SkipChecks), nil
	case "color":
		return cfg.Color, nil
	case "retry_count":
		if cfg.RetryCount == nil {
			return 0, nil
		}
		return *cfg.RetryCount, nil
	case "retry_delay_sec":
		if cfg.RetryDelaySec == nil {
			return 0, nil
		}
		return *cfg.RetryDelaySec, nil
	default:
		return nil, fmt.Errorf("unknown config key: %s", key)
	}
}

func Set(cfg *Config, key, value string) error {
	switch key {
	case "mirror":
		cfg.Mirror = value
	case "proxy":
		cfg.Proxy = value
	case "proxy_auto_detect":
		b, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("proxy_auto_detect must be true or false")
		}
		cfg.ProxyAutoDetect = &b
	case "log_level":
		valid := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
		if !valid[strings.ToLower(value)] {
			return fmt.Errorf("log_level must be one of: debug, info, warn, error")
		}
		cfg.LogLevel = strings.ToLower(value)
	case "log_file":
		cfg.LogFile = value
	case "default_manifest":
		cfg.DefaultManifest = value
	case "skip_existing":
		b, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("skip_existing must be true or false")
		}
		cfg.SkipExisting = &b
	case "skip_checks":
		b, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("skip_checks must be true or false")
		}
		cfg.SkipChecks = &b
	case "color":
		valid := map[string]bool{"auto": true, "always": true, "never": true}
		if !valid[strings.ToLower(value)] {
			return fmt.Errorf("color must be auto, always, or never")
		}
		cfg.Color = strings.ToLower(value)
	case "retry_count":
		n, err := strconv.Atoi(value)
		if err != nil || n < 0 {
			return fmt.Errorf("retry_count must be a non-negative integer")
		}
		cfg.RetryCount = &n
	case "retry_delay_sec":
		n, err := strconv.Atoi(value)
		if err != nil || n < 1 {
			return fmt.Errorf("retry_delay_sec must be a positive integer")
		}
		cfg.RetryDelaySec = &n
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}
	return nil
}

func boolVal(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

func BoolVal(b *bool) bool {
	return boolVal(b)
}

func IntVal(v *int, defaultVal int) int {
	if v == nil {
		return defaultVal
	}
	return *v
}

func IsValidKey(key string) bool {
	return validKeys[key]
}
