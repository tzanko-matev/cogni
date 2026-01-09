package runner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"cogni/internal/agent"
	"cogni/internal/spec"
	"cogni/internal/testutil"
	"cogni/internal/tools"
	"cogni/internal/vcs"
)

// sequenceProvider replays responses in order across Stream calls.
type sequenceProvider struct {
	responses []string
	index     *int
}

// Stream returns the next response in the sequence.
func (p sequenceProvider) Stream(_ context.Context, _ agent.Prompt) (agent.Stream, error) {
	if *p.index >= len(p.responses) {
		return &fakeStream{events: []agent.StreamEvent{}}, nil
	}
	message := p.responses[*p.index]
	*p.index++
	return &fakeStream{events: []agent.StreamEvent{{Type: agent.StreamEventMessage, Message: message}}}, nil
}

// TestRunQuestionEvalTask verifies question_eval tasks are executed and summarized.
func TestRunQuestionEvalTask(t *testing.T) {
	repoRoot := t.TempDir()
	specPath := filepath.Join(repoRoot, "questions.yml")
	specBody := `version: 1
questions:
  - id: q1
    question: "What is 2+2?"
    answers: ["4", "5"]
    correct_answers: ["4"]
  - id: q2
    question: "Pick a color"
    answers: ["blue", "green"]
    correct_answers: ["blue"]
`
	if err := os.WriteFile(specPath, []byte(specBody), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}

	cfg := spec.Config{
		Repo: spec.RepoConfig{OutputDir: "./out"},
		Agents: []spec.AgentConfig{
			{ID: "agent-1", Type: "builtin", Provider: "openrouter", Model: "model"},
		},
		DefaultAgent: "agent-1",
		Tasks: []spec.TaskConfig{
			{ID: "task-1", Type: "question_eval", Agent: "agent-1", QuestionsFile: "questions.yml"},
		},
	}

	index := 0
	responses := []string{
		"Reasoning.\n<answer>4</answer>",
		"Reasoning.\n<answer>green</answer>",
	}

	ctx := testutil.Context(t, 0)
	results, err := Run(ctx, cfg, RunParams{
		RepoRoot: repoRoot,
		Deps: RunDependencies{
			ProviderFactory: func(_ spec.AgentConfig, _ string) (agent.Provider, error) {
				return sequenceProvider{responses: responses, index: &index}, nil
			},
			ToolRunnerFactory: func(root string) (*tools.Runner, error) {
				return tools.NewRunner(root)
			},
			RepoRootResolver: func(_ context.Context, root string) (string, error) {
				return root, nil
			},
			RepoMetadataLoader: func(_ context.Context, root string) (vcs.Metadata, error) {
				return vcs.Metadata{Name: filepath.Base(root), VCS: "git", Commit: "commit", Branch: "main", Dirty: false}, nil
			},
			RunID: func() (string, error) { return "run-1", nil },
			Now:   func() time.Time { return time.Now() },
		},
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(results.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(results.Tasks))
	}
	task := results.Tasks[0]
	if task.QuestionEval == nil {
		t.Fatalf("expected question eval results")
	}
	if task.QuestionEval.Summary.QuestionsTotal != 2 {
		t.Fatalf("unexpected question total: %+v", task.QuestionEval.Summary)
	}
	if task.QuestionEval.Summary.QuestionsCorrect != 1 {
		t.Fatalf("unexpected correct count: %+v", task.QuestionEval.Summary)
	}
	if task.Status != "fail" {
		t.Fatalf("expected fail status, got %+v", task.Status)
	}
	if results.Summary.QuestionsTotal != 2 || results.Summary.QuestionsCorrect != 1 {
		t.Fatalf("unexpected run summary: %+v", results.Summary)
	}
	if results.Summary.TokensTotal == 0 {
		t.Fatalf("expected token count to be recorded")
	}
}
