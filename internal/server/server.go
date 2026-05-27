package server

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	"github.com/MarlonJD/graphgate/internal/config"
	"github.com/MarlonJD/graphgate/internal/core"
	"github.com/MarlonJD/graphgate/internal/report"
)

//go:embed static/*
var staticFS embed.FS

type Options struct {
	Addr        string
	OpenBrowser bool
}

type State struct {
	Validation core.ValidationResult `json:"validation"`
	Manifest   core.Manifest         `json:"manifest"`
	Schema     core.SchemaSummary    `json:"schema"`
	Reports    []ReportLink          `json:"reports"`
}

type ReportLink struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func Serve(ctx context.Context, cfg *config.Config, options Options) error {
	addr := options.Addr
	if addr == "" {
		addr = "127.0.0.1:4317"
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	mux, err := Handler(cfg)
	if err != nil {
		_ = listener.Close()
		return err
	}
	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	url := "http://" + listener.Addr().String()
	fmt.Printf("GraphGate UI: %s\n", url)
	if options.OpenBrowser {
		go openBrowser(url)
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
	err = server.Serve(listener)
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

func Handler(cfg *config.Config) (*http.ServeMux, error) {
	static, err := fs.Sub(staticFS, "static")
	if err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/state", func(w http.ResponseWriter, r *http.Request) {
		validation, err := core.ValidateProject(cfg)
		if err != nil {
			writeError(w, err)
			return
		}
		manifest := core.BuildManifest(cfg, validation)
		schema, issues, err := core.SummarizeSchema(cfg)
		if err != nil {
			writeError(w, err)
			return
		}
		if len(issues) > 0 {
			writeJSON(w, State{Validation: validation, Manifest: manifest})
			return
		}
		writeJSON(w, State{
			Validation: validation,
			Manifest:   manifest,
			Schema:     schema,
			Reports:    []ReportLink{{Name: "Validation report", Path: "/api/report/markdown"}},
		})
	})
	mux.HandleFunc("/api/report/markdown", func(w http.ResponseWriter, r *http.Request) {
		validation, err := core.ValidateProject(cfg)
		if err != nil {
			writeError(w, err)
			return
		}
		manifest := core.BuildManifest(cfg, validation)
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		_, _ = w.Write(report.RenderMarkdown(report.NewSummary(validation, manifest, cfg.RelativePath(cfg.ManifestOutputPath()))))
	})
	mux.Handle("/", http.FileServer(http.FS(static)))
	return mux, nil
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}
