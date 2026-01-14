package runner

import (
	"strings"
	"sync"
	"time"

	"cogni/internal/question"
	"cogni/pkg/ratelimiter"
)

// questionEventOptions carries optional metadata for a question event.
type questionEventOptions struct {
	EventType    QuestionEventType
	RetryAfterMs int
	ToolName     string
	ToolDuration time.Duration
	ToolError    string
	Tokens       int
	WallTime     time.Duration
	Error        string
	EmittedAt    time.Time
}

// questionJobObserver bridges scheduler/job events to RunObserver callbacks.
type questionJobObserver struct {
	observer  RunObserver
	taskID    string
	questions []question.Question
	mu        sync.RWMutex
	jobIndex  map[string]int
}

// newQuestionJobObserver constructs a job observer when a RunObserver is set.
func newQuestionJobObserver(observer RunObserver, taskID string, questions []question.Question) *questionJobObserver {
	if observer == nil {
		return nil
	}
	return &questionJobObserver{
		observer:  observer,
		taskID:    taskID,
		questions: questions,
		jobIndex:  map[string]int{},
	}
}

// EmitQueuedAll emits queued events for every question in the task.
func (o *questionJobObserver) EmitQueuedAll() {
	if o == nil {
		return
	}
	for index := range o.questions {
		o.Emit(index, questionEventOptions{EventType: QuestionQueued})
	}
}

// RegisterJob associates a scheduler job id with a question index.
func (o *questionJobObserver) RegisterJob(jobID string, index int) {
	if o == nil {
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	o.jobIndex[jobID] = index
}

// Emit emits an observer event for the given question index.
func (o *questionJobObserver) Emit(index int, opts questionEventOptions) {
	if o == nil || o.observer == nil {
		return
	}
	if index < 0 || index >= len(o.questions) {
		return
	}
	item := o.questions[index]
	emittedAt := opts.EmittedAt
	if emittedAt.IsZero() {
		emittedAt = time.Now()
	}
	o.observer.OnQuestionEvent(QuestionEvent{
		TaskID:        o.taskID,
		QuestionIndex: index,
		QuestionID:    item.ID,
		QuestionText:  item.Prompt,
		Type:          opts.EventType,
		RetryAfterMs:  opts.RetryAfterMs,
		ToolName:      opts.ToolName,
		ToolDuration:  opts.ToolDuration,
		ToolError:     opts.ToolError,
		Tokens:        opts.Tokens,
		WallTime:      opts.WallTime,
		Error:         opts.Error,
		EmittedAt:     emittedAt,
	})
}

// OnReserveStart reports reserve attempts from the scheduler.
func (o *questionJobObserver) OnReserveStart(job ratelimiter.Job) {
	o.emitByJob(job.JobID, questionEventOptions{EventType: QuestionReserving})
}

// OnReserveDenied reports reserve denials from the scheduler.
func (o *questionJobObserver) OnReserveDenied(job ratelimiter.Job, res ratelimiter.ReserveResponse) {
	eventType := QuestionWaitingRateLimit
	if strings.HasPrefix(res.Error, "limit_decreasing") {
		eventType = QuestionWaitingLimitDecreasing
	}
	o.emitByJob(job.JobID, questionEventOptions{
		EventType:    eventType,
		RetryAfterMs: res.RetryAfterMs,
		Error:        res.Error,
	})
}

// OnReserveError reports reserve errors from the scheduler.
func (o *questionJobObserver) OnReserveError(job ratelimiter.Job, err error) {
	if err == nil {
		return
	}
	o.emitByJob(job.JobID, questionEventOptions{
		EventType: QuestionWaitingLimiterError,
		Error:     err.Error(),
	})
}

// emitByJob resolves a job id to its question index and emits an event.
func (o *questionJobObserver) emitByJob(jobID string, opts questionEventOptions) {
	if o == nil {
		return
	}
	o.mu.RLock()
	index, ok := o.jobIndex[jobID]
	o.mu.RUnlock()
	if !ok {
		return
	}
	o.Emit(index, opts)
}
