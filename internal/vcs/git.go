package vcs

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Repo represents a git repository location.
type Repo struct {
	Root   string
	runner gitRunner
}

// Metadata captures repository identity and dirty state.
type Metadata struct {
	Name   string
	VCS    string
	Commit string
	Branch string
	Dirty  bool
}

// gitRunner executes git commands for repository metadata.
type gitRunner interface {
	Run(ctx context.Context, dir string, args ...string) (string, error)
}

// execGitRunner invokes git via the system binary.
type execGitRunner struct{}

// Run executes a git command and returns trimmed stdout.
func (execGitRunner) Run(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = "no stderr"
		}
		return "", fmt.Errorf("git %s: %w (%s)", strings.Join(args, " "), err, msg)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// Client coordinates git operations and allows dependency injection.
type Client struct {
	runner gitRunner
}

// NewClient constructs a git client with an optional runner override.
func NewClient(runner gitRunner) Client {
	if runner == nil {
		runner = execGitRunner{}
	}
	return Client{runner: runner}
}

var defaultClient = NewClient(nil)

// DiscoverRepoRoot resolves the git root for a starting directory.
func DiscoverRepoRoot(ctx context.Context, startDir string) (string, error) {
	return defaultClient.DiscoverRepoRoot(ctx, startDir)
}

// Discover returns a Repo rooted at the discovered git root.
func Discover(ctx context.Context, startDir string) (Repo, error) {
	return defaultClient.Discover(ctx, startDir)
}

// DiscoverRepoRoot resolves the git root for a starting directory.
func (c Client) DiscoverRepoRoot(ctx context.Context, startDir string) (string, error) {
	dir := strings.TrimSpace(startDir)
	if dir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("get working directory: %w", err)
		}
		dir = wd
	}
	root, err := c.runner.Run(ctx, dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("discover git root: %w", err)
	}
	return root, nil
}

// Discover returns a Repo rooted at the discovered git root.
func (c Client) Discover(ctx context.Context, startDir string) (Repo, error) {
	root, err := c.DiscoverRepoRoot(ctx, startDir)
	if err != nil {
		return Repo{}, err
	}
	return Repo{Root: root, runner: c.runner}, nil
}

// Metadata reads git metadata for the repository.
func (r Repo) Metadata(ctx context.Context) (Metadata, error) {
	if strings.TrimSpace(r.Root) == "" {
		return Metadata{}, fmt.Errorf("repo root is empty")
	}
	runner := r.runner
	if runner == nil {
		runner = execGitRunner{}
	}
	commit, err := runner.Run(ctx, r.Root, "rev-parse", "HEAD")
	if err != nil {
		return Metadata{}, fmt.Errorf("resolve HEAD: %w", err)
	}
	branch, err := runner.Run(ctx, r.Root, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return Metadata{}, fmt.Errorf("resolve branch: %w", err)
	}
	status, err := runner.Run(ctx, r.Root, "status", "--porcelain")
	if err != nil {
		return Metadata{}, fmt.Errorf("check dirty state: %w", err)
	}
	return Metadata{
		Name:   filepath.Base(r.Root),
		VCS:    "git",
		Commit: commit,
		Branch: branch,
		Dirty:  strings.TrimSpace(status) != "",
	}, nil
}
