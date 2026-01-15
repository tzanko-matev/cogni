//go:build cucumber

package reportserver

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cucumber/godog"
)

// TestServeReportScenarios runs the report server feature scenarios.
func TestServeReportScenarios(t *testing.T) {
	featurePath := filepath.Join("..", "..", "spec", "features", "output-report-serve.feature")
	suite := godog.TestSuite{
		Name:                "output-report-serve",
		ScenarioInitializer: InitializeServeScenario,
		Options: &godog.Options{
			Format:    "pretty",
			Paths:     []string{featurePath},
			Strict:    true,
			TestingT:  t,
			Randomize: 0,
		},
	}
	if suite.Run() != 0 {
		t.Fatalf("non-zero godog status")
	}
}

// InitializeServeScenario wires steps for report server feature scenarios.
func InitializeServeScenario(ctx *godog.ScenarioContext) {
	state := &serveScenarioState{}
	ctx.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		state.reset()
		return ctx, nil
	})

	ctx.Step(`^a DuckDB report file$`, state.givenDuckDBReportFile)
	ctx.Step(`^an assets base URL "([^"]+)"$`, state.givenAssetsBaseURL)
	ctx.Step(`^I start the report server$`, state.whenIStartTheReportServer)
	ctx.Step(`^I request "([^"]+)"$`, state.whenIRequest)
	ctx.Step(`^the response status is (\d+)$`, state.thenResponseStatus)
	ctx.Step(`^the response body contains "([^"]+)"$`, state.thenResponseBodyContains)
	ctx.Step(`^the response body equals the DuckDB file bytes$`, state.thenResponseBodyEqualsDB)
}

// serveScenarioState holds scenario state for report server feature tests.
type serveScenarioState struct {
	dbPath        string
	dbContents    []byte
	assetsBaseURL string
	handler       http.Handler
	response      *httptest.ResponseRecorder
}

// reset clears scenario state.
func (s *serveScenarioState) reset() {
	s.dbPath = ""
	s.dbContents = nil
	s.assetsBaseURL = ""
	s.handler = nil
	s.response = nil
}

// givenDuckDBReportFile creates a temporary DuckDB file for the scenario.
func (s *serveScenarioState) givenDuckDBReportFile() error {
	content := []byte("duckdb")
	path, err := writeTempFile("report-*.duckdb", content)
	if err != nil {
		return err
	}
	s.dbPath = path
	s.dbContents = content
	return nil
}

// givenAssetsBaseURL sets the asset base URL for the scenario.
func (s *serveScenarioState) givenAssetsBaseURL(url string) error {
	s.assetsBaseURL = url
	return nil
}

// whenIStartTheReportServer builds the report handler with the scenario config.
func (s *serveScenarioState) whenIStartTheReportServer() error {
	if s.dbPath == "" {
		return fmt.Errorf("db path is not set")
	}
	handler, err := NewHandler(Config{
		DBPath:        s.dbPath,
		AssetsBaseURL: s.assetsBaseURL,
	})
	if err != nil {
		return err
	}
	s.handler = handler
	return nil
}

// whenIRequest sends a request to the report handler.
func (s *serveScenarioState) whenIRequest(path string) error {
	if s.handler == nil {
		return fmt.Errorf("handler not initialized")
	}
	req := httptest.NewRequest(http.MethodGet, "http://example.com"+path, nil)
	recorder := httptest.NewRecorder()
	s.handler.ServeHTTP(recorder, req)
	s.response = recorder
	return nil
}

// thenResponseStatus asserts the HTTP response status code.
func (s *serveScenarioState) thenResponseStatus(expected int) error {
	if s.response == nil {
		return fmt.Errorf("response not recorded")
	}
	if s.response.Code != expected {
		return fmt.Errorf("expected status %d, got %d", expected, s.response.Code)
	}
	return nil
}

// thenResponseBodyContains asserts the response body includes the given substring.
func (s *serveScenarioState) thenResponseBodyContains(snippet string) error {
	if s.response == nil {
		return fmt.Errorf("response not recorded")
	}
	if !strings.Contains(s.response.Body.String(), snippet) {
		return fmt.Errorf("expected response to contain %q", snippet)
	}
	return nil
}

// thenResponseBodyEqualsDB asserts the response body matches the DuckDB bytes.
func (s *serveScenarioState) thenResponseBodyEqualsDB() error {
	if s.response == nil {
		return fmt.Errorf("response not recorded")
	}
	if s.dbContents == nil {
		return fmt.Errorf("db contents not set")
	}
	if got := s.response.Body.Bytes(); string(got) != string(s.dbContents) {
		return fmt.Errorf("response body did not match db bytes")
	}
	return nil
}

// writeTempFile writes a temporary file with the provided contents.
func writeTempFile(pattern string, contents []byte) (string, error) {
	file, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", err
	}
	defer file.Close()
	if _, err := file.Write(contents); err != nil {
		return "", err
	}
	return file.Name(), nil
}
