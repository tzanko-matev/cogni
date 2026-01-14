package live

import (
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model renders a live console UI using Bubble Tea.
type Model struct {
	state        State
	table        table.Model
	events       <-chan Event
	tickInterval time.Duration
	now          time.Time
	noColor      bool
}

// Options configures the live UI model.
type Options struct {
	NoColor      bool
	TickInterval time.Duration
}

// NewModel constructs a live UI model for an event stream.
func NewModel(events <-chan Event, opts Options) Model {
	tickInterval := opts.TickInterval
	if tickInterval <= 0 {
		tickInterval = 200 * time.Millisecond
	}
	t := table.New(
		table.WithColumns(defaultColumns()),
		table.WithRows([]table.Row{}),
		table.WithFocused(false),
	)
	t.SetStyles(tableStyles(opts.NoColor))
	return Model{
		state:        State{},
		table:        t,
		events:       events,
		tickInterval: tickInterval,
		now:          time.Now(),
		noColor:      opts.NoColor,
	}
}

// Init starts ticking and waits for the first event.
func (m Model) Init() tea.Cmd {
	return tea.Batch(waitForEvent(m.events), tick(m.tickInterval))
}

// Update consumes UI events and timer ticks.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		m.table.SetWidth(typed.Width)
		m.table.SetHeight(max(typed.Height-4, 1))
		m.table.SetColumns(columnsForWidth(typed.Width))
		return m, nil
	case EventMsg:
		m = applyEvent(m, typed.Event)
		return m, waitForEvent(m.events)
	case tickMsg:
		m.now = time.Time(typed)
		return m, tick(m.tickInterval)
	}
	return m, nil
}

// View renders the live UI.
func (m Model) View() string {
	header := renderHeader(m.state, m.now, m.noColor)
	summary := renderSummary(m.state, m.noColor)
	taskLine := renderTaskLine(m.state, m.noColor)
	tableView := m.table.View()
	footer := renderFooter(m.state, m.noColor)
	return lipgloss.JoinVertical(lipgloss.Left, header, summary, taskLine, tableView, footer)
}

// EventMsg wraps a UI event for Bubble Tea.
type EventMsg struct {
	Event Event
}

// tickMsg carries a clock tick for updates.
type tickMsg time.Time

// waitForEvent blocks until a UI event is available.
func waitForEvent(events <-chan Event) tea.Cmd {
	return func() tea.Msg {
		if events == nil {
			return nil
		}
		event, ok := <-events
		if !ok {
			return tea.Quit()
		}
		return EventMsg{Event: event}
	}
}

// tick emits a periodic tick message.
func tick(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// applyEvent mutates model state based on a UI event.
func applyEvent(model Model, event Event) Model {
	switch event.Kind {
	case EventRunStart:
		model.state.RunID = event.RunID
		model.state.Repo = event.Repo
		if model.state.StartedAt.IsZero() {
			model.state.StartedAt = time.Now()
		}
	case EventTaskStart:
		model.state.TaskID = event.TaskID
		model.state.QuestionsFile = event.QuestionsFile
		model.state.AgentID = event.AgentID
		model.state.Model = event.Model
		model.state.Rows = nil
		model.state.Counts = StatusCounts{}
		model.state.LastEvent = ""
	case EventQuestion:
		model.state = Reduce(model.state, event.Question)
	case EventTaskEnd:
		model.state.LastEvent = formatTaskEnd(event.TaskID, event.TaskStatus, event.TaskReason)
	case EventRunEnd:
		return model
	}
	model.table.SetRows(rowsForState(model.state, model.now, model.noColor))
	return model
}
