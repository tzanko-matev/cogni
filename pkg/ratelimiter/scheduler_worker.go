package ratelimiter

import (
	"context"
	"time"
)

// worker consumes jobs from the work channel and executes them.
func (s *Scheduler) worker() {
	defer s.wg.Done()
	for job := range s.workCh {
		s.handleJob(job)
	}
}

// handleJob runs a single reserve/execute/complete attempt.
func (s *Scheduler) handleJob(job Job) {
	job = s.ensureLeaseID(job)
	req := buildReserveRequest(job)
	res, err := s.limiter.Reserve(s.ctx, req)
	if err != nil {
		s.requeue(job, s.now().Add(s.errorRetryDelay))
		return
	}
	if !res.Allowed {
		job.LeaseID = s.newLeaseID()
		s.requeue(job, s.now().Add(s.retryDelay(res)))
		return
	}
	actuals := []Actual{}
	if job.Execute != nil {
		actualTokens, _ := job.Execute(s.ctx)
		actuals = buildLLMActuals(job, actualTokens)
	}
	s.complete(job, actuals)
}

// ensureLeaseID assigns a lease ID if one is missing.
func (s *Scheduler) ensureLeaseID(job Job) Job {
	if job.LeaseID == "" {
		job.LeaseID = s.newLeaseID()
	}
	return job
}

// retryDelay calculates retry timing for a denied reservation.
func (s *Scheduler) retryDelay(res ReserveResponse) time.Duration {
	delay := time.Duration(res.RetryAfterMs) * time.Millisecond
	if delay < 0 {
		delay = 0
	}
	jitter := s.jitter(delay)
	if jitter < 0 {
		jitter = 0
	}
	return delay + jitter
}

// buildReserveRequest creates a reserve request for a job.
func buildReserveRequest(job Job) ReserveRequest {
	reqs := BuildLLMRequirements(LLMReserveInput{
		LeaseID:         job.LeaseID,
		JobID:           job.JobID,
		TenantID:        job.TenantID,
		Provider:        job.Provider,
		Model:           job.Model,
		Prompt:          job.Prompt,
		MaxOutputTokens: job.MaxOutputTokens,
		WantDailyBudget: job.WantDailyBudget,
	})
	return ReserveRequest{LeaseID: job.LeaseID, JobID: job.JobID, Requirements: reqs}
}

// buildLLMActuals builds actual usage entries for token-based limits.
func buildLLMActuals(job Job, actualTokens uint64) []Actual {
	actuals := []Actual{
		{Key: LimitKey(buildTPMKey(job.Provider, job.Model)), ActualAmount: actualTokens},
	}
	if job.WantDailyBudget {
		actuals = append(actuals, Actual{
			Key:          LimitKey(buildDailyKey(job.TenantID)),
			ActualAmount: actualTokens,
		})
	}
	return actuals
}

// complete reports completion to the limiter, ignoring errors.
func (s *Scheduler) complete(job Job, actuals []Actual) {
	_, _ = s.limiter.Complete(context.Background(), CompleteRequest{
		LeaseID: job.LeaseID,
		JobID:   job.JobID,
		Actuals: actuals,
	})
}
