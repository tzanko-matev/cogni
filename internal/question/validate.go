package question

import (
	"fmt"
	"strings"
)

// Issue captures a validation problem in a question specification.
type Issue struct {
	Field   string
	Message string
}

// ValidationError reports one or more validation issues.
type ValidationError struct {
	Issues []Issue
}

// Error returns a readable message for validation failures.
func (err *ValidationError) Error() string {
	if err == nil || len(err.Issues) == 0 {
		return ""
	}
	parts := make([]string, 0, len(err.Issues))
	for _, issue := range err.Issues {
		parts = append(parts, fmt.Sprintf("%s: %s", issue.Field, issue.Message))
	}
	return fmt.Sprintf("question spec validation failed: %s", strings.Join(parts, "; "))
}

type issueCollector struct {
	issues []Issue
}

func (collector *issueCollector) add(field, message string) {
	collector.issues = append(collector.issues, Issue{Field: field, Message: message})
}

func (collector *issueCollector) result() error {
	if len(collector.issues) == 0 {
		return nil
	}
	return &ValidationError{Issues: collector.issues}
}

// NormalizeSpec trims whitespace and validates a question spec.
func NormalizeSpec(spec Spec) (Spec, error) {
	collector := &issueCollector{}
	if spec.Version == 0 {
		collector.add("version", "is required")
	} else if spec.Version != 1 {
		collector.add("version", fmt.Sprintf("unsupported version %d", spec.Version))
	}
	if len(spec.Questions) == 0 {
		collector.add("questions", "must include at least one entry")
	}

	seenIDs := map[string]struct{}{}
	for i, question := range spec.Questions {
		prefix := fmt.Sprintf("questions[%d]", i)
		question.ID = strings.TrimSpace(question.ID)
		if question.ID != "" {
			if _, exists := seenIDs[question.ID]; exists {
				collector.add(prefix+".id", fmt.Sprintf("duplicate id %q", question.ID))
			} else {
				seenIDs[question.ID] = struct{}{}
			}
		}

		question.Prompt = strings.TrimSpace(question.Prompt)
		if question.Prompt == "" {
			collector.add(prefix+".question", "is required")
		}

		question.Answers = normalizeStringSlice(question.Answers)
		if len(question.Answers) == 0 {
			collector.add(prefix+".answers", "must include at least one entry")
		} else {
			for answerIndex, answer := range question.Answers {
				if answer == "" {
					collector.add(fmt.Sprintf("%s.answers[%d]", prefix, answerIndex), "is required")
				}
			}
		}

		question.CorrectAnswers = normalizeStringSlice(question.CorrectAnswers)
		if len(question.CorrectAnswers) == 0 {
			collector.add(prefix+".correct_answers", "must include at least one entry")
		} else {
			answerSet := map[string]struct{}{}
			for _, answer := range question.Answers {
				if answer == "" {
					continue
				}
				answerSet[NormalizeAnswerText(answer)] = struct{}{}
			}
			for correctIndex, correct := range question.CorrectAnswers {
				if correct == "" {
					collector.add(fmt.Sprintf("%s.correct_answers[%d]", prefix, correctIndex), "is required")
					continue
				}
				if _, ok := answerSet[NormalizeAnswerText(correct)]; !ok {
					collector.add(fmt.Sprintf("%s.correct_answers[%d]", prefix, correctIndex), fmt.Sprintf("unknown answer %q", correct))
				}
			}
		}
		spec.Questions[i] = question
	}

	if err := collector.result(); err != nil {
		return Spec{}, err
	}
	return spec, nil
}

func normalizeStringSlice(values []string) []string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		normalized = append(normalized, strings.TrimSpace(value))
	}
	return normalized
}
