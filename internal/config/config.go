package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	DefaultConfigPath     = "graphgate.yaml"
	DefaultManifestFormat = "graphgate"
	DefaultManifestOutput = "./graphgate.manifest.json"
	DefaultReportsOutput  = "./graphgate/reports"
)

var ErrNoOperationMatches = errors.New("operation pattern matched no files")
var ErrNoFixtureMatches = errors.New("fixture pattern matched no files")

type Config struct {
	Path         string                       `yaml:"-"`
	BaseDir      string                       `yaml:"-"`
	Schema       string                       `yaml:"schema"`
	Operations   []string                     `yaml:"operations"`
	Manifest     ManifestConfig               `yaml:"manifest"`
	Environments map[string]EnvironmentConfig `yaml:"environments"`
	Profiles     map[string]ProfileConfig     `yaml:"profiles"`
	Tests        TestConfig                   `yaml:"tests"`
	Reports      ReportConfig                 `yaml:"reports"`
}

type ManifestConfig struct {
	Format string `yaml:"format"`
	Output string `yaml:"output"`
}

type EnvironmentConfig struct {
	Endpoint    string            `yaml:"endpoint"`
	Headers     map[string]string `yaml:"headers"`
	RequiredEnv []string          `yaml:"requiredEnv"`
	HealthCheck HealthCheckConfig `yaml:"healthCheck"`
}

type HealthCheckConfig struct {
	Method         string            `yaml:"method"`
	URL            string            `yaml:"url"`
	Path           string            `yaml:"path"`
	ExpectedStatus int               `yaml:"expectedStatus"`
	Timeout        string            `yaml:"timeout"`
	Headers        map[string]string `yaml:"headers"`
}

type ProfileConfig struct {
	Headers   map[string]string `yaml:"headers"`
	Variables map[string]any    `yaml:"variables"`
}

type TestConfig struct {
	Fixtures                 string                     `yaml:"fixtures"`
	RequireOperationCoverage bool                       `yaml:"requireOperationCoverage"`
	Suites                   map[string]TestSuiteConfig `yaml:"suites"`
	Coverage                 CoverageConfig             `yaml:"coverage"`
	Snapshot                 SnapshotConfig             `yaml:"snapshot"`
	Retry                    RetryConfig                `yaml:"retry"`
	MaxLatencyMs             int                        `yaml:"maxLatencyMs"`
}

type TestSuiteConfig struct {
	Tags         []string `yaml:"tags"`
	Exclude      []string `yaml:"exclude"`
	MaxLatencyMs int      `yaml:"maxLatencyMs"`
}

type CoverageConfig struct {
	RequirePositiveFixture     bool     `yaml:"requirePositiveFixture"`
	RequireTags                []string `yaml:"requireTags"`
	ForbidUnknownOperations    bool     `yaml:"forbidUnknownOperations"`
	ForbidDeprecatedOperations bool     `yaml:"forbidDeprecatedOperations"`
}

type SnapshotConfig struct {
	Mode        string   `yaml:"mode"`
	IgnorePaths []string `yaml:"ignorePaths"`
	RedactPaths []string `yaml:"redactPaths"`
}

type RetryConfig struct {
	MaxAttempts             int      `yaml:"maxAttempts"`
	Backoff                 string   `yaml:"backoff"`
	RetryableFailureClasses []string `yaml:"retryableFailureClasses"`
}

type ReportConfig struct {
	Output string `yaml:"output"`
}

func Load(path string) (*Config, error) {
	if path == "" {
		path = DefaultConfigPath
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve config path: %w", err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %q: %w", path, err)
	}

	cfg.Path = absPath
	cfg.BaseDir = filepath.Dir(absPath)
	cfg.applyDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func Template() []byte {
	return []byte(`schema: ./schema.graphql
operations:
  - ./operations/**/*.graphql
manifest:
  format: graphgate
  output: ./graphgate.manifest.json
environments:
  local:
    endpoint: http://localhost:8080/graphql
    headers:
      Authorization: Bearer ${GRAPHGATE_TOKEN}
profiles:
  local-user:
    headers:
      X-User-ID: ${GRAPHGATE_USER_ID}
tests:
  fixtures: ./graphgate/fixtures/**/*.json
  requireOperationCoverage: false
  suites:
    smoke:
      tags: [smoke]
reports:
  output: ./graphgate/reports
`)
}

func (c *Config) Validate() error {
	if strings.TrimSpace(c.Schema) == "" {
		return errors.New("config schema is required")
	}
	if strings.TrimSpace(c.Manifest.Format) != DefaultManifestFormat {
		return fmt.Errorf("config manifest.format must be %q", DefaultManifestFormat)
	}
	if strings.TrimSpace(c.Manifest.Output) == "" {
		return errors.New("config manifest.output is required")
	}
	if len(c.Operations) == 0 {
		return errors.New("config operations must include at least one pattern")
	}

	schemaPath := c.ResolvePath(c.Schema)
	info, err := os.Stat(schemaPath)
	if err != nil {
		return fmt.Errorf("schema file %q is not readable: %w", c.Schema, err)
	}
	if info.IsDir() {
		return fmt.Errorf("schema path %q is a directory", c.Schema)
	}

	if _, err := c.OperationFiles(); err != nil {
		return err
	}

	return nil
}

func (c *Config) ResolvePath(path string) string {
	path = os.ExpandEnv(strings.TrimSpace(path))
	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	baseDir := c.BaseDir
	if baseDir == "" {
		baseDir = "."
	}
	return filepath.Clean(filepath.Join(baseDir, path))
}

func (c *Config) RelativePath(path string) string {
	rel, err := filepath.Rel(c.BaseDir, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func (c *Config) ManifestOutputPath() string {
	return c.ResolvePath(c.Manifest.Output)
}

func (c *Config) ReportsOutputPath() string {
	return c.ResolvePath(c.Reports.Output)
}

func (c *Config) FixtureFiles() ([]string, error) {
	if strings.TrimSpace(c.Tests.Fixtures) == "" {
		return nil, errors.New("tests.fixtures is required")
	}
	files, err := c.matchFilePattern("fixture", c.Tests.Fixtures, ErrNoFixtureMatches)
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func (c *Config) OperationFiles() ([]string, error) {
	seen := map[string]struct{}{}
	var files []string
	for _, pattern := range c.Operations {
		matches, err := c.matchFilePattern("operation", pattern, ErrNoOperationMatches)
		if err != nil {
			if errors.Is(err, ErrNoOperationMatches) {
				continue
			}
			return nil, err
		}
		for _, match := range matches {
			if _, ok := seen[match]; ok {
				continue
			}
			seen[match] = struct{}{}
			files = append(files, match)
		}
	}
	sort.Strings(files)
	if len(files) == 0 {
		return nil, errors.New("operation patterns matched no files")
	}
	return files, nil
}

func (c *Config) applyDefaults() {
	if c.Manifest.Format == "" {
		c.Manifest.Format = DefaultManifestFormat
	}
	if c.Manifest.Output == "" {
		c.Manifest.Output = DefaultManifestOutput
	}
	if c.Reports.Output == "" {
		c.Reports.Output = DefaultReportsOutput
	}
}

func (c *Config) matchFilePattern(kind string, pattern string, noMatchesErr error) ([]string, error) {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return nil, fmt.Errorf("%s pattern cannot be empty", kind)
	}

	resolved := c.ResolvePath(pattern)
	if !hasGlob(resolved) {
		info, err := os.Stat(resolved)
		if err != nil {
			return nil, fmt.Errorf("%s file %q is not readable: %w", kind, pattern, err)
		}
		if info.IsDir() {
			return nil, fmt.Errorf("%s path %q is a directory", kind, pattern)
		}
		return []string{resolved}, nil
	}

	var matches []string
	var err error
	if strings.Contains(filepath.ToSlash(resolved), "**") {
		matches, err = recursiveGlob(resolved)
	} else {
		matches, err = filepath.Glob(resolved)
	}
	if err != nil {
		return nil, fmt.Errorf("%s glob %q is invalid: %w", kind, pattern, err)
	}

	matches = fileMatches(matches)
	if len(matches) == 0 {
		return nil, fmt.Errorf("%w: %q", noMatchesErr, pattern)
	}
	sort.Strings(matches)
	return matches, nil
}

func hasGlob(path string) bool {
	return strings.ContainsAny(path, "*?[")
}

func fileMatches(matches []string) []string {
	files := matches[:0]
	for _, match := range matches {
		info, err := os.Stat(match)
		if err == nil && !info.IsDir() {
			files = append(files, filepath.Clean(match))
		}
	}
	return files
}

func recursiveGlob(pattern string) ([]string, error) {
	slashPattern := filepath.ToSlash(filepath.Clean(pattern))
	re, err := globRegex(slashPattern)
	if err != nil {
		return nil, err
	}

	root := staticRoot(pattern)
	var matches []string
	if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if re.MatchString(filepath.ToSlash(filepath.Clean(path))) {
			matches = append(matches, filepath.Clean(path))
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return matches, nil
}

func staticRoot(pattern string) string {
	idx := strings.IndexAny(pattern, "*?[")
	if idx == -1 {
		return filepath.Dir(pattern)
	}
	prefix := pattern[:idx]
	if strings.HasSuffix(prefix, string(filepath.Separator)) {
		return filepath.Clean(prefix)
	}
	root := filepath.Dir(prefix)
	if root == "" {
		return "."
	}
	return filepath.Clean(root)
}

func globRegex(pattern string) (*regexp.Regexp, error) {
	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(pattern); {
		switch pattern[i] {
		case '*':
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				i += 2
				if i < len(pattern) && pattern[i] == '/' {
					b.WriteString("(?:.*/)?")
					i++
				} else {
					b.WriteString(".*")
				}
				continue
			}
			b.WriteString("[^/]*")
			i++
		case '?':
			b.WriteString("[^/]")
			i++
		case '[':
			return nil, errors.New("character classes are not supported with ** globs")
		default:
			b.WriteString(regexp.QuoteMeta(string(pattern[i])))
			i++
		}
	}
	b.WriteString("$")
	return regexp.Compile(b.String())
}
