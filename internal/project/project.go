package project

import "time"

type Project struct {
	ID      int64
	Name    string
	MaxTime time.Duration
	Running bool
	Elapsed time.Duration
}

func NewProject(name string, maxTime time.Duration) *Project {
	return &Project{
		Name:    name,
		MaxTime: maxTime,
		Running: false,
		Elapsed: 0,
	}
}

func (p *Project) Remaining() time.Duration {
	if p.Elapsed >= p.MaxTime {
		return 0
	}
	return p.MaxTime - p.Elapsed
}

func (p *Project) IsComplete() bool {
	return p.Elapsed >= p.MaxTime
}
