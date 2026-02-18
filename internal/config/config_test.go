package config

import (
	"goctx/internal/model"
	"os"
	"testing"
)

func TestSaveAndLoadConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "goctx_config_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := model.Config{
		Ignore: []string{"custom_ignore"},
		Scripts: model.Scripts{
			Build: "go build .",
			Test:  "go test ./...",
		},
	}

	err = Save(tmpDir, cfg)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.Scripts.Build != cfg.Scripts.Build {
		t.Errorf("Expected build script %q, got %q", cfg.Scripts.Build, loaded.Scripts.Build)
	}

	if len(loaded.Ignore) != 1 || loaded.Ignore[0] != "custom_ignore" {
		t.Errorf("Ignore list mismatch: %v", loaded.Ignore)
	}
}
