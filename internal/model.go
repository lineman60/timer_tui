package internal

import (
	"fmt"
	"strconv"
	"time"

	"timer_tui/internal/project"
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
	for i := range projects {
		timers[projects[i].ID] = timer.New()
		if projects[i].Running {
			timers[projects[i].ID].Start()
		}
	}

	m := &Model{
		Projects:      projects,
		SelectedIndex: 0,
		ShowAddForm:   false,
		ShowEditForm:  false,
		Timers:        timers,
		repo:          repo,
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
	for _, t := range m.Timers {
		t.Stop()
	}
	for _, p := range m.Projects {
		p.Running = false
		m.repo.Update(p)
	}
}

func (m *Model) Close() error {
	for _, t := range m.Timers {
		t.Stop()
	}
	return m.repo.Close()
}

func (m *Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
				t.Stop()
				p.Running = false
				p.Elapsed = t.Elapsed()
				m.repo.Update(p)
			} else {
				m.StopAllTimers()
				t.SetElapsed(p.Elapsed)
				t.Start()
				p.Running = true
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
			m.repo.Update(p)
		}
	case "tab":
		m.InputFocus = 1 - m.InputFocus
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
