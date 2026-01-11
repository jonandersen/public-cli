package tui

import (
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

// Color constants
const (
	ColorPrimary    = lipgloss.Color("39")  // Cyan/blue
	ColorMuted      = lipgloss.Color("241") // Gray
	ColorBackground = lipgloss.Color("236") // Dark gray
	ColorSelected   = lipgloss.Color("57")  // Purple
	ColorSelectedFg = lipgloss.Color("229") // Light yellow
	ColorGreen      = lipgloss.Color("82")  // Green for gains
	ColorRed        = lipgloss.Color("196") // Red for losses
	ColorWarning    = lipgloss.Color("220") // Yellow for warnings
)

// Shared styles
var (
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			Background(ColorBackground).
			Padding(0, 1)

	ContentStyle = lipgloss.NewStyle().
			Padding(1, 2)

	KeyStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)

	DescStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	SummaryStyle = lipgloss.NewStyle().Bold(true)

	LabelStyle = lipgloss.NewStyle().Foreground(ColorMuted)

	ValueStyle = lipgloss.NewStyle().Bold(true)

	GreenStyle = lipgloss.NewStyle().Foreground(ColorGreen)

	RedStyle = lipgloss.NewStyle().Foreground(ColorRed)

	ErrorStyle = lipgloss.NewStyle().Foreground(ColorRed)

	WarningStyle = lipgloss.NewStyle().
			Foreground(ColorWarning).
			Bold(true)

	InputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(0, 1)
)

// TableStyles returns the default table styles for TUI tables.
func TableStyles() table.Styles {
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(ColorSelectedFg).
		Background(ColorSelected).
		Bold(true)
	return s
}
