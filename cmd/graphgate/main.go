package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/MarlonJD/graphgate/internal/config"
	"github.com/MarlonJD/graphgate/internal/core"
	"github.com/MarlonJD/graphgate/internal/report"
	"github.com/MarlonJD/graphgate/internal/server"
)

const (
	exitOK               = 0
	exitInternalError    = 1
	exitInvalidConfig    = 2
	exitInvalidSchema    = 3
	exitInvalidOperation = 4
	exitManifestMismatch = 5
	exitBreakingChange   = 6
	exitTestFailure      = 7
)

func main() {
	os.Exit(run(os.Args, os.Stdout, os.Stderr))
}

func run(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) < 2 {
		printUsage(stderr)
		return exitInvalidConfig
	}

	switch args[1] {
	case "help", "-h", "--help":
		printUsage(stdout)
		return exitOK
	case "init":
		return runInit(args[2:], stdout, stderr)
	case "validate":
		return runValidate(args[2:], stdout, stderr)
	case "manifest":
		return runManifest(args[2:], stdout, stderr)
	case "report":
		return runReport(args[2:], stdout, stderr)
	case "diff":
		return runDiff(args[2:], stdout, stderr)
	case "test":
		return runTest(args[2:], stdout, stderr)
	case "ui":
		return runUI(args[2:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command %q\n\n", args[1])
		printUsage(stderr)
		return exitInvalidConfig
	}
}

func runDiff(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := newFlagSet("diff", stderr)
	configPath := fs.String("config", config.DefaultConfigPath, "path to graphgate.yaml")
	base := fs.String("base", "", "path to the base schema")
	format := fs.String("format", "markdown", "report format: markdown or json")
	output := fs.String("output", "", "optional report output path")
	if err := fs.Parse(args); err != nil {
		return exitInvalidConfig
	}
	if strings.TrimSpace(*base) == "" {
		fmt.Fprintln(stderr, "diff failed: --base is required")
		return exitInvalidConfig
	}
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(stderr, "config error: %v\n", err)
		return exitInvalidConfig
	}
	result, err := core.DiffSchemas(cfg, *base)
	if err != nil {
		fmt.Fprintf(stderr, "diff failed: %v\n", err)
		return exitInternalError
	}
	data, err := renderDiffReport(*format, result)
	if err != nil {
		fmt.Fprintf(stderr, "diff failed: %v\n", err)
		return exitInvalidConfig
	}
	if code := writeReportData(cfg, *output, data, stdout, stderr); code != exitOK {
		return code
	}
	if !result.OK {
		return exitBreakingChange
	}
	return exitOK
}

func runTest(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := newFlagSet("test", stderr)
	configPath := fs.String("config", config.DefaultConfigPath, "path to graphgate.yaml")
	env := fs.String("env", "local", "environment name from graphgate.yaml")
	update := fs.Bool("update", false, "update fixture snapshots with latest responses")
	format := fs.String("format", "markdown", "report format: markdown or json")
	output := fs.String("output", "", "optional report output path")
	timeout := fs.Duration("timeout", 10*time.Second, "HTTP timeout per fixture")
	if err := fs.Parse(args); err != nil {
		return exitInvalidConfig
	}
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(stderr, "config error: %v\n", err)
		return exitInvalidConfig
	}
	result, err := core.RunContractTests(context.Background(), cfg, core.TestOptions{
		Environment: *env,
		Update:      *update,
		Timeout:     *timeout,
	})
	if err != nil {
		fmt.Fprintf(stderr, "test failed: %v\n", err)
		return exitInternalError
	}
	data, err := renderTestReport(*format, result)
	if err != nil {
		fmt.Fprintf(stderr, "test failed: %v\n", err)
		return exitInvalidConfig
	}
	if code := writeReportData(cfg, *output, data, stdout, stderr); code != exitOK {
		return code
	}
	if !result.OK {
		return exitTestFailure
	}
	return exitOK
}

func runUI(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := newFlagSet("ui", stderr)
	configPath := fs.String("config", config.DefaultConfigPath, "path to graphgate.yaml")
	addr := fs.String("addr", "127.0.0.1:4317", "local UI listen address")
	noOpen := fs.Bool("no-open", false, "do not open the browser automatically")
	if err := fs.Parse(args); err != nil {
		return exitInvalidConfig
	}
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(stderr, "config error: %v\n", err)
		return exitInvalidConfig
	}
	if err := server.Serve(context.Background(), cfg, server.Options{Addr: *addr, OpenBrowser: !*noOpen}); err != nil {
		fmt.Fprintf(stderr, "ui failed: %v\n", err)
		return exitInternalError
	}
	_, _ = stdout.Write([]byte{})
	return exitOK
}

func runInit(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := newFlagSet("init", stderr)
	force := fs.Bool("force", false, "overwrite graphgate.yaml if it already exists")
	if err := fs.Parse(args); err != nil {
		return exitInvalidConfig
	}

	if err := mkdirs("operations", "graphgate/fixtures", "graphgate/reports"); err != nil {
		fmt.Fprintf(stderr, "init failed: %v\n", err)
		return exitInternalError
	}

	if err := writeConfigTemplate(config.DefaultConfigPath, *force); err != nil {
		fmt.Fprintf(stderr, "init failed: %v\n", err)
		return exitInvalidConfig
	}

	fmt.Fprintln(stdout, "GraphGate initialized")
	fmt.Fprintln(stdout, "Created graphgate.yaml, operations/, graphgate/fixtures/, and graphgate/reports/")
	return exitOK
}

func runValidate(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := newFlagSet("validate", stderr)
	configPath := fs.String("config", config.DefaultConfigPath, "path to graphgate.yaml")
	if err := fs.Parse(args); err != nil {
		return exitInvalidConfig
	}

	cfg, result, code := loadAndValidate(*configPath, stderr)
	if code != exitOK {
		return code
	}
	if !result.OK() {
		printIssues(stderr, result.Issues)
		return validationExitCode(result)
	}

	fmt.Fprintf(stdout, "GraphGate validate passed: %d operations across %d files\n", len(result.Operations), len(result.OperationFiles))
	fmt.Fprintf(stdout, "Schema: %s\n", result.Schema)
	fmt.Fprintf(stdout, "Manifest output: %s\n", cfg.RelativePath(cfg.ManifestOutputPath()))
	return exitOK
}

func runManifest(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := newFlagSet("manifest", stderr)
	configPath := fs.String("config", config.DefaultConfigPath, "path to graphgate.yaml")
	check := fs.Bool("check", false, "fail if the committed manifest differs from generated output")
	if err := fs.Parse(args); err != nil {
		return exitInvalidConfig
	}

	cfg, result, code := loadAndValidate(*configPath, stderr)
	if code != exitOK {
		return code
	}
	if !result.OK() {
		printIssues(stderr, result.Issues)
		return validationExitCode(result)
	}

	manifest := core.BuildManifest(cfg, result)
	outputPath := cfg.ManifestOutputPath()
	if *check {
		matches, err := core.CheckManifest(outputPath, manifest)
		if err != nil {
			fmt.Fprintf(stderr, "manifest check failed: %v\n", err)
			return exitManifestMismatch
		}
		if !matches {
			fmt.Fprintf(stderr, "manifest check failed: %s is stale\n", cfg.RelativePath(outputPath))
			return exitManifestMismatch
		}
		fmt.Fprintf(stdout, "Manifest is up to date: %s\n", cfg.RelativePath(outputPath))
		return exitOK
	}

	if err := core.WriteManifest(outputPath, manifest); err != nil {
		fmt.Fprintf(stderr, "manifest write failed: %v\n", err)
		return exitInternalError
	}
	fmt.Fprintf(stdout, "Wrote manifest: %s (%d operations)\n", cfg.RelativePath(outputPath), len(manifest.Operations))
	return exitOK
}

func runReport(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := newFlagSet("report", stderr)
	configPath := fs.String("config", config.DefaultConfigPath, "path to graphgate.yaml")
	format := fs.String("format", "markdown", "report format: markdown or json")
	output := fs.String("output", "", "optional report output path")
	if err := fs.Parse(args); err != nil {
		return exitInvalidConfig
	}

	cfg, result, code := loadAndValidate(*configPath, stderr)
	if code != exitOK {
		return code
	}

	manifest := core.BuildManifest(cfg, result)
	summary := report.NewSummary(result, manifest, cfg.RelativePath(cfg.ManifestOutputPath()))
	data, err := renderReport(*format, summary)
	if err != nil {
		fmt.Fprintf(stderr, "report failed: %v\n", err)
		return exitInvalidConfig
	}

	if strings.TrimSpace(*output) != "" {
		outputPath := cfg.ResolvePath(*output)
		if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
			fmt.Fprintf(stderr, "report failed: %v\n", err)
			return exitInternalError
		}
		if err := os.WriteFile(outputPath, data, 0o644); err != nil {
			fmt.Fprintf(stderr, "report failed: %v\n", err)
			return exitInternalError
		}
		fmt.Fprintf(stdout, "Wrote report: %s\n", cfg.RelativePath(outputPath))
	} else {
		_, _ = stdout.Write(data)
	}

	if !result.OK() {
		return validationExitCode(result)
	}
	return exitOK
}

func loadAndValidate(configPath string, stderr io.Writer) (*config.Config, core.ValidationResult, int) {
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(stderr, "config error: %v\n", err)
		return nil, core.ValidationResult{}, exitInvalidConfig
	}

	result, err := core.ValidateProject(cfg)
	if err != nil {
		fmt.Fprintf(stderr, "validation failed: %v\n", err)
		return cfg, result, exitInternalError
	}
	return cfg, result, exitOK
}

func validationExitCode(result core.ValidationResult) int {
	if result.HasSchemaIssues() {
		return exitInvalidSchema
	}
	if result.HasOperationIssues() {
		return exitInvalidOperation
	}
	return exitOK
}

func renderReport(format string, summary report.Summary) ([]byte, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "markdown", "md":
		return report.RenderMarkdown(summary), nil
	case "json":
		return report.RenderJSON(summary)
	default:
		return nil, fmt.Errorf("unsupported report format %q", format)
	}
}

func renderDiffReport(format string, result core.DiffResult) ([]byte, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "markdown", "md":
		return report.RenderDiffMarkdown(result), nil
	case "json":
		return report.RenderDiffJSON(result)
	default:
		return nil, fmt.Errorf("unsupported report format %q", format)
	}
}

func renderTestReport(format string, result core.TestRunResult) ([]byte, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "markdown", "md":
		return report.RenderTestMarkdown(result), nil
	case "json":
		return report.RenderTestJSON(result)
	default:
		return nil, fmt.Errorf("unsupported report format %q", format)
	}
}

func writeReportData(cfg *config.Config, output string, data []byte, stdout io.Writer, stderr io.Writer) int {
	if strings.TrimSpace(output) == "" {
		_, _ = stdout.Write(data)
		return exitOK
	}
	outputPath := cfg.ResolvePath(output)
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		fmt.Fprintf(stderr, "report write failed: %v\n", err)
		return exitInternalError
	}
	if err := os.WriteFile(outputPath, data, 0o644); err != nil {
		fmt.Fprintf(stderr, "report write failed: %v\n", err)
		return exitInternalError
	}
	fmt.Fprintf(stdout, "Wrote report: %s\n", cfg.RelativePath(outputPath))
	return exitOK
}

func newFlagSet(name string, stderr io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	return fs
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "GraphGate catches GraphQL contract breaks before they ship.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  graphgate init [--force]")
	fmt.Fprintln(w, "  graphgate validate [--config graphgate.yaml]")
	fmt.Fprintln(w, "  graphgate manifest [--config graphgate.yaml] [--check]")
	fmt.Fprintln(w, "  graphgate diff --base old-schema.graphql [--config graphgate.yaml] [--format markdown|json]")
	fmt.Fprintln(w, "  graphgate test [--config graphgate.yaml] [--env local] [--update]")
	fmt.Fprintln(w, "  graphgate report [--config graphgate.yaml] [--format markdown|json] [--output path]")
	fmt.Fprintln(w, "  graphgate ui [--config graphgate.yaml] [--addr 127.0.0.1:4317]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Exit codes:")
	fmt.Fprintf(w, "  %d invalid config\n", exitInvalidConfig)
	fmt.Fprintf(w, "  %d invalid schema\n", exitInvalidSchema)
	fmt.Fprintf(w, "  %d invalid operation\n", exitInvalidOperation)
	fmt.Fprintf(w, "  %d manifest mismatch\n", exitManifestMismatch)
	fmt.Fprintf(w, "  %d breaking schema change\n", exitBreakingChange)
	fmt.Fprintf(w, "  %d contract test failure\n", exitTestFailure)
}

func printIssues(w io.Writer, issues []core.Issue) {
	for _, issue := range issues {
		location := issue.File
		if issue.Line > 0 {
			location = fmt.Sprintf("%s:%d", location, issue.Line)
			if issue.Column > 0 {
				location = fmt.Sprintf("%s:%d", location, issue.Column)
			}
		}
		if issue.Operation != "" {
			fmt.Fprintf(w, "[%s] %s %s: %s\n", issue.Code, location, issue.Operation, issue.Message)
			continue
		}
		fmt.Fprintf(w, "[%s] %s: %s\n", issue.Code, location, issue.Message)
	}
}

func mkdirs(paths ...string) error {
	for _, path := range paths {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func writeConfigTemplate(path string, force bool) error {
	if !force {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("%s already exists; use --force to overwrite it", path)
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	return os.WriteFile(path, config.Template(), 0o644)
}
