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
