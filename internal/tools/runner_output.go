package tools

import (
	"fmt"
	"time"
)

// finalize assembles a CallResult with timing and truncation metadata.
func (r *Runner) finalize(tool string, start, end time.Time, output string, truncated bool, err error) CallResult {
	if err != nil {
		output = fmt.Sprintf("error: %s", err.Error())
	}
	output, limited := applyOutputLimit(output, r.Limits.MaxOutputBytes, truncated)
	return CallResult{
		Tool:        tool,
		Output:      output,
		OutputBytes: len(output),
		Truncated:   limited,
		StartedAt:   start,
		FinishedAt:  end,
		Duration:    end.Sub(start),
		Error:       errorString(err),
	}
}

// applyOutputLimit truncates output to a maximum size.
func applyOutputLimit(output string, max int, truncated bool) (string, bool) {
	if max <= 0 {
		return output, truncated
	}
	if len(output) > max {
		return truncateOutput(output, max)
	}
	if truncated {
		if len(output)+len(truncationMarker) <= max {
			return output + truncationMarker, true
		}
		return truncateOutput(output, max)
	}
	return output, false
}

// truncateOutput trims output and appends a truncation marker.
func truncateOutput(output string, max int) (string, bool) {
	if max <= 0 || len(output) <= max {
		return output, false
	}
	if max <= len(truncationMarker) {
		return truncationMarker[:max], true
	}
	return output[:max-len(truncationMarker)] + truncationMarker, true
}

// errorString formats errors for CallResult output.
func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
