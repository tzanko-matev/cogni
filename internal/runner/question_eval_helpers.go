package runner

import (
	"path/filepath"
	"strings"

	"cogni/internal/question"
)

// resolveQuestionsFile resolves the questions file path against the repo root.
func resolveQuestionsFile(repoRoot, questionsFile string) string {
	if strings.TrimSpace(questionsFile) == "" || filepath.IsAbs(questionsFile) {
		return questionsFile
	}
	return filepath.Join(repoRoot, questionsFile)
}

// isCorrectAnswer checks whether a normalized answer matches any correct answer.
func isCorrectAnswer(normalized string, correctAnswers []string) bool {
	for _, candidate := range correctAnswers {
		if question.NormalizeAnswerText(candidate) == normalized {
			return true
		}
	}
	return false
}
