package runner

// summarize aggregates run results into a summary.
func summarize(tasks []TaskResult) RunSummary {
	summary := RunSummary{
		TasksTotal: len(tasks),
	}
	for _, task := range tasks {
		switch task.Status {
		case "pass":
			summary.TasksPassed++
		case "fail":
			summary.TasksFailed++
		}
		if task.QuestionEval != nil {
			for _, questionResult := range task.QuestionEval.Questions {
				summary.TokensTotal += questionResult.TokensTotal
			}
			summary.QuestionsTotal += task.QuestionEval.Summary.QuestionsTotal
			summary.QuestionsCorrect += task.QuestionEval.Summary.QuestionsCorrect
			summary.QuestionsIncorrect += task.QuestionEval.Summary.QuestionsIncorrect
		}
	}
	if summary.TasksTotal > 0 {
		summary.PassRate = float64(summary.TasksPassed) / float64(summary.TasksTotal)
	}
	if summary.QuestionsTotal > 0 {
		summary.QuestionAccuracy = float64(summary.QuestionsCorrect) / float64(summary.QuestionsTotal)
	}
	return summary
}

// limitOrDefault returns the limit when set, otherwise the fallback.
func limitOrDefault(limit, fallback int) int {
	if limit > 0 {
		return limit
	}
	if fallback > 0 {
		return fallback
	}
	return 0
}
