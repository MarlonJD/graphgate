package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/MarlonJD/graphgate/internal/config"
)

func TestHandlerServesHealthAndStaticUI(t *testing.T) {
	cfg := testConfig(t)
	handler, err := Handler(cfg)
	if err != nil {
		t.Fatalf("Handler() error = %v", err)
	}

	for _, path := range []string{"/healthz", "/"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status = %d", path, rec.Code)
		}
	}
}

func testConfig(t *testing.T) *config.Config {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, dir, "schema.graphql", "type Query { viewer: String! }\n")
	writeFile(t, dir, "operations/GetViewer.graphql", "query GetViewer { viewer }\n")
	writeFile(t, dir, "graphgate.yaml", "schema: ./schema.graphql\noperations:\n  - ./operations/**/*.graphql\nmanifest:\n  format: graphgate\n  output: ./graphgate.manifest.json\n")
	cfg, err := config.Load(filepath.Join(dir, "graphgate.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func writeFile(t *testing.T, dir string, name string, data string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}
