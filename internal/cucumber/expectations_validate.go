package cucumber

import (
	"fmt"
	"strings"
)

// ValidateExpectations ensures expectations cover the provided examples.
func ValidateExpectations(expectations map[string]Expectation, examples []Example) error {
	byID := make(map[string]Example, len(examples))
	for _, example := range examples {
		byID[example.ID] = example
	}
	missing := make([]string, 0)
	for _, example := range examples {
		if _, ok := expectations[example.ID]; !ok {
			missing = append(missing, example.ID)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing expectations for examples: %s", strings.Join(missing, ", "))
	}
	for id := range expectations {
		if _, ok := byID[id]; !ok {
			return fmt.Errorf("expectation references unknown example %q", id)
		}
	}
	return nil
}
