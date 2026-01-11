package cmd

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// View represents the current active view in the TUI.
type view int

const (
	viewPortfolio view = iota
	viewWatchlist
	viewOrders
	viewTrade
)

// Model is the main bubbletea model for the TUI.
type Model struct {
	currentView view
	width       int
	height      int
	ready       bool
}

// Styles for the TUI
var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39")).
			Background(lipgloss.Color("236")).
			Padding(0, 1)

	contentStyle = lipgloss.NewStyle().
			Padding(1, 2)

	keyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Bold(true)

	descStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
)

// newModel creates a new TUI model.
func newModel() Model {
	return Model{
		currentView: viewPortfolio,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "1":
			m.currentView = viewPortfolio
		case "2":
			m.currentView = viewWatchlist
		case "3":
			m.currentView = viewOrders
		case "4":
			m.currentView = viewTrade
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
	}
	return m, nil
}

// View implements tea.Model.
func (m Model) View() string {
	if !m.ready {
		return "Loading..."
	}

	header := m.renderHeader()
	footer := m.renderFooter()
	content := m.renderContent()

	// Calculate content height
	headerHeight := lipgloss.Height(header)
	footerHeight := lipgloss.Height(footer)
	contentHeight := m.height - headerHeight - footerHeight

	// Pad content to fill available space
	contentLines := strings.Split(content, "\n")
	for len(contentLines) < contentHeight {
		contentLines = append(contentLines, "")
	}
	if len(contentLines) > contentHeight {
		contentLines = contentLines[:contentHeight]
	}
	content = strings.Join(contentLines, "\n")

	return header + "\n" + content + "\n" + footer
}

// renderHeader renders the header bar.
func (m Model) renderHeader() string {
	title := headerStyle.Render("pub")

	tabs := []struct {
		name   string
		key    string
		active bool
	}{
		{"Portfolio", "1", m.currentView == viewPortfolio},
		{"Watchlist", "2", m.currentView == viewWatchlist},
		{"Orders", "3", m.currentView == viewOrders},
		{"Trade", "4", m.currentView == viewTrade},
	}

	var tabStrs []string
	for _, tab := range tabs {
		style := lipgloss.NewStyle().Padding(0, 1)
		if tab.active {
			style = style.Bold(true).Foreground(lipgloss.Color("39"))
		} else {
			style = style.Foreground(lipgloss.Color("241"))
		}
		tabStrs = append(tabStrs, style.Render(fmt.Sprintf("[%s] %s", tab.key, tab.name)))
	}

	tabBar := strings.Join(tabStrs, " ")
	headerContent := title + "  " + tabBar

	// Pad to full width
	padding := m.width - lipgloss.Width(headerContent)
	if padding > 0 {
		headerContent += strings.Repeat(" ", padding)
	}

	return lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Width(m.width).
		Render(headerContent)
}

// renderContent renders the main content area.
func (m Model) renderContent() string {
	var content string
	switch m.currentView {
	case viewPortfolio:
		content = "Portfolio view - Coming soon"
	case viewWatchlist:
		content = "Watchlist view - Coming soon"
	case viewOrders:
		content = "Orders view - Coming soon"
	case viewTrade:
		content = "Trade view - Coming soon"
	}
	return contentStyle.Render(content)
}

// renderFooter renders the footer bar with key hints.
func (m Model) renderFooter() string {
	keys := []struct {
		key  string
		desc string
	}{
		{"1-4", "switch view"},
		{"q", "quit"},
	}

	var parts []string
	for _, k := range keys {
		parts = append(parts, keyStyle.Render(k.key)+" "+descStyle.Render(k.desc))
	}

	footerContent := strings.Join(parts, "  •  ")

	// Pad to full width
	padding := m.width - lipgloss.Width(footerContent)
	if padding > 0 {
		footerContent += strings.Repeat(" ", padding)
	}

	return lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Width(m.width).
		Render(footerContent)
}

func init() {
	uiCmd := &cobra.Command{
		Use:   "ui",
		Short: "Interactive terminal UI",
		Long: `Launch an interactive terminal UI for trading.

The UI provides a full-screen experience with keyboard navigation,
live data updates, and views for:
  - Portfolio: View your positions and account summary
  - Watchlist: Track symbols with live quotes
  - Orders: View and manage open orders
  - Trade: Buy and sell securities

Keyboard shortcuts:
  1-4     Switch between views
  q/esc   Quit the application`,
		RunE: func(cmd *cobra.Command, args []string) error {
			p := tea.NewProgram(newModel(), tea.WithAltScreen())
			_, err := p.Run()
			return err
		},
	}

	uiCmd.SilenceUsage = true
	rootCmd.AddCommand(uiCmd)
}
