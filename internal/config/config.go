package config

import (
	"encoding/json"
	"goctx/internal/model"
	"os"
	"path/filepath"
)

const (
	OverlayOpacity25 = 0.25
	OverlayOpacity50 = 0.50
	OverlayOpacity75 = 0.75
	OverlayOpacity100 = 1.00
)

func Load(root string) (model.Config, error) {
	var cfg model.Config
	// Priority: goctx.json -> ctx.json
	files := []string{"goctx.json", "ctx.json"}

	for _, file := range files {
		path := filepath.Join(root, file)
		if _, err := os.Stat(path); err == nil {
			data, err := os.ReadFile(path)
			if err != nil {
				return cfg, err
			}
			if err := json.Unmarshal(data, &cfg); err == nil {
				return cfg, nil
			}
		}
	}
	return cfg, nil
}

// Save writes the configuration back to goctx.json
func Save(root string, cfg model.Config) error {
	path := filepath.Join(root, "goctx.json")
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func LoadKeys(root string) (map[string]string, error) {
	path := filepath.Join(root, "keys.json")
	if _, err := os.Stat(path); err != nil {
		return make(map[string]string), nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var keys map[string]string
	err = json.Unmarshal(data, &keys)
	return keys, err
}

func SaveKeys(root string, keys map[string]string) error {
	path := filepath.Join(root, "keys.json")
	data, err := json.MarshalIndent(keys, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
