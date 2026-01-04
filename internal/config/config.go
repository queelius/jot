// Package config handles jot configuration loading and management.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the jot configuration.
type Config struct {
	// Journal metadata
	Name        string `yaml:"name,omitempty"`
	Description string `yaml:"description,omitempty"`

	// Default values for new entries
	Defaults Defaults `yaml:"defaults,omitempty"`

	// Editor preference (falls back to $EDITOR)
	Editor string `yaml:"editor,omitempty"`

	// Output preferences
	Output OutputConfig `yaml:"output,omitempty"`

	// Date format for display
	DateFormat string `yaml:"date_format,omitempty"`
}

// Defaults contains default values for new entries.
type Defaults struct {
	Type string   `yaml:"type,omitempty"`
	Tags []string `yaml:"tags,omitempty"`
}

// OutputConfig contains output formatting preferences.
type OutputConfig struct {
	Format string `yaml:"format,omitempty"` // json, markdown, table
	Color  string `yaml:"color,omitempty"`  // auto, always, never
}

// DefaultConfig returns a config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Name:       "My Journal",
		Defaults:   Defaults{Type: "note"},
		Editor:     "",
		DateFormat: "2006-01-02",
		Output: OutputConfig{
			Format: "table",
			Color:  "auto",
		},
	}
}

// Load reads the configuration from the given root.
// Checks both root/.jot/config.yaml (local journals) and root/config.yaml (global journal).
func Load(root string) (*Config, error) {
	// Try local journal path first: root/.jot/config.yaml
	configPath := filepath.Join(root, ".jot", "config.yaml")
	data, err := os.ReadFile(configPath)

	if os.IsNotExist(err) {
		// Try global journal path: root/config.yaml
		configPath = filepath.Join(root, "config.yaml")
		data, err = os.ReadFile(configPath)
	}

	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return cfg, nil
}

// Save writes the configuration to .jot/config.yaml.
func (c *Config) Save(root string) error {
	configDir := filepath.Join(root, ".jot")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}

// RootInfo contains information about the resolved journal root.
type RootInfo struct {
	Path     string // The journal root path
	IsGlobal bool   // True if using ~/.jot global journal
}

// FindRoot walks up from the current directory to find a .jot directory.
// If no local .jot is found, falls back to ~/.jot (global journal).
// The global journal is auto-initialized if it doesn't exist.
func FindRoot() (string, error) {
	info, err := FindRootWithInfo()
	if err != nil {
		return "", err
	}
	return info.Path, nil
}

// FindRootWithInfo is like FindRoot but returns additional metadata.
func FindRootWithInfo() (*RootInfo, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getting working directory: %w", err)
	}

	// Get global root path to avoid treating it as a local journal
	globalRoot, err := GlobalRoot()
	if err != nil {
		return nil, err
	}

	home, _ := os.UserHomeDir()

	// Walk up looking for .jot
	for {
		jotDir := filepath.Join(dir, ".jot")
		if info, err := os.Stat(jotDir); err == nil && info.IsDir() {
			// Skip if this is the home directory and .jot is the global journal
			// (we don't want to treat ~ as a local journal root)
			if dir == home && jotDir == globalRoot {
				// This is the global journal, not a local one - fall through
			} else {
				return &RootInfo{Path: dir, IsGlobal: false}, nil
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root, fall back to global
			break
		}
		dir = parent
	}

	// No local .jot found, use global
	// Auto-initialize global journal if it doesn't exist
	configPath := filepath.Join(globalRoot, "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := initGlobalJournal(globalRoot); err != nil {
			return nil, fmt.Errorf("initializing global journal: %w", err)
		}
	}

	return &RootInfo{Path: globalRoot, IsGlobal: true}, nil
}

// GlobalRoot returns the path to the global journal (~/.jot).
func GlobalRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}
	return filepath.Join(home, ".jot"), nil
}

// initGlobalJournal creates the global journal structure.
// Global journal has a simpler structure: ~/.jot/config.yaml, ~/.jot/entries/
func initGlobalJournal(root string) error {
	// Create the root directory
	if err := os.MkdirAll(root, 0755); err != nil {
		return err
	}

	// Create entries directory
	entriesDir := filepath.Join(root, "entries")
	if err := os.MkdirAll(entriesDir, 0755); err != nil {
		return err
	}

	// Create default config directly in root (not in .jot subdirectory)
	cfg := DefaultConfig()
	cfg.Name = "Global Journal"

	configPath := filepath.Join(root, "config.yaml")
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	return os.WriteFile(configPath, data, 0644)
}

// GetEditor returns the configured editor, falling back to $EDITOR or "vi".
func (c *Config) GetEditor() string {
	if c.Editor != "" {
		return c.Editor
	}
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}
	if editor := os.Getenv("VISUAL"); editor != "" {
		return editor
	}
	return "vi"
}

// Set updates a config value by dot-notation key.
func (c *Config) Set(key, value string) error {
	switch key {
	case "name":
		c.Name = value
	case "description":
		c.Description = value
	case "editor":
		c.Editor = value
	case "date_format":
		c.DateFormat = value
	case "defaults.type":
		c.Defaults.Type = value
	case "output.format":
		c.Output.Format = value
	case "output.color":
		c.Output.Color = value
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}
	return nil
}

// Get retrieves a config value by dot-notation key.
func (c *Config) Get(key string) (string, error) {
	switch key {
	case "name":
		return c.Name, nil
	case "description":
		return c.Description, nil
	case "editor":
		return c.Editor, nil
	case "date_format":
		return c.DateFormat, nil
	case "defaults.type":
		return c.Defaults.Type, nil
	case "output.format":
		return c.Output.Format, nil
	case "output.color":
		return c.Output.Color, nil
	default:
		return "", fmt.Errorf("unknown config key: %s", key)
	}
}
