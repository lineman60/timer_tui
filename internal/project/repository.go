package project

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type Repository struct {
	db *sql.DB
}

func NewRepository() (*Repository, error) {
	db, err := sql.Open("sqlite", "timer_tui.db")
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	repo := &Repository{db: db}
	if err := repo.init(); err != nil {
		return nil, err
	}

	return repo, nil
}

func (r *Repository) init() error {
	query := `
	CREATE TABLE IF NOT EXISTS projects (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		max_time INTEGER NOT NULL,
		running INTEGER DEFAULT 0,
		elapsed INTEGER DEFAULT 0
	)
	`
	_, err := r.db.Exec(query)
	return err
}

func (r *Repository) GetAll() ([]Project, error) {
	rows, err := r.db.Query("SELECT id, name, max_time, running, elapsed FROM projects")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []Project
	for rows.Next() {
		var p Project
		var maxTime, elapsed int64
		var running int
		if err := rows.Scan(&p.ID, &p.Name, &maxTime, &running, &elapsed); err != nil {
			return nil, err
		}
		p.MaxTime = time.Duration(maxTime)
		p.Running = running == 1
		p.Elapsed = time.Duration(elapsed)
		projects = append(projects, p)
	}
	return projects, nil
}

func (r *Repository) GetByID(id int64) (*Project, error) {
	var p Project
	var maxTime, elapsed int64
	var running int
	err := r.db.QueryRow("SELECT id, name, max_time, running, elapsed FROM projects WHERE id = ?", id).
		Scan(&p.ID, &p.Name, &maxTime, &running, &elapsed)
	if err != nil {
		return nil, err
	}
	p.MaxTime = time.Duration(maxTime)
	p.Running = running == 1
	p.Elapsed = time.Duration(elapsed)
	return &p, nil
}

func (r *Repository) Create(name string, maxTime time.Duration) (*Project, error) {
	result, err := r.db.Exec(
		"INSERT INTO projects (name, max_time, running, elapsed) VALUES (?, ?, 0, 0)",
		name, int64(maxTime),
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &Project{
		ID:      id,
		Name:    name,
		MaxTime: maxTime,
		Running: false,
		Elapsed: 0,
	}, nil
}

func (r *Repository) Update(p *Project) error {
	running := 0
	if p.Running {
		running = 1
	}
	_, err := r.db.Exec(
		"UPDATE projects SET name = ?, max_time = ?, running = ?, elapsed = ? WHERE id = ?",
		p.Name, int64(p.MaxTime), running, int64(p.Elapsed), p.ID,
	)
	return err
}

func (r *Repository) Delete(id int64) error {
	_, err := r.db.Exec("DELETE FROM projects WHERE id = ?", id)
	return err
}

func (r *Repository) StopAllTimers() error {
	_, err := r.db.Exec("UPDATE projects SET running = 0")
	return err
}

func (r *Repository) Close() error {
	return r.db.Close()
}

func ParseDuration(input string) (time.Duration, error) {
	var d time.Duration
	_, err := fmt.Sscanf(input, "%d", &d)
	if err == nil {
		return d * time.Minute, nil
	}

	d, err = time.ParseDuration(input)
	if err == nil {
		return d, nil
	}

	return 0, fmt.Errorf("invalid duration format")
}
