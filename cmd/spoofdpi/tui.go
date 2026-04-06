package main

import (
	_ "embed"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/xvzc/spoofdpi/internal/netutil"
)

// UI styling
var (
	logoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00ff00")).
			Bold(true).
			MarginBottom(0)

	speedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Bold(true).
			MarginBottom(0)

	logStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	spinnerFrames = []string{
		"⢀⠀", "⡀⠀", "⠄⠀", "⢂⠀", "⡂⠀", "⠅⠀", "⢃⠀", "⡃⠀", "⠍⠀", "⢋⠀", "⡋⠀", "⠍⠁", "⢋⠁", "⡋⠁",
		"⠍⠉", "⠋⠉", "⠋⠉", "⠉⠙", "⠉⠙", "⠉⠩", "⠈⢙", "⠈⡙", "⢈⠩", "⡀⢙", "⠄⡙", "⢂⠩", "⡂⢘", "⠅⡘",
		"⢃⠨", "⡃⢐", "⠍⡐", "⢋⠠", "⡋⢀", "⠍⡁", "⢋⠁", "⡋⠁", "⠍⠉", "⠋⠉", "⠋⠉", "⠉⠙", "⠉⠙", "⠉⠩",
		"⠈⢙", "⠈⡙", "⠈⠩", "⠀⢙", "⠀⡙", "⠀⠩", "⠀⢘", "⠀⡘", "⠀⠨", "⠀⢐", "⠀⡐", "⠀⠠", "⠀⢀", "⠀⡀",
	}
)

//go:embed logo.txt
var logo string

type (
	tickMsg time.Time
	logMsg  string
)

type model struct {
	viewport     viewport.Model
	spinner      spinner.Model
	logs         []string
	speed        string
	lastTxBytes  uint64
	lastRxBytes  uint64
	lastTickTime time.Time
	ready        bool
}

func formatSpeed(up, down float64) string {
	return fmt.Sprintf("↑ %8.2f KB/s │ ↓ %8.2f KB/s", up, down)
}

func initialModel() model {
	s := spinner.New()
	s.Spinner = spinner.Spinner{
		Frames: spinnerFrames,
		FPS:    time.Second / 25,
	}

	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("119"))

	return model{
		spinner:      s,
		logs:         []string{},
		speed:        formatSpeed(0, 0),
		lastTickTime: time.Now(),
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		tickCmd(),
		m.spinner.Tick,
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		headerHeight := lipgloss.Height(m.headerView())
		footerHeight := 1
		verticalMarginHeight := headerHeight + footerHeight

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
			m.viewport.YPosition = headerHeight
			m.viewport.SetContent(strings.Join(m.logs, "\n"))
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMarginHeight
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case tickMsg:
		now := time.Time(msg)
		elapsed := now.Sub(m.lastTickTime).Seconds()
		if elapsed <= 0.0 {
			elapsed = 1.0
		}

		currentRx := netutil.GetRxBytes()
		currentTx := netutil.GetTxBytes()

		upSpeed := (float64(currentTx-m.lastTxBytes) / 1024.0) / elapsed
		downSpeed := (float64(currentRx-m.lastRxBytes) / 1024.0) / elapsed

		m.lastRxBytes = currentRx
		m.lastTxBytes = currentTx
		m.lastTickTime = now

		m.speed = formatSpeed(upSpeed, downSpeed)
		cmds = append(cmds, tickCmd())

	case logMsg:
		m.logs = append(m.logs, string(msg))

		const maxLogs = 5000
		if len(m.logs) > maxLogs {
			m.logs = m.logs[len(m.logs)-maxLogs:]
		}

		if m.ready {
			isAtBottom := m.viewport.AtBottom()
			m.viewport.SetContent(strings.Join(m.logs, "\n"))

			if isAtBottom {
				m.viewport.GotoBottom()
			}
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m model) headerView() string {
	logoView := logoStyle.Render(strings.TrimPrefix(logo, "\n"))

	speedText := fmt.Sprintf("%s %s", m.spinner.View(), m.speed)
	speedView := speedStyle.Render(speedText)

	width := m.viewport.Width
	if width == 0 {
		width = 80 // fallback width before WindowSizeMsg
	}
	divider := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(strings.Repeat("─", width))

	return lipgloss.JoinVertical(lipgloss.Left, logoView, speedView, divider)
}

func (m model) footerView() string {
	if !m.ready {
		return ""
	}

	help := "[filter (f)] [clear filter (F)] [clear logs (R)] [quit (q)]"
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("249")).
		MarginRight(0)

	total := m.viewport.TotalLineCount()

	current := m.viewport.YOffset + m.viewport.Height
	if m.viewport.AtBottom() || current > total {
		current = total
	}

	info := fmt.Sprintf("%d / %d lines", current, total)

	return lipgloss.JoinHorizontal(
		lipgloss.Left,
		helpStyle.Render(help),
		lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Width(m.viewport.Width-lipgloss.Width(help)).
			Align(lipgloss.Right).
			Render(info),
	)
}

func (m model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	return fmt.Sprintf(
		"%s\n%s\n%s",
		m.headerView(),
		logStyle.Render(m.viewport.View()),
		m.footerView(),
	)
}

var (
	p     *tea.Program
	ready = make(chan struct{})
)

type TuiWriter struct{}

func (TuiWriter) Write(b []byte) (n int, err error) {
	<-ready
	if p != nil {
		p.Send(logMsg(strings.TrimSpace(string(b))))
	}
	return len(b), nil
}

func startTUI() error {
	p = tea.NewProgram(initialModel(), tea.WithAltScreen())
	close(ready)
	_, err := p.Run()
	return err
}
