package cucumber

import (
	"io"
	"path/filepath"
	"testing"

	"github.com/cucumber/godog"
)

func TestCucumberFeatures(t *testing.T) {
	featuresPath := filepath.Join("..", "..", "spec", "features")
	options := godog.Options{
		Format:    "progress",
		Paths:     []string{featuresPath},
		Tags:      "@smoke",
		Output:    io.Discard,
		TestingT:  t,
		Randomize: 0,
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
