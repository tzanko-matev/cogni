package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

type testEvent struct {
	Action  string  `json:"Action"`
	Package string  `json:"Package"`
	Test    string  `json:"Test"`
	Elapsed float64 `json:"Elapsed"`
}

type testResult struct {
	Package  string  `json:"package"`
	Name     string  `json:"name"`
	Status   string  `json:"status"`
	Duration float64 `json:"duration"`
}

type output struct {
	GeneratedAt string                `json:"generated_at"`
	Tests       map[string]testResult `json:"tests"`
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	results := make(map[string]testResult)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var event testEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		if event.Test == "" {
			continue
		}
		switch event.Action {
		case "pass", "fail", "skip":
			id := fmt.Sprintf("%s::%s", event.Package, event.Test)
			results[id] = testResult{
				Package:  event.Package,
				Name:     event.Test,
				Status:   event.Action,
				Duration: event.Elapsed,
			}
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "read test events: %v\n", err)
		os.Exit(1)
	}

	payload := output{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Tests:       results,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(payload); err != nil {
		fmt.Fprintf(os.Stderr, "write output: %v\n", err)
		os.Exit(1)
	}
}
