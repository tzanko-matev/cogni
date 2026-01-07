package config

// issueAdder adds a validation issue to a shared collector.
type issueAdder func(field, message string)

// issueCollector accumulates validation issues.
type issueCollector struct {
	issues []Issue
}

// add records a new validation issue.
func (c *issueCollector) add(field, message string) {
	c.issues = append(c.issues, Issue{Field: field, Message: message})
}

// result returns a ValidationError when issues are present.
func (c *issueCollector) result() error {
	if len(c.issues) == 0 {
		return nil
	}
	return &ValidationError{Issues: c.issues}
}
