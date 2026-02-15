package timelog

import "time"

// TimeLog represents a recorded timer session for a project.
type TimeLog struct {
	ID        int64
	ProjectID int64
	StartedAt time.Time
	StoppedAt time.Time
	Duration  time.Duration
	Tag       string
}
