package call

import (
	"io"
	"os"
	"strings"
)

// verboseStyle selects how verbose output is styled.
type verboseStyle int

const (
	styleDefault verboseStyle = iota
	styleDim
	styleHeadingPrompt
	styleHeadingOutput
	styleHeadingToolCall
	styleHeadingToolResult
	styleHeadingMetrics
	styleHeadingTask
	styleHeadingError
)

// verbosePalette controls ANSI styling for verbose output.
type verbosePalette struct {
	enabled bool
}

// paletteFor selects a palette based on the writer and color settings.
func paletteFor(writer io.Writer, noColor bool) verbosePalette {
	if noColor {
		return verbosePalette{enabled: false}
	}
	return verbosePalette{enabled: shouldUseStyling(writer)}
}

// shouldUseStyling reports whether ANSI styling should be enabled.
func shouldUseStyling(writer io.Writer) bool {
	if writer == nil {
		return false
	}
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	if strings.EqualFold(os.Getenv("CLICOLOR"), "0") {
		return false
	}
	if file, ok := writer.(*os.File); ok {
		return isTerminal(int(file.Fd()))
	}
	if fder, ok := writer.(interface{ Fd() uintptr }); ok {
		return isTerminal(int(fder.Fd()))
	}
	return false
}

// apply wraps text with ANSI codes for the requested style.
func (p verbosePalette) apply(style verboseStyle, text string) string {
	if !p.enabled {
		return text
	}
	switch style {
	case styleDim:
		return ansiDim + ansiGray + text + ansiReset
	case styleHeadingPrompt:
		return ansiBold + ansiCyan + text + ansiReset
	case styleHeadingOutput:
		return ansiBold + ansiMagenta + text + ansiReset
	case styleHeadingToolCall:
		return ansiBold + ansiYellow + text + ansiReset
	case styleHeadingToolResult:
		return ansiBold + ansiGreen + text + ansiReset
	case styleHeadingMetrics:
		return ansiBold + ansiBlue + text + ansiReset
	case styleHeadingTask:
		return ansiBold + ansiCyan + text + ansiReset
	case styleHeadingError:
		return ansiBold + ansiRed + text + ansiReset
	default:
		return text
	}
}
