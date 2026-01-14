package live

import (
	"io"
	"os"
	"sync"

	tea "github.com/charmbracelet/bubbletea"

	"cogni/internal/runner"
)

// Controller runs the live UI and implements runner.RunObserver.
type Controller struct {
	events    chan Event
	program   *tea.Program
	done      chan struct{}
	closeOnce sync.Once
}

// Start launches a live UI controller that writes to stdout.
func Start(stdout io.Writer, opts Options) *Controller {
	if stdout == nil {
		stdout = os.Stdout
	}
	events := make(chan Event, 256)
	model := NewModel(events, opts)
	program := tea.NewProgram(model, tea.WithOutput(stdout), tea.WithAltScreen())
	controller := &Controller{
		events:  events,
		program: program,
		done:    make(chan struct{}),
	}
	go func() {
		_ = program.Start()
		close(controller.done)
	}()
	return controller
}

// Close signals the UI to stop.
func (c *Controller) Close() {
	if c == nil {
		return
	}
	c.closeOnce.Do(func() {
		close(c.events)
	})
}

// Wait blocks until the UI has exited.
func (c *Controller) Wait() {
	if c == nil {
		return
	}
	<-c.done
}

// OnRunStart forwards run start events to the UI.
func (c *Controller) OnRunStart(runID string, repo string) {
	c.send(Event{Kind: EventRunStart, RunID: runID, Repo: repo})
}

// OnTaskStart forwards task start events to the UI.
func (c *Controller) OnTaskStart(taskID string, taskType string, questionsFile string, agentID string, model string) {
	c.send(Event{
		Kind:          EventTaskStart,
		TaskID:        taskID,
		QuestionsFile: questionsFile,
		AgentID:       agentID,
		Model:         model,
	})
}

// OnQuestionEvent forwards question status updates to the UI.
func (c *Controller) OnQuestionEvent(event runner.QuestionEvent) {
	c.send(Event{Kind: EventQuestion, Question: event})
}

// OnTaskEnd forwards task completion events to the UI.
func (c *Controller) OnTaskEnd(taskID string, status string, reason *string) {
	c.send(Event{Kind: EventTaskEnd, TaskID: taskID, TaskStatus: status, TaskReason: reason})
}

// OnRunEnd forwards run completion events to the UI and closes it.
func (c *Controller) OnRunEnd(results runner.Results) {
	c.send(Event{Kind: EventRunEnd})
	c.Close()
}

// send enqueues an event without blocking the caller.
func (c *Controller) send(event Event) {
	if c == nil {
		return
	}
	select {
	case c.events <- event:
	default:
	}
}
