package runner

// QuestionEval contains per-question evaluation results.
type QuestionEval struct {
	QuestionsFile string           `json:"questions_file"`
	Questions     []QuestionResult `json:"questions"`
	Summary       QuestionSummary  `json:"summary"`
}

// QuestionResult records evaluation results for a single question.
type QuestionResult struct {
	ID                string         `json:"id,omitempty"`
	Question          string         `json:"question"`
	Answers           []string       `json:"answers,omitempty"`
	CorrectAnswers    []string       `json:"correct_answers,omitempty"`
	AgentAnswer       string         `json:"agent_answer,omitempty"`
	Correct           bool           `json:"correct"`
	ParseError        string         `json:"parse_error,omitempty"`
	RunError          string         `json:"run_error,omitempty"`
	TokensTotal       int            `json:"tokens_total,omitempty"`
	WallTimeSeconds   float64        `json:"wall_time_seconds,omitempty"`
	AgentSteps        int            `json:"agent_steps,omitempty"`
	ToolCalls         map[string]int `json:"tool_calls,omitempty"`
	Compactions       int            `json:"compactions,omitempty"`
	LastSummaryTokens int            `json:"last_summary_tokens,omitempty"`
}

// QuestionSummary aggregates accuracy metrics for a question evaluation.
type QuestionSummary struct {
	QuestionsTotal     int     `json:"questions_total"`
	QuestionsCorrect   int     `json:"questions_correct"`
	QuestionsIncorrect int     `json:"questions_incorrect"`
	Accuracy           float64 `json:"accuracy"`
}
