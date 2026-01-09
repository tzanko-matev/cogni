package question

// Spec defines the question specification schema loaded from JSON or YAML.
type Spec struct {
	Version   int        `json:"version" yaml:"version"`
	Questions []Question `json:"questions" yaml:"questions"`
}

// Question represents a single question with answer choices and correct answers.
type Question struct {
	ID             string   `json:"id" yaml:"id"`
	Prompt         string   `json:"question" yaml:"question"`
	Answers        []string `json:"answers" yaml:"answers"`
	CorrectAnswers []string `json:"correct_answers" yaml:"correct_answers"`
}
