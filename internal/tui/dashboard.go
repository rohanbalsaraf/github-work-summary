package tui

import (
	"fmt"
	"strings"

	"github.com/RDX463/github-work-summary/internal/summary"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type state int

const (
	stateView state = iota
	stateEdit
)

// MainModel is the top-level state for the interactive dashboard.
type MainModel struct {
	Report     summary.Report
	state      state
	viewport   viewport.Model
	textarea   textarea.Model
	ready      bool
	width      int
	height     int
	LastAction string
	ExitReport *summary.Report
	NotifyFunc func(platform string, report summary.Report) error
}

func NewMainModel(r summary.Report, notify func(string, summary.Report) error) MainModel {
	ta := textarea.New()
	ta.Placeholder = "Edit AI Summary..."
	ta.Focus()
	ta.CharLimit = 1000
	ta.SetValue(r.AISummary)

	return MainModel{
		Report:     r,
		state:      stateView,
		textarea:   ta,
		NotifyFunc: notify,
	}
}

func (m MainModel) Init() tea.Cmd {
	return nil
}

func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
		cmds  []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case stateView:
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			case "enter":
				m.ExitReport = &m.Report
				return m, tea.Quit
			case "e":
				m.state = stateEdit
				m.textarea.SetValue(m.Report.AISummary)
				m.textarea.Focus()
				return m, textarea.Blink
			case "s":
				m.LastAction = "Sharing to Slack..."
				return m, m.SharingCmd("slack")
			case "d":
				m.LastAction = "Sharing to Discord..."
				return m, m.SharingCmd("discord")
			}

		case stateEdit:
			if msg.Type == tea.KeyEsc {
				m.state = stateView
				return m, nil
			}
			if msg.Type == tea.KeyCtrlS {
				m.Report.AISummary = m.textarea.Value()
				m.state = stateView
				m.viewport.SetContent(m.renderReport())
				return m, nil
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		headerHeight := 3
		footerHeight := 3
		verticalMargin := headerHeight + footerHeight

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMargin)
			m.viewport.YPosition = headerHeight
			m.viewport.SetContent(m.renderReport())
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMargin
		}

		m.textarea.SetWidth(msg.Width - 4)
		m.textarea.SetHeight(msg.Height - verticalMargin - 2)

	case ActionDoneMsg:
		m.LastAction = msg.Message
		return m, nil
	}

	if m.state == stateEdit {
		m.textarea, tiCmd = m.textarea.Update(msg)
		cmds = append(cmds, tiCmd)
	} else {
		m.viewport, vpCmd = m.viewport.Update(msg)
		cmds = append(cmds, vpCmd)
	}

	return m, tea.Batch(cmds...)
}

func (m MainModel) View() string {
	if !m.ready {
		return "\n  Initializing Dashboard..."
	}

	header := m.renderHeader()
	footer := m.renderFooter()

	var content string
	if m.state == stateEdit {
		content = "\n  " + lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10")).Render("EDIT MODE: Press Ctrl+S to save, Esc to cancel") + "\n\n" + m.textarea.View()
	} else {
		content = m.viewport.View()
	}

	return lipgloss.JoinVertical(lipgloss.Left, header, content, footer)
}

func (m MainModel) renderHeader() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("36")).
		Padding(0, 1).
		Render("GWS DASHBOARD")

	window := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(fmt.Sprintf("%s -> %s", m.Report.WindowStart.Format("15:04"), m.Report.WindowEnd.Format("15:04")))

	return lipgloss.JoinHorizontal(lipgloss.Center, title, " ", window) + "\n" + strings.Repeat("─", m.width)
}

func (m MainModel) renderFooter() string {
	var help string
	if m.state == stateEdit {
		help = "ctrl+s: save • esc: cancel"
	} else {
		help = "e: edit • s: slack • d: discord • enter: finalize • q: quit"
	}

	status := lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Render(m.LastAction)
	helpText := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(help)

	footerLine := strings.Repeat("─", m.width)
	return "\n" + footerLine + "\n" + helpText + "  " + status
}

func (m MainModel) renderReport() string {
	// Simple text rendering for the viewport
	var sb strings.Builder

	sb.WriteString("\n")
	if m.Report.AISummary != "" {
		sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10")).Render("✨ AI IMPACT SUMMARY") + "\n")
		sb.WriteString(m.Report.AISummary + "\n\n")
	}

	for _, repo := range m.Report.Repositories {
		sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("36")).Render("📦 "+repo.Repository) + "\n")

		if len(repo.Features) > 0 {
			sb.WriteString(fmt.Sprintf("  • Features: %d\n", len(repo.Features)))
		}
		if len(repo.BugFixes) > 0 {
			sb.WriteString(fmt.Sprintf("  • Bug Fixes: %d\n", len(repo.BugFixes)))
		}
		if len(repo.Maintenance) > 0 {
			sb.WriteString(fmt.Sprintf("  • Maintenance: %d\n", len(repo.Maintenance)))
		}
		if len(repo.PullRequests) > 0 {
			sb.WriteString(fmt.Sprintf("  • PRs: %d\n", len(repo.PullRequests)))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func (m MainModel) SharingCmd(platform string) tea.Cmd {
	return func() tea.Msg {
		if m.NotifyFunc == nil {
			return ActionDoneMsg{Message: "Error: No notify func"}
		}
		err := m.NotifyFunc(platform, m.Report)
		if err != nil {
			return ActionDoneMsg{Message: fmt.Sprintf("Error: %v", err)}
		}
		return ActionDoneMsg{Message: fmt.Sprintf("Shared to %s!", platform)}
	}
}

// Msg types
type ActionMsg struct{ Platform string }
type ActionDoneMsg struct{ Message string }
type ErrorMsg struct{ Err error }
