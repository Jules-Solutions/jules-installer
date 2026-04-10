// Package config manages the jules-installer configuration file at ~/.config/jules/config.toml.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/BurntSushi/toml"
)

// Config is the top-level configuration structure persisted to config.toml.
type Config struct {
	Auth  AuthConfig  `toml:"auth"`
	Local LocalConfig `toml:"local"`
}

// AuthConfig holds authentication credentials and service endpoints.
type AuthConfig struct {
	APIKey  string `toml:"api_key"`
	APIURL  string `toml:"api_url"`
	AuthURL string `toml:"auth_url"`
}

// LocalConfig holds paths and local machine settings.
type LocalConfig struct {
	VaultPath  string `toml:"vault_path"`
	Shell      string `toml:"shell"`
	InstallDir string `toml:"install_dir"`
}

// ConfigDir returns the path to the jules config directory.
// Respects XDG_CONFIG_HOME on Linux, falls back to ~/.config/jules.
func ConfigDir() (string, error) {
	if runtime.GOOS == "linux" {
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
			return filepath.Join(xdg, "jules"), nil
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory: %w", err)
	}

	return filepath.Join(home, ".config", "jules"), nil
}

// ConfigPath returns the full path to config.toml.
func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.toml"), nil
}

// LoadConfig reads and parses the config file. Returns a zero-value Config
// (not an error) if the file does not exist yet.
func LoadConfig() (Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return Config{}, err
	}

	var cfg Config

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("reading config file: %w", err)
	}

	if _, err := toml.Decode(string(data), &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing config file: %w", err)
	}

	return cfg, nil
}

// SaveConfig writes cfg to disk, creating the config directory if needed.
func SaveConfig(cfg Config) error {
	dir, err := ConfigDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	path, err := ConfigPath()
	if err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("opening config file for write: %w", err)
	}
	defer f.Close()

	if err := toml.NewEncoder(f).Encode(cfg); err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}

	return nil
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	home, _ := os.UserHomeDir()
	return Config{
		Auth: AuthConfig{
			APIURL:  "https://api.jules.solutions",
			AuthURL: "https://auth.jules.solutions",
		},
		Local: LocalConfig{
			VaultPath: filepath.Join(home, "Jules.Life"),
		},
	}
}
