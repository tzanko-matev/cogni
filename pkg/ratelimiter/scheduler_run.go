package ratelimiter

import "time"

// requeueRequest carries a job and its next eligible time.
type requeueRequest struct {
	job       Job
	notBefore time.Time
}

// run drives the scheduler loop until shutdown.
func (s *Scheduler) run() {
	timer := time.NewTimer(s.idleInterval)
	defer timer.Stop()

	for {
		s.state.promoteReady(s.now())
		s.dispatchReady()
		nextDelay := s.nextWakeDelay()
		resetTimer(timer, nextDelay)

		select {
		case <-s.stopCh:
			close(s.workCh)
			close(s.doneCh)
			return
		case job := <-s.submitCh:
			s.state.enqueueReady(job)
		case msg := <-s.requeueCh:
			s.state.enqueueBlocked(msg.job, msg.notBefore)
		case <-timer.C:
		}
	}
}

// dispatchReady sends available work to workers.
func (s *Scheduler) dispatchReady() {
	for len(s.workCh) < cap(s.workCh) {
		job, ok := s.state.nextReady()
		if !ok {
			return
		}
		s.workCh <- job
	}
}

// requeue schedules a job to be retried later.
func (s *Scheduler) requeue(job Job, notBefore time.Time) {
	msg := requeueRequest{job: job, notBefore: notBefore}
	select {
	case <-s.doneCh:
	case s.requeueCh <- msg:
	}
}

// nextWakeDelay computes the delay until the next blocked job is ready.
func (s *Scheduler) nextWakeDelay() time.Duration {
	next, ok := s.state.nextBlockedTime()
	if !ok {
		return s.idleInterval
	}
	delay := time.Until(next)
	if delay < 0 {
		return 0
	}
	if delay < s.idleInterval {
		return delay
	}
	return delay
}
