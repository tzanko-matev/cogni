package memory

import "time"

// Clock provides the current time for the backend.
type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time {
	return time.Now()
}
