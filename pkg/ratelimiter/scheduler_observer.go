package ratelimiter

// SchedulerObserver receives scheduler lifecycle events for a job.
type SchedulerObserver interface {
	// OnReserveStart signals a reserve attempt.
	OnReserveStart(job Job)
	// OnReserveDenied signals a reserve denial with retry metadata.
	OnReserveDenied(job Job, res ReserveResponse)
	// OnReserveError signals a reserve error before retry.
	OnReserveError(job Job, err error)
}
