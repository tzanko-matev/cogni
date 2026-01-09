package runner

import (
	"strings"

	"cogni/internal/question"
)

// buildQuestionPrompt constructs the prompt for a single question evaluation.
func buildQuestionPrompt(item question.Question) string {
	var builder strings.Builder
	builder.WriteString("Answer the question about the repository.\n")
	builder.WriteString("You may include reasoning, but the final output must end with:\n")
	builder.WriteString("<answer>...</answer>\n")
	builder.WriteString("Do not add any text after </answer>.\n\n")
	builder.WriteString("Question:\n")
	builder.WriteString(item.Prompt)
	builder.WriteString("\n\nAnswer choices:\n")
	for _, answer := range item.Answers {
		builder.WriteString("- ")
		builder.WriteString(answer)
		builder.WriteString("\n")
	}
	return builder.String()
}
