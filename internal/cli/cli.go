package cli

import (
	"fmt"
	"io"
)

// Exit codes returned by CLI commands.
const (
	ExitOK    = 0
	ExitError = 1
	ExitUsage = 2
)

// Command defines a CLI command and its handler.
type Command struct {
	Name    string
	Summary string
	Usage   []string
	Run     func(args []string, stdout, stderr io.Writer) int
}

// Run dispatches the CLI to the appropriate command.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stdout)
		return ExitUsage
	}
	if isHelpArg(args[0]) {
		printUsage(stdout)
		return ExitOK
	}

	cmd := findCommand(args[0])
	if cmd == nil {
		fmt.Fprintf(stderr, "Unknown command: %s\n\n", args[0])
		printUsage(stderr)
		return ExitUsage
	}

	return cmd.Run(args[1:], stdout, stderr)
}

// findCommand locates a command by name.
func findCommand(name string) *Command {
	for _, cmd := range commands {
		if cmd.Name == name {
			return cmd
		}
	}
	return nil
}

// isHelpArg reports whether an argument triggers global help.
func isHelpArg(arg string) bool {
	switch arg {
	case "-h", "--help", "help":
		return true
	default:
		return false
	}
}

// wantsHelp checks for per-command help flags.
func wantsHelp(args []string) bool {
	for _, arg := range args {
		switch arg {
		case "-h", "--help":
			return true
		}
	}
	return false
}

// printUsage prints the top-level CLI usage.
func printUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  cogni <command> [options]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	for _, cmd := range commands {
		fmt.Fprintf(w, "  %-8s %s\n", cmd.Name, cmd.Summary)
	}
	fmt.Fprintln(w, "\nUse \"cogni <command> --help\" for more information.")
}

// printCommandUsage prints usage for a specific command.
func printCommandUsage(cmd *Command, w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	for _, line := range cmd.Usage {
		fmt.Fprintf(w, "  %s\n", line)
	}
	if cmd.Summary != "" {
		fmt.Fprintf(w, "\n%s\n", cmd.Summary)
	}
}

// runNotImplemented returns a handler for unimplemented commands.
func runNotImplemented(cmd *Command) func(args []string, stdout, stderr io.Writer) int {
	return func(args []string, stdout, stderr io.Writer) int {
		if wantsHelp(args) {
			printCommandUsage(cmd, stdout)
			return ExitOK
		}
		fmt.Fprintf(stderr, "cogni %s is not implemented yet\n", cmd.Name)
		return ExitError
	}
}

// command constructs a Command with a configured runner.
func command(name, summary string, usage []string, runner func(cmd *Command) func(args []string, stdout, stderr io.Writer) int) *Command {
	cmd := &Command{
		Name:    name,
		Summary: summary,
		Usage:   usage,
	}
	if runner == nil {
		cmd.Run = runNotImplemented(cmd)
	} else {
		cmd.Run = runner(cmd)
	}
	return cmd
}

// commands registers all CLI commands.
var commands = []*Command{
	command("init", "Scaffold .cogni config and schemas", []string{
		"cogni init",
		"cogni init --spec <path>",
	}, runInit),
	command("validate", "Validate .cogni config and schemas", []string{
		"cogni validate [--spec <path>]",
	}, runValidate),
	command("run", "Execute benchmark tasks", []string{
		"cogni run [task-id|task-id@agent-id]...",
		"cogni run --verbose [task-id|task-id@agent-id]...",
		"cogni run --verbose --no-color [task-id|task-id@agent-id]...",
	}, runRun),
	command("eval", "Evaluate a question spec", []string{
		"cogni eval <questions_file> --agent <id>",
		"cogni eval <questions_file> --agent <id> --verbose",
		"cogni eval <questions_file> --agent <id> --no-color",
	}, runEval),
	command("compare", "Compare runs between commits", []string{
		"cogni compare --base <commit|run-id|ref> [--head <commit|run-id|ref>]",
		"cogni compare --range <start>..<end>",
	}, runCompare),
	command("report", "Generate HTML reports", []string{
		"cogni report --range <start>..<end>",
	}, runReport),
}
