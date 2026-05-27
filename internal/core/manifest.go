package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/MarlonJD/graphgate/internal/config"
)

type Manifest struct {
	Format     string              `json:"format"`
	Version    int                 `json:"version"`
	Operations []ManifestOperation `json:"operations"`
}

type ManifestOperation struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	File   string `json:"file"`
	SHA256 string `json:"sha256"`
	Body   string `json:"body"`
}

func BuildManifest(cfg *config.Config, result ValidationResult) Manifest {
	manifest := Manifest{
		Format:  cfg.Manifest.Format,
		Version: 1,
	}
	for _, operation := range result.Operations {
		manifest.Operations = append(manifest.Operations, ManifestOperation{
			ID:     operation.ID,
			Name:   operation.Name,
			File:   operation.File,
			SHA256: operation.SHA256,
			Body:   operation.Normalized,
		})
	}
	return manifest
}

func ManifestBytes(manifest Manifest) ([]byte, error) {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func WriteManifest(path string, manifest Manifest) error {
	data, err := ManifestBytes(manifest)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func CheckManifest(path string, manifest Manifest) (bool, error) {
	expected, err := ManifestBytes(manifest)
	if err != nil {
		return false, err
	}
	actual, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("read manifest %q: %w", path, err)
	}
	return bytes.Equal(actual, expected), nil
}
