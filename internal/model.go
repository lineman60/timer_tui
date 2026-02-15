package internal

import (
	"fmt"
	"strconv"
	"time"

	"timer_tui/internal/project"
	"timer_tui/internal/timelog"
	"timer_tui/internal/timer"

	tea "github.com/charmbracelet/bubbletea"
)

type MsgTick struct{}

type Model struct {
	Projects       []*project.Project
	SelectedIndex  int
	ShowAddForm    bool
	ShowEditForm   bool
	EditingProject *project.Project
	NewProjectName string
	NewProjectTime string
	InputFocus     int
	Err            error
	Timers         map[int64]*timer.Timer
	repo           *project.Repository

	// Session tracking for time logs
	SessionStarts map[int64]time.Time // tracks when each project's current session started

	// Tag input state (shown after stopping a timer)
	ShowTagInput bool
	TagInput     string
	PendingLog   *timelog.TimeLog // the log entry waiting for a tag

	// Time logs per project
	TimeLogs map[int64][]timelog.TimeLog

	// All-logs viewer state
	ShowLogView   bool
	LogViewScroll int
	AllLogs       []project.LogWithProject
}

func NewModel() (*Model, error) {
	repo, err := project.NewRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	projectList, err := repo.GetAll()
	if err != nil {
		repo.Close()
		return nil, fmt.Errorf("failed to load projects: %w", err)
	}

	projects := make([]*project.Project, len(projectList))
	for i := range projectList {
		projects[i] = &projectList[i]
	}

	timers := make(map[int64]*timer.Timer)
	sessionStarts := make(map[int64]time.Time)
	for i := range projects {
		timers[projects[i].ID] = timer.New()
		if projects[i].Running {
			timers[projects[i].ID].Start()
			sessionStarts[projects[i].ID] = time.Now()
		}
	}

	// Load time logs for all projects
	timeLogs := make(map[int64][]timelog.TimeLog)
	for _, p := range projects {
		logs, err := repo.GetLogsByProject(p.ID)
		if err == nil {
			timeLogs[p.ID] = logs
		}
	}

	m := &Model{
		Projects:      projects,
		SelectedIndex: 0,
		ShowAddForm:   false,
		ShowEditForm:  false,
		Timers:        timers,
		repo:          repo,
		SessionStarts: sessionStarts,
		TimeLogs:      timeLogs,
	}

	return m, nil
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case MsgTick:
		for _, p := range m.Projects {
			t := m.Timers[p.ID]
			if t.Running() {
				p.Elapsed = t.Elapsed()
			}
		}
		return m, nil
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	case tea.WindowSizeMsg:
		return m, nil
	}
	return m, nil
}

func (m *Model) View() string {
	if m.ShowTagInput {
		return m.tagInputView()
	}

	if m.ShowLogView {
		return m.allLogsView()
	}

	if len(m.Projects) == 0 && !m.ShowAddForm {
		return m.emptyStateView()
	}

	if m.ShowAddForm {
		return m.addFormView()
	}

	if m.ShowEditForm {
		return m.editFormView()
	}

	return m.mainView()
}

func (m *Model) SelectedProject() *project.Project {
	if m.SelectedIndex >= 0 && m.SelectedIndex < len(m.Projects) {
		return m.Projects[m.SelectedIndex]
	}
	return nil
}

func (m *Model) SelectedTimer() *timer.Timer {
	p := m.SelectedProject()
	if p == nil {
		return nil
	}
	return m.Timers[p.ID]
}

func (m *Model) AddProject(name string, maxTime time.Duration) error {
	p, err := m.repo.Create(name, maxTime)
	if err != nil {
		return err
	}
	m.Timers[p.ID] = timer.New()
	m.TimeLogs[p.ID] = nil
	m.Projects = append(m.Projects, p)
	m.SelectedIndex = len(m.Projects) - 1
	return nil
}

func (m *Model) UpdateProject(p *project.Project) error {
	if err := m.repo.Update(p); err != nil {
		return err
	}
	for i, proj := range m.Projects {
		if proj.ID == p.ID {
			m.Projects[i] = p
			break
		}
	}
	return nil
}

func (m *Model) DeleteProject(id int64) error {
	if err := m.repo.Delete(id); err != nil {
		return err
	}
	delete(m.Timers, id)
	delete(m.SessionStarts, id)
	delete(m.TimeLogs, id)
	for i, p := range m.Projects {
		if p.ID == id {
			m.Projects = append(m.Projects[:i], m.Projects[i+1:]...)
			break
		}
	}
	if m.SelectedIndex >= len(m.Projects) {
		m.SelectedIndex = len(m.Projects) - 1
	}
	return nil
}

func (m *Model) StopAllTimers() {
	for _, p := range m.Projects {
		t := m.Timers[p.ID]
		if t.Running() {
			t.Stop()
			p.Elapsed = t.Elapsed()
			p.Running = false
			m.repo.Update(p)
			// Note: when stopping all timers due to starting another,
			// we silently log without a tag prompt
			if startedAt, ok := m.SessionStarts[p.ID]; ok {
				stoppedAt := time.Now()
				duration := stoppedAt.Sub(startedAt)
				log := &timelog.TimeLog{
					ProjectID: p.ID,
					StartedAt: startedAt,
					StoppedAt: stoppedAt,
					Duration:  duration,
					Tag:       "",
				}
				m.repo.CreateLog(log)
				m.TimeLogs[p.ID] = append([]timelog.TimeLog{*log}, m.TimeLogs[p.ID]...)
				delete(m.SessionStarts, p.ID)
			}
		}
	}
}

func (m *Model) Close() error {
	for _, p := range m.Projects {
		t := m.Timers[p.ID]
		if t.Running() {
			t.Stop()
			p.Elapsed = t.Elapsed()
			p.Running = false
			// Log any running sessions on close
			if startedAt, ok := m.SessionStarts[p.ID]; ok {
				stoppedAt := time.Now()
				duration := stoppedAt.Sub(startedAt)
				log := &timelog.TimeLog{
					ProjectID: p.ID,
					StartedAt: startedAt,
					StoppedAt: stoppedAt,
					Duration:  duration,
					Tag:       "",
				}
				m.repo.CreateLog(log)
			}
			m.repo.Update(p)
		}
	}
	return m.repo.Close()
}

func (m *Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.ShowTagInput {
		return m.handleTagInput(msg)
	}

	if m.ShowLogView {
		return m.handleLogViewInput(msg)
	}

	if m.ShowAddForm || m.ShowEditForm {
		return m.handleFormInput(msg)
	}

	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "up", "k":
		if m.SelectedIndex > 0 {
			m.SelectedIndex--
		}
	case "down", "j":
		if m.SelectedIndex < len(m.Projects)-1 {
			m.SelectedIndex++
		}
	case "enter":
		p := m.SelectedProject()
		if p != nil {
			t := m.SelectedTimer()
			if t.Running() {
				// Stop the timer and show tag input prompt
				t.Stop()
				p.Elapsed = t.Elapsed()
				p.Running = false
				m.repo.Update(p)

				stoppedAt := time.Now()
				startedAt := stoppedAt // fallback
				if sa, ok := m.SessionStarts[p.ID]; ok {
					startedAt = sa
					delete(m.SessionStarts, p.ID)
				}
				duration := stoppedAt.Sub(startedAt)

				m.PendingLog = &timelog.TimeLog{
					ProjectID: p.ID,
					StartedAt: startedAt,
					StoppedAt: stoppedAt,
					Duration:  duration,
					Tag:       "",
				}
				m.TagInput = ""
				m.ShowTagInput = true
			} else {
				// Stop all other timers first (will auto-log them without tag)
				m.StopAllTimers()
				t.SetElapsed(p.Elapsed)
				t.Start()
				p.Running = true
				m.SessionStarts[p.ID] = time.Now()
				m.repo.Update(p)
			}
		}
	case "n":
		m.ShowAddForm = true
		m.NewProjectName = ""
		m.NewProjectTime = ""
		m.InputFocus = 0
	case "e":
		p := m.SelectedProject()
		if p != nil {
			m.ShowEditForm = true
			m.EditingProject = p
			m.NewProjectName = p.Name
			m.NewProjectTime = fmt.Sprintf("%d", int(p.MaxTime.Minutes()))
			m.InputFocus = 0
		}
	case "d":
		p := m.SelectedProject()
		if p != nil {
			m.DeleteProject(p.ID)
		}
	case "r":
		p := m.SelectedProject()
		if p != nil {
			t := m.SelectedTimer()
			t.Reset()
			p.Elapsed = 0
			p.Running = false
			delete(m.SessionStarts, p.ID)
			m.repo.Update(p)
		}
	case "l":
		// Open the all-logs viewer
		allLogs, err := m.repo.GetAllLogs()
		if err == nil {
			m.AllLogs = allLogs
		} else {
			m.AllLogs = nil
		}
		m.ShowLogView = true
		m.LogViewScroll = 0
	case "tab":
		m.InputFocus = 1 - m.InputFocus
	}
	return m, nil
}

func (m *Model) handleLogViewInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q", "esc", "l":
		m.ShowLogView = false
		m.AllLogs = nil
	case "up", "k":
		if m.LogViewScroll > 0 {
			m.LogViewScroll--
		}
	case "down", "j":
		maxScroll := len(m.AllLogs) - 1
		if maxScroll < 0 {
			maxScroll = 0
		}
		if m.LogViewScroll < maxScroll {
			m.LogViewScroll++
		}
	}
	return m, nil
}

func (m *Model) handleTagInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc":
		// Save the log without a tag
		if m.PendingLog != nil {
			m.PendingLog.Tag = ""
			m.repo.CreateLog(m.PendingLog)
			m.TimeLogs[m.PendingLog.ProjectID] = append(
				[]timelog.TimeLog{*m.PendingLog},
				m.TimeLogs[m.PendingLog.ProjectID]...,
			)
			m.PendingLog = nil
		}
		m.ShowTagInput = false
		m.TagInput = ""
	case "enter":
		// Save the log with the tag
		if m.PendingLog != nil {
			m.PendingLog.Tag = m.TagInput
			m.repo.CreateLog(m.PendingLog)
			m.TimeLogs[m.PendingLog.ProjectID] = append(
				[]timelog.TimeLog{*m.PendingLog},
				m.TimeLogs[m.PendingLog.ProjectID]...,
			)
			m.PendingLog = nil
		}
		m.ShowTagInput = false
		m.TagInput = ""
	case "backspace":
		if len(m.TagInput) > 0 {
			m.TagInput = m.TagInput[:len(m.TagInput)-1]
		}
	default:
		runes := []rune(msg.String())
		if len(runes) == 1 {
			m.TagInput += string(runes[0])
		}
	}
	return m, nil
}

func (m *Model) handleFormInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc":
		m.ShowAddForm = false
		m.ShowEditForm = false
		m.EditingProject = nil
	case "enter":
		if m.InputFocus == 0 {
			m.InputFocus = 1
		} else {
			if m.ShowAddForm {
				minutes := 0
				if m.NewProjectTime != "" {
					if v, err := strconv.Atoi(m.NewProjectTime); err == nil {
						minutes = v
					}
				}
				duration := time.Duration(minutes) * time.Minute
				if duration <= 0 {
					duration = 25 * time.Minute
				}
				m.AddProject(m.NewProjectName, duration)
			} else if m.ShowEditForm && m.EditingProject != nil {
				minutes := 0
				if m.NewProjectTime != "" {
					if v, err := strconv.Atoi(m.NewProjectTime); err == nil {
						minutes = v
					}
				}
				duration := time.Duration(minutes) * time.Minute
				if duration <= 0 {
					duration = 25 * time.Minute
				}
				m.EditingProject.Name = m.NewProjectName
				m.EditingProject.MaxTime = duration
				if m.EditingProject.Elapsed > duration {
					m.EditingProject.Elapsed = duration
				}
				m.UpdateProject(m.EditingProject)
			}
			m.ShowAddForm = false
			m.ShowEditForm = false
			m.EditingProject = nil
		}
	case "backspace":
		if m.InputFocus == 0 {
			if len(m.NewProjectName) > 0 {
				m.NewProjectName = m.NewProjectName[:len(m.NewProjectName)-1]
			}
		} else {
			if len(m.NewProjectTime) > 0 {
				m.NewProjectTime = m.NewProjectTime[:len(m.NewProjectTime)-1]
			}
		}
	case "tab":
		m.InputFocus = 1 - m.InputFocus
	default:
		if msg.String() == "tab" || msg.String() == "shift+tab" {
			break
		}
		runes := []rune(msg.String())
		if len(runes) == 1 {
			if m.InputFocus == 0 {
				m.NewProjectName += string(runes[0])
			} else {
				if runes[0] >= '0' && runes[0] <= '9' {
					m.NewProjectTime += string(runes[0])
				}
			}
		}
	}
	return m, nil
}
