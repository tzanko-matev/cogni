package cli

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// readLine reads a line from the reader, trimming line endings.
func readLine(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			return strings.TrimRight(line, "\r\n"), io.EOF
		}
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

// promptString asks for a string value with an optional default.
func promptString(reader *bufio.Reader, out io.Writer, label, defaultValue string) (string, error) {
	for {
		if defaultValue != "" {
			fmt.Fprintf(out, "%s [%s]: ", label, defaultValue)
		} else {
			fmt.Fprintf(out, "%s: ", label)
		}
		line, err := readLine(reader)
		if err != nil && err != io.EOF {
			return "", err
		}
		line = strings.TrimSpace(line)
		if line == "" && defaultValue != "" {
			return defaultValue, nil
		}
		if line != "" {
			return line, nil
		}
		if err == io.EOF {
			return "", fmt.Errorf("missing input for %s", label)
		}
	}
}

// promptYesNo prompts for a yes/no response with a default.
func promptYesNo(reader *bufio.Reader, out io.Writer, label string, defaultYes bool) (bool, error) {
	suffix := "y/N"
	if defaultYes {
		suffix = "Y/n"
	}
	for {
		fmt.Fprintf(out, "%s [%s]: ", label, suffix)
		line, err := readLine(reader)
		if err != nil && err != io.EOF {
			return false, err
		}
		line = strings.TrimSpace(strings.ToLower(line))
		if line == "" {
			if err == io.EOF {
				return defaultYes, nil
			}
			return defaultYes, nil
		}
		switch line {
		case "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		default:
			if err == io.EOF {
				return false, fmt.Errorf("invalid response %q", line)
			}
			fmt.Fprintln(out, "Please answer yes or no.")
		}
	}
}
