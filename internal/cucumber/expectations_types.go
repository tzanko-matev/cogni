package cucumber

// Expectation records the expected implementation status for an example.
type Expectation struct {
	ExampleID   string
	Implemented bool
	Notes       string
}
