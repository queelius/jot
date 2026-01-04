package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Name != "My Journal" {
		t.Errorf("Name = %q, want %q", cfg.Name, "My Journal")
	}
	if cfg.Defaults.Type != "note" {
		t.Errorf("Defaults.Type = %q, want %q", cfg.Defaults.Type, "note")
	}
	if cfg.Output.Format != "table" {
		t.Errorf("Output.Format = %q, want %q", cfg.Output.Format, "table")
	}
	if cfg.Output.Color != "auto" {
		t.Errorf("Output.Color = %q, want %q", cfg.Output.Color, "auto")
	}
	if cfg.DateFormat != "2006-01-02" {
		t.Errorf("DateFormat = %q, want %q", cfg.DateFormat, "2006-01-02")
	}
}

func TestConfig_LoadAndSave(t *testing.T) {
	root := t.TempDir()

	// Create a config and save it
	cfg := &Config{
		Name:        "Test Journal",
		Description: "A test journal",
		Editor:      "vim",
		DateFormat:  "2006-01-02",
		Defaults: Defaults{
			Type: "task",
			Tags: []string{"test"},
		},
		Output: OutputConfig{
			Format: "json",
			Color:  "never",
		},
	}

	if err := cfg.Save(root); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file was created
	configPath := filepath.Join(root, ".jot", "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	// Load it back
	loaded, err := Load(root)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.Name != cfg.Name {
		t.Errorf("Name = %q, want %q", loaded.Name, cfg.Name)
	}
	if loaded.Description != cfg.Description {
		t.Errorf("Description = %q, want %q", loaded.Description, cfg.Description)
	}
	if loaded.Editor != cfg.Editor {
		t.Errorf("Editor = %q, want %q", loaded.Editor, cfg.Editor)
	}
	if loaded.Defaults.Type != cfg.Defaults.Type {
		t.Errorf("Defaults.Type = %q, want %q", loaded.Defaults.Type, cfg.Defaults.Type)
	}
	if loaded.Output.Format != cfg.Output.Format {
		t.Errorf("Output.Format = %q, want %q", loaded.Output.Format, cfg.Output.Format)
	}
}

func TestConfig_LoadDefault(t *testing.T) {
	root := t.TempDir()

	// Load from empty directory should return defaults
	cfg, err := Load(root)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Name != "My Journal" {
		t.Errorf("Name = %q, want default %q", cfg.Name, "My Journal")
	}
}

func TestConfig_LoadGlobalJournalPath(t *testing.T) {
	root := t.TempDir()

	// Create config.yaml directly in root (global journal style)
	configContent := `name: Global Journal
defaults:
  type: idea
`
	if err := os.WriteFile(filepath.Join(root, "config.yaml"), []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err := Load(root)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Name != "Global Journal" {
		t.Errorf("Name = %q, want %q", cfg.Name, "Global Journal")
	}
	if cfg.Defaults.Type != "idea" {
		t.Errorf("Defaults.Type = %q, want %q", cfg.Defaults.Type, "idea")
	}
}

func TestConfig_GetEditor(t *testing.T) {
	tests := []struct {
		name       string
		cfgEditor  string
		envEditor  string
		envVisual  string
		want       string
	}{
		{"config editor", "nano", "", "", "nano"},
		{"env EDITOR", "", "emacs", "", "emacs"},
		{"env VISUAL", "", "", "code", "code"},
		{"default vi", "", "", "", "vi"},
		{"config overrides env", "nano", "emacs", "code", "nano"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore env
			oldEditor := os.Getenv("EDITOR")
			oldVisual := os.Getenv("VISUAL")
			defer func() {
				os.Setenv("EDITOR", oldEditor)
				os.Setenv("VISUAL", oldVisual)
			}()

			os.Setenv("EDITOR", tt.envEditor)
			os.Setenv("VISUAL", tt.envVisual)

			cfg := &Config{Editor: tt.cfgEditor}
			got := cfg.GetEditor()

			if got != tt.want {
				t.Errorf("GetEditor() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestConfig_SetAndGet(t *testing.T) {
	cfg := DefaultConfig()

	tests := []struct {
		key   string
		value string
	}{
		{"name", "New Name"},
		{"description", "New Description"},
		{"editor", "emacs"},
		{"date_format", "01/02/2006"},
		{"defaults.type", "task"},
		{"output.format", "json"},
		{"output.color", "always"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			if err := cfg.Set(tt.key, tt.value); err != nil {
				t.Fatalf("Set(%q, %q) error = %v", tt.key, tt.value, err)
			}

			got, err := cfg.Get(tt.key)
			if err != nil {
				t.Fatalf("Get(%q) error = %v", tt.key, err)
			}

			if got != tt.value {
				t.Errorf("Get(%q) = %q, want %q", tt.key, got, tt.value)
			}
		})
	}
}

func TestConfig_SetInvalidKey(t *testing.T) {
	cfg := DefaultConfig()

	err := cfg.Set("invalid.key", "value")
	if err == nil {
		t.Error("Set() should fail for invalid key")
	}
}

func TestConfig_GetInvalidKey(t *testing.T) {
	cfg := DefaultConfig()

	_, err := cfg.Get("invalid.key")
	if err == nil {
		t.Error("Get() should fail for invalid key")
	}
}

func TestGlobalRoot(t *testing.T) {
	root, err := GlobalRoot()
	if err != nil {
		t.Fatalf("GlobalRoot() error = %v", err)
	}

	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".jot")

	if root != want {
		t.Errorf("GlobalRoot() = %q, want %q", root, want)
	}
}

func TestFindRoot_LocalJournal(t *testing.T) {
	// Create a temp directory structure with a .jot folder
	root := t.TempDir()
	jotDir := filepath.Join(root, "project", ".jot")
	if err := os.MkdirAll(jotDir, 0755); err != nil {
		t.Fatalf("failed to create .jot dir: %v", err)
	}

	// Create a subdirectory to test walking up
	subDir := filepath.Join(root, "project", "src", "components")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdirectory: %v", err)
	}

	// Change to subdirectory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)

	if err := os.Chdir(subDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	info, err := FindRootWithInfo()
	if err != nil {
		t.Fatalf("FindRootWithInfo() error = %v", err)
	}

	expectedRoot := filepath.Join(root, "project")
	if info.Path != expectedRoot {
		t.Errorf("Path = %q, want %q", info.Path, expectedRoot)
	}
	if info.IsGlobal {
		t.Error("IsGlobal should be false for local journal")
	}
}

func TestRootInfo(t *testing.T) {
	info := &RootInfo{
		Path:     "/home/user/project",
		IsGlobal: false,
	}

	if info.Path != "/home/user/project" {
		t.Errorf("Path = %q, want %q", info.Path, "/home/user/project")
	}
	if info.IsGlobal != false {
		t.Errorf("IsGlobal = %v, want false", info.IsGlobal)
	}
}
