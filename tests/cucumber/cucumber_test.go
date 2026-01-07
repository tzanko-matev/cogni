//go:build cucumber
// +build cucumber

package cucumber

import (
	"io"
	"path/filepath"
	"testing"

	"cogni/internal/testutil"
	"github.com/cucumber/godog"
)

// TestCucumberFeatures runs the cucumber feature suite.
func TestCucumberFeatures(t *testing.T) {
	ctx := testutil.Context(t, 0)
	featuresPath := filepath.Join("..", "..", "spec", "features")
	options := godog.Options{
		Format:         "progress",
		Paths:          []string{featuresPath},
		Tags:           "@smoke",
		Output:         io.Discard,
		TestingT:       t,
		Randomize:      0,
		DefaultContext: ctx,
	}

	suite := godog.TestSuite{
		Name:                "cogni-features",
		ScenarioInitializer: InitializeScenario,
		Options:             &options,
	}

	if suite.Run() != 0 {
		t.Fatalf("cucumber features failed")
	}
}
