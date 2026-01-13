package ratelimiter

import "time"

// schedulerState owns queue state for the scheduler loop.
type schedulerState struct {
	queues  map[string]*workQueue
	order   []string
	rrIndex int
}

// workQueue holds ready and blocked jobs for a provider/model pair.
type workQueue struct {
	key     string
	ready   []Job
	blocked blockedQueue
}

// blockedQueue maintains blocked jobs ordered by not-before time.
type blockedQueue struct {
	items []blockedItem
}

// blockedItem stores a job that cannot run until notBefore.
type blockedItem struct {
	job       Job
	notBefore time.Time
}

// newSchedulerState initializes queue state for a Scheduler.
func newSchedulerState() *schedulerState {
	return &schedulerState{
		queues: map[string]*workQueue{},
		order:  []string{},
	}
}

// enqueueReady adds a job to the ready list for its queue.
func (s *schedulerState) enqueueReady(job Job) {
	q := s.queue(queueKey(job))
	q.ready = append(q.ready, job)
}

// enqueueBlocked adds a job to the blocked list for its queue.
func (s *schedulerState) enqueueBlocked(job Job, notBefore time.Time) {
	q := s.queue(queueKey(job))
	q.blocked.push(blockedItem{job: job, notBefore: notBefore})
}

// promoteReady moves any blocked jobs that are ready into their queues.
func (s *schedulerState) promoteReady(now time.Time) {
	for _, q := range s.queues {
		q.promoteReady(now)
	}
}

// nextReady returns the next job using round-robin across queues.
func (s *schedulerState) nextReady() (Job, bool) {
	if len(s.order) == 0 {
		return Job{}, false
	}
	start := s.rrIndex
	for i := 0; i < len(s.order); i++ {
		idx := (start + i) % len(s.order)
		key := s.order[idx]
		q := s.queues[key]
		if q == nil || len(q.ready) == 0 {
			continue
		}
		job := q.ready[0]
		q.ready = q.ready[1:]
		s.rrIndex = (idx + 1) % len(s.order)
		return job, true
	}
	return Job{}, false
}

// nextBlockedTime returns the earliest blocked job time.
func (s *schedulerState) nextBlockedTime() (time.Time, bool) {
	var earliest time.Time
	ok := false
	for _, q := range s.queues {
		if next, has := q.blocked.peekTime(); has {
			if !ok || next.Before(earliest) {
				earliest = next
				ok = true
			}
		}
	}
	return earliest, ok
}

// queue returns the workQueue for a key, creating it on demand.
func (s *schedulerState) queue(key string) *workQueue {
	if q, ok := s.queues[key]; ok {
		return q
	}
	q := &workQueue{key: key}
	s.queues[key] = q
	s.order = append(s.order, key)
	return q
}

// promoteReady moves ready blocked jobs to the queue's ready list.
func (q *workQueue) promoteReady(now time.Time) {
	for {
		item, ok := q.blocked.popReady(now)
		if !ok {
			return
		}
		q.ready = append(q.ready, item.job)
	}
}

// push inserts a blocked item in time order.
func (b *blockedQueue) push(item blockedItem) {
	idx := b.searchIndex(item.notBefore)
	b.items = append(b.items, blockedItem{})
	copy(b.items[idx+1:], b.items[idx:])
	b.items[idx] = item
}

// popReady removes the earliest item if it is ready.
func (b *blockedQueue) popReady(now time.Time) (blockedItem, bool) {
	if len(b.items) == 0 || b.items[0].notBefore.After(now) {
		return blockedItem{}, false
	}
	item := b.items[0]
	b.items = b.items[1:]
	return item, true
}

// peekTime returns the earliest not-before time without removing it.
func (b *blockedQueue) peekTime() (time.Time, bool) {
	if len(b.items) == 0 {
		return time.Time{}, false
	}
	return b.items[0].notBefore, true
}

// searchIndex finds the insertion index for the provided time.
func (b *blockedQueue) searchIndex(target time.Time) int {
	for i, item := range b.items {
		if !item.notBefore.Before(target) {
			return i
		}
	}
	return len(b.items)
}

// queueKey groups jobs by provider and model.
func queueKey(job Job) string {
	return job.Provider + ":" + job.Model
}
