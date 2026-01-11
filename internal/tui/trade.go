package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// TradeModel holds the state for the trade view.
type TradeModel struct {
	// Symbol to pre-populate from watchlist selection
	Symbol string
}

// NewTradeModel creates a new trade model.
func NewTradeModel() *TradeModel {
	return &TradeModel{}
}

// SetSymbol sets the symbol to trade.
func (m *TradeModel) SetSymbol(symbol string) {
	m.Symbol = symbol
}

// Update handles messages for the trade view.
func (m *TradeModel) Update(msg tea.Msg) (*TradeModel, tea.Cmd) {
	// Placeholder - no handling yet
	return m, nil
}

// View renders the trade view.
func (m *TradeModel) View() string {
	var b strings.Builder
	b.WriteString(SummaryStyle.Render("Trade"))
	b.WriteString("\n\n")

	if m.Symbol != "" {
		b.WriteString(LabelStyle.Render("Symbol: "))
		b.WriteString(ValueStyle.Render(m.Symbol))
		b.WriteString("\n\n")
	}

	b.WriteString(LabelStyle.Render("Coming soon..."))
	b.WriteString("\n\n")
	b.WriteString(LabelStyle.Render("This view will allow you to place buy and sell orders."))
	return b.String()
}
