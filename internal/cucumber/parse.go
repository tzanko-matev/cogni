package cucumber

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	gherkin "github.com/cucumber/gherkin-go/v19"
	"github.com/cucumber/messages-go/v16"
)

// Example identifies a single scenario example in a feature file.
type Example struct {
	ID           string
	FeaturePath  string
	ScenarioName string
	ScenarioLine int
	ExampleLine  int
	RowIndex     int
	RowID        string
	TagID        string
}

// ParseFeatureFile parses a feature file into example identifiers.
func ParseFeatureFile(path string) ([]Example, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("read feature: %w", err)
	}
	defer file.Close()

	doc, err := gherkin.ParseGherkinDocument(file, (&messages.Incrementing{}).NewId)
	if err != nil {
		return nil, fmt.Errorf("parse feature: %w", err)
	}
	if doc.Feature == nil {
		return nil, fmt.Errorf("missing feature in %s", path)
	}

	scenarios := collectScenarios(doc.Feature)
	examples := make([]Example, 0)
	for _, scenario := range scenarios {
		tagID := scenarioTagID(scenario.Tags)
		scenarioLine := lineFromLocation(scenario.Location)
		if len(scenario.Examples) == 0 {
			rowIndex := 1
			exampleID := buildExampleID(path, scenario.Name, tagID, "", rowIndex, scenarioLine, scenarioLine)
			examples = append(examples, Example{
				ID:           exampleID,
				FeaturePath:  path,
				ScenarioName: scenario.Name,
				ScenarioLine: scenarioLine,
				ExampleLine:  scenarioLine,
				RowIndex:     rowIndex,
				RowID:        "",
				TagID:        tagID,
			})
			continue
		}
		for _, exampleSet := range scenario.Examples {
			idIndex := idColumnIndex(exampleSet.TableHeader)
			for rowIndex, row := range exampleSet.TableBody {
				rowID := ""
				if idIndex >= 0 && idIndex < len(row.Cells) {
					rowID = strings.TrimSpace(row.Cells[idIndex].Value)
				}
				exampleLine := lineFromLocation(row.Location)
				exampleID := buildExampleID(path, scenario.Name, tagID, rowID, rowIndex+1, exampleLine, scenarioLine)
				examples = append(examples, Example{
					ID:           exampleID,
					FeaturePath:  path,
					ScenarioName: scenario.Name,
					ScenarioLine: scenarioLine,
					ExampleLine:  exampleLine,
					RowIndex:     rowIndex + 1,
					RowID:        rowID,
					TagID:        tagID,
				})
			}
		}
	}
	return examples, nil
}

// collectScenarios flattens scenarios from a feature and its rules.
func collectScenarios(feature *messages.Feature) []*messages.Scenario {
	if feature == nil {
		return nil
	}
	scenarios := make([]*messages.Scenario, 0)
	for _, child := range feature.Children {
		if child == nil {
			continue
		}
		if child.Scenario != nil {
			scenarios = append(scenarios, child.Scenario)
		}
		if child.Rule != nil {
			for _, ruleChild := range child.Rule.Children {
				if ruleChild == nil {
					continue
				}
				if ruleChild.Scenario != nil {
					scenarios = append(scenarios, ruleChild.Scenario)
				}
			}
		}
	}
	return scenarios
}

// scenarioTagID extracts a tagged example id from tags.
func scenarioTagID(tags []*messages.Tag) string {
	for _, tag := range tags {
		if tag == nil {
			continue
		}
		name := strings.TrimSpace(tag.Name)
		if strings.HasPrefix(name, "@id:") {
			return strings.TrimSpace(strings.TrimPrefix(name, "@id:"))
		}
	}
	return ""
}

// idColumnIndex locates an "id" column in an examples table header.
func idColumnIndex(header *messages.TableRow) int {
	if header == nil {
		return -1
	}
	for i, cell := range header.Cells {
		if cell == nil {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(cell.Value), "id") {
			return i
		}
	}
	return -1
}

// buildExampleID constructs a stable example identifier.
func buildExampleID(featurePath, scenarioName, tagID, rowID string, rowIndex, exampleLine, scenarioLine int) string {
	rowIndex = normalizeRowIndex(rowIndex)
	rowID = strings.TrimSpace(rowID)
	if tagID != "" && rowID != "" {
		return fmt.Sprintf("%s:%s", tagID, rowID)
	}
	if tagID != "" {
		return fmt.Sprintf("%s:%d", tagID, rowIndex)
	}
	if rowID != "" {
		return fmt.Sprintf("%s#%s", scenarioName, rowID)
	}
	line := scenarioLine
	if exampleLine > 0 {
		line = exampleLine
	}
	return fmt.Sprintf("%s:%d:%d", filepath.ToSlash(featurePath), line, rowIndex)
}

// normalizeRowIndex clamps row indexes to be at least 1.
func normalizeRowIndex(index int) int {
	if index < 1 {
		return 1
	}
	return index
}

// lineFromLocation extracts the line number from a Gherkin location.
func lineFromLocation(location *messages.Location) int {
	if location == nil {
		return 0
	}
	return int(location.Line)
}
