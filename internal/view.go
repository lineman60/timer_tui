package internal

import (
	"fmt"
	"strings"
	"time"

	"timer_tui/internal/timelog"

	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Bold(true).
			Align(lipgloss.Center)

	projectItemStyle = lipgloss.NewStyle().
				Padding(0, 1)

	projectItemSelectedStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("170")).
					Background(lipgloss.Color("235")).
					Padding(0, 1)

	timerDisplayStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("69")).
				Bold(true)

	timerRunningStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("82")).
				Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 0)

	inputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("170"))

	inputInactiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240"))

	logHeaderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Bold(true)

	logTagStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("170"))

	logTimeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
)

func formatDuration(d time.Duration) string {
	total := int(d.Seconds())
	hours := total / 3600
	minutes := (total % 3600) / 60
	seconds := total % 60

	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
	}
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

func (m *Model) emptyStateView() string {
	return lipgloss.Place(
		80, 24,
		lipgloss.Center, lipgloss.Center,
		titleStyle.Render("Timer TUI")+"\n\n"+
			inactiveStyle.Render("No projects yet. Press 'n' to add one."),
	)
}

func (m *Model) mainView() string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Width(80).Render("Timer TUI"))
	sb.WriteString("\n\n")

	boxes := lipgloss.JoinHorizontal(lipgloss.Top,
		m.projectListView(),
		"  ",
		m.projectDetailView(),
	)
	sb.WriteString(boxes)
	sb.WriteString("\n\n")
	sb.WriteString(helpStyle.Render("Navigate: Up/Down | Start/Stop: Enter | New: n | Edit: e | Delete: d | Reset: r | Quit: q"))

	return sb.String()
}

func (m *Model) projectListView() string {
	var sb strings.Builder

	sb.WriteString("Projects\n\n")

	for i, p := range m.Projects {
		t := m.Timers[p.ID]
		running := ""
		if t.Running() {
			running = " ●"
		}
		remaining := p.MaxTime - p.Elapsed
		remaining = max(remaining, 0)
		timerStr := formatDuration(remaining)

		line := fmt.Sprintf("%s %s%s", p.Name, timerStr, running)

		if i == m.SelectedIndex {
			sb.WriteString(projectItemSelectedStyle.Render(line))
		} else {
			sb.WriteString(projectItemStyle.Render(inactiveStyle.Render(line)))
		}
		sb.WriteString("\n")
	}

	return boxStyle.Width(25).Height(15).Render(sb.String())
}

func (m *Model) projectDetailView() string {
	p := m.SelectedProject()
	if p == nil {
		return boxStyle.Width(45).Height(15).Render("Select a project")
	}

	t := m.Timers[p.ID]
	remaining := p.MaxTime - p.Elapsed
	remaining = max(remaining, 0)

	var timerStr string
	if t.Running() {
		timerStr = timerRunningStyle.Render(formatDuration(remaining))
	} else {
		timerStr = timerDisplayStyle.Render(formatDuration(remaining))
	}

	status := "Stopped"
	statusStyle := inactiveStyle
	if t.Running() {
		status = "Running"
		statusStyle = runningStyle
	}

	maxTimeStr := fmt.Sprintf("Max: %s", formatDuration(p.MaxTime))

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Project: %s\n\n", p.Name))
	sb.WriteString(timerStr)
	sb.WriteString(fmt.Sprintf("\n\n%s\n", statusStyle.Render(status)))
	sb.WriteString(fmt.Sprintf("%s\n", maxTimeStr))

	// Show recent time logs
	logs := m.TimeLogs[p.ID]
	if len(logs) > 0 {
		sb.WriteString("\n")
		sb.WriteString(logHeaderStyle.Render("Recent Logs"))
		sb.WriteString("\n")
		displayCount := len(logs)
		if displayCount > 5 {
			displayCount = 5
		}
		for _, l := range logs[:displayCount] {
			sb.WriteString(m.formatLogEntry(l))
			sb.WriteString("\n")
		}
	}

	return boxStyle.Width(45).Height(15).Render(sb.String())
}

func (m *Model) addFormView() string {
	var sb strings.Builder
	sb.WriteString(titleStyle.Width(80).Render("Add New Project"))
	sb.WriteString("\n\n")

	// Add a visible focus marker so it's obvious which field is active.
	nameMarker := "  "
	if m.InputFocus == 0 {
		nameMarker = "→ "
	}
	nameLabel := fmt.Sprintf("%sProject Name: ", nameMarker)
	if m.InputFocus == 0 {
		nameLabel = inputStyle.Render(nameLabel)
	} else {
		nameLabel = inputInactiveStyle.Render(nameLabel)
	}

	timeMarker := "  "
	if m.InputFocus == 1 {
		timeMarker = "→ "
	}
	timeLabel := fmt.Sprintf("%sDuration (min): ", timeMarker)
	if m.InputFocus == 1 {
		timeLabel = inputStyle.Render(timeLabel)
	} else {
		timeLabel = inputInactiveStyle.Render(timeLabel)
	}

	nameValue := m.NewProjectName
	if m.InputFocus == 0 {
		nameValue = inputStyle.Render(nameValue + "\u2588")
	}

	timeValue := m.NewProjectTime
	if m.InputFocus == 1 {
		timeValue = inputStyle.Render(timeValue + "\u2588")
	}

	// Show which field is currently focused in the help line to make tab behavior explicit
	focusName := "Project Name"
	if m.InputFocus == 1 {
		focusName = "Duration"
	}
	helpText := fmt.Sprintf("Tab: Switch (Focused: %s) | Enter: Save | Esc: Cancel", focusName)

	form := fmt.Sprintf("%s%s\n\n%s%s\n\n%s",
		nameLabel, nameValue,
		timeLabel, timeValue,
		helpStyle.Render(helpText),
	)

	return lipgloss.Place(
		80, 24,
		lipgloss.Center, lipgloss.Center,
		boxStyle.Width(50).Render(form),
	)
}

func (m *Model) editFormView() string {
	var sb strings.Builder
	sb.WriteString(titleStyle.Width(80).Render("Edit Project"))
	sb.WriteString("\n\n")

	// Add a visible focus marker so it's obvious which field is active.
	nameMarker := "  "
	if m.InputFocus == 0 {
		nameMarker = "→ "
	}
	nameLabel := fmt.Sprintf("%sProject Name: ", nameMarker)
	if m.InputFocus == 0 {
		nameLabel = inputStyle.Render(nameLabel)
	} else {
		nameLabel = inputInactiveStyle.Render(nameLabel)
	}

	timeMarker := "  "
	if m.InputFocus == 1 {
		timeMarker = "→ "
	}
	timeLabel := fmt.Sprintf("%sDuration (min): ", timeMarker)
	if m.InputFocus == 1 {
		timeLabel = inputStyle.Render(timeLabel)
	} else {
		timeLabel = inputInactiveStyle.Render(timeLabel)
	}

	nameValue := m.NewProjectName
	if m.InputFocus == 0 {
		nameValue = inputStyle.Render(nameValue + "\u2588")
	}

	timeValue := m.NewProjectTime
	if m.InputFocus == 1 {
		timeValue = inputStyle.Render(timeValue + "\u2588")
	}

	// Show which field is currently focused in the help line to make tab behavior explicit
	focusName := "Project Name"
	if m.InputFocus == 1 {
		focusName = "Duration"
	}
	helpText := fmt.Sprintf("Tab: Switch (Focused: %s) | Enter: Save | Esc: Cancel", focusName)

	form := fmt.Sprintf("%s%s\n\n%s%s\n\n%s",
		nameLabel, nameValue,
		timeLabel, timeValue,
		helpStyle.Render(helpText),
	)

	return lipgloss.Place(
		80, 24,
		lipgloss.Center, lipgloss.Center,
		boxStyle.Width(50).Render(form),
	)
}

func (m *Model) tagInputView() string {
	var sb strings.Builder
	sb.WriteString(titleStyle.Width(80).Render("Log Time Session"))
	sb.WriteString("\n\n")

	durationStr := ""
	if m.PendingLog != nil {
		durationStr = formatDuration(m.PendingLog.Duration)
	}

	label := inputStyle.Render("→ Tag: ")
	value := inputStyle.Render(m.TagInput + "\u2588")

	form := fmt.Sprintf(
		"%s\n\n%s%s\n\n%s",
		fmt.Sprintf("Session duration: %s", timerDisplayStyle.Render(durationStr)),
		label, value,
		helpStyle.Render("Enter: Save | Esc: Skip (no tag)"),
	)

	return lipgloss.Place(
		80, 24,
		lipgloss.Center, lipgloss.Center,
		boxStyle.Width(50).Render(form),
	)
}

func (m *Model) formatLogEntry(l timelog.TimeLog) string {
	timeStr := logTimeStyle.Render(l.StoppedAt.Format("Jan 02 15:04"))
	dur := formatDuration(l.Duration)
	tag := ""
	if l.Tag != "" {
		tag = " " + logTagStyle.Render("["+l.Tag+"]")
	}
	return fmt.Sprintf("  %s  %s%s", timeStr, dur, tag)
}

var (
	inactiveStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))
	runningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")).
			Bold(true)
)
