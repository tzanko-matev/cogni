package cli

import (
	"fmt"
	"io"
)

const (
	ExitOK    = 0
	ExitError = 1
	ExitUsage = 2
)

type Command struct {
	Name    string
	Summary string
	Usage   []string
	Run     func(args []string, stdout, stderr io.Writer) int
}

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

func findCommand(name string) *Command {
	for _, cmd := range commands {
		if cmd.Name == name {
			return cmd
		}
	}
	return nil
}

func isHelpArg(arg string) bool {
	switch arg {
	case "-h", "--help", "help":
		return true
	default:
		return false
	}
}

func wantsHelp(args []string) bool {
	for _, arg := range args {
		switch arg {
		case "-h", "--help":
			return true
		}
	}
	return false
}

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

func printCommandUsage(cmd *Command, w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	for _, line := range cmd.Usage {
		fmt.Fprintf(w, "  %s\n", line)
	}
	if cmd.Summary != "" {
		fmt.Fprintf(w, "\n%s\n", cmd.Summary)
	}
}

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

var commands = []*Command{
	command("init", "Scaffold .cogni.yml and schemas", []string{
		"cogni init",
	}, nil),
	command("validate", "Validate .cogni.yml and schemas", []string{
		"cogni validate --spec <path>",
	}, runValidate),
	command("run", "Execute benchmark tasks", []string{
		"cogni run [task-id|task-id@agent-id]...",
	}, nil),
	command("compare", "Compare runs between commits", []string{
		"cogni compare --base <commit|run-id|ref> [--head <commit|run-id|ref>]",
		"cogni compare --range <start>..<end>",
	}, nil),
	command("report", "Generate HTML reports", []string{
		"cogni report --range <start>..<end>",
	}, nil),
}
