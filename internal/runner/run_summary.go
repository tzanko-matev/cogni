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
		for _, attempt := range task.Attempts {
			summary.TokensTotal += attempt.TokensTotal
		}
		if task.QuestionEval != nil {
			for _, questionResult := range task.QuestionEval.Questions {
				summary.TokensTotal += questionResult.TokensTotal
			}
			summary.QuestionsTotal += task.QuestionEval.Summary.QuestionsTotal
			summary.QuestionsCorrect += task.QuestionEval.Summary.QuestionsCorrect
			summary.QuestionsIncorrect += task.QuestionEval.Summary.QuestionsIncorrect
		}
		if task.Cucumber != nil {
			summary.CucumberExamplesTotal += task.Cucumber.Summary.ExamplesTotal
			summary.CucumberExamplesCorrect += task.Cucumber.Summary.ExamplesCorrect
			summary.CucumberExamplesIncorrect += task.Cucumber.Summary.ExamplesIncorrect
		}
	}
	if summary.TasksTotal > 0 {
		summary.PassRate = float64(summary.TasksPassed) / float64(summary.TasksTotal)
	}
	if summary.QuestionsTotal > 0 {
		summary.QuestionAccuracy = float64(summary.QuestionsCorrect) / float64(summary.QuestionsTotal)
	}
	if summary.CucumberExamplesTotal > 0 {
		summary.CucumberAccuracy = float64(summary.CucumberExamplesCorrect) / float64(summary.CucumberExamplesTotal)
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
