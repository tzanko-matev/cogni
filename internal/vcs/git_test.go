package vcs

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverRepoRootAndMetadata(t *testing.T) {
	requireGit(t)

	ctx := context.Background()
	repo := setupTestRepo(t)

	subdir := filepath.Join(repo.Root, "nested")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	root, err := DiscoverRepoRoot(ctx, subdir)
	if err != nil {
		t.Fatalf("discover repo root: %v", err)
	}
	if root != repo.Root {
		t.Fatalf("expected root %q, got %q", repo.Root, root)
	}

	meta, err := Repo{Root: root}.Metadata(ctx)
	if err != nil {
		t.Fatalf("metadata: %v", err)
	}
	if meta.Commit != repo.Commits[len(repo.Commits)-1] {
		t.Fatalf("expected commit %q, got %q", repo.Commits[len(repo.Commits)-1], meta.Commit)
	}
	if meta.Branch != "main" {
		t.Fatalf("expected branch main, got %q", meta.Branch)
	}
	if meta.Dirty {
		t.Fatalf("expected clean repo, got dirty")
	}

	if err := os.WriteFile(filepath.Join(repo.Root, "untracked.txt"), []byte("dirty"), 0o644); err != nil {
		t.Fatalf("write dirty file: %v", err)
	}
	meta, err = Repo{Root: root}.Metadata(ctx)
	if err != nil {
		t.Fatalf("metadata dirty: %v", err)
	}
	if !meta.Dirty {
		t.Fatalf("expected dirty repo")
	}
}

type testRepo struct {
	Root    string
	Commits []string
}

func setupTestRepo(t *testing.T) testRepo {
	t.Helper()

	root := t.TempDir()
	runGitTest(t, root, "-c", "init.defaultBranch=main", "init")

	readmePath := filepath.Join(root, "README.md")
	if err := os.WriteFile(readmePath, []byte("initial"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	runGitTest(t, root, "add", "README.md")
	runGitTest(t, root, "commit", "-m", "initial")
	commit1 := runGitTest(t, root, "rev-parse", "HEAD")

	if err := os.WriteFile(readmePath, []byte("second"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	runGitTest(t, root, "add", "README.md")
	runGitTest(t, root, "commit", "-m", "second")
	commit2 := runGitTest(t, root, "rev-parse", "HEAD")

	notesPath := filepath.Join(root, "notes.txt")
	if err := os.WriteFile(notesPath, []byte("third"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	runGitTest(t, root, "add", "notes.txt")
	runGitTest(t, root, "commit", "-m", "third")
	commit3 := runGitTest(t, root, "rev-parse", "HEAD")

	return testRepo{
		Root:    root,
		Commits: []string{commit1, commit2, commit3},
	}
}

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
}

func runGitTest(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test User",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test User",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, string(out))
	}
	return strings.TrimSpace(string(out))
}
