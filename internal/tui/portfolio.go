package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/jonandersen/public-cli/internal/api"
	"github.com/jonandersen/public-cli/internal/config"
	"github.com/jonandersen/public-cli/internal/keyring"
	"github.com/jonandersen/public-cli/pkg/publicapi"
)

// PortfolioState represents the loading state of portfolio data.
type PortfolioState int

const (
	PortfolioStateLoading PortfolioState = iota
	PortfolioStateLoaded
	PortfolioStateError
)

// PortfolioModel holds the state for the portfolio view.
type PortfolioModel struct {
	State       PortfolioState
	Data        Portfolio
	Err         error
	LastUpdated time.Time
	Table       table.Model
}

// NewPortfolioModel creates a new portfolio model.
func NewPortfolioModel() *PortfolioModel {
	cols := []table.Column{
		{Title: "Symbol", Width: 10},
		{Title: "Qty", Width: 8},
		{Title: "Price", Width: 10},
		{Title: "Value", Width: 12},
		{Title: "Day G/L", Width: 12},
		{Title: "Day %", Width: 8},
		{Title: "Total G/L", Width: 12},
		{Title: "Total %", Width: 8},
	}

	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
		table.WithHeight(10),
	)
	t.SetStyles(TableStyles())

	return &PortfolioModel{
		State: PortfolioStateLoading,
		Table: t,
	}
}

// SetHeight sets the table height.
func (m *PortfolioModel) SetHeight(height int) {
	m.Table.SetHeight(height)
}

// Update handles messages for the portfolio view.
func (m *PortfolioModel) Update(msg tea.Msg) (*PortfolioModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case PortfolioLoadedMsg:
		m.State = PortfolioStateLoaded
		m.Data = msg.Portfolio
		m.LastUpdated = time.Now()
		m.Err = nil
		m.updateTable()

	case PortfolioErrorMsg:
		m.State = PortfolioStateError
		m.Err = msg.Err

	case tea.KeyMsg:
		// Table navigation
		m.Table, cmd = m.Table.Update(msg)
		return m, cmd
	}

	// Pass other messages to table
	m.Table, cmd = m.Table.Update(msg)
	return m, cmd
}

// updateTable updates the table rows from portfolio data.
func (m *PortfolioModel) updateTable() {
	rows := make([]table.Row, 0, len(m.Data.Positions))
	for _, pos := range m.Data.Positions {
		totalGainValue := pos.CostBasis.GainValue
		totalGainPct := pos.CostBasis.GainPercentage
		if totalGainValue == "" {
			totalGainValue = "0"
			totalGainPct = "0"
		}
		rows = append(rows, table.Row{
			pos.Instrument.Symbol,
			pos.Quantity,
			"$" + pos.LastPrice.LastPrice,
			"$" + pos.CurrentValue,
			publicapi.FormatGainLoss(pos.PositionDailyGain.GainValue),
			pos.PositionDailyGain.GainPercentage + "%",
			publicapi.FormatGainLoss(totalGainValue),
			totalGainPct + "%",
		})
	}
	m.Table.SetRows(rows)
}

// View renders the portfolio view.
func (m *PortfolioModel) View() string {
	var b strings.Builder

	switch m.State {
	case PortfolioStateLoading:
		b.WriteString("Loading portfolio...")
		return b.String()

	case PortfolioStateError:
		b.WriteString(ErrorStyle.Render(fmt.Sprintf("Error: %v", m.Err)))
		b.WriteString("\n\nPress 'r' to retry")
		return b.String()

	case PortfolioStateLoaded:
		// Account Summary Section
		b.WriteString(SummaryStyle.Render("Account Summary"))
		b.WriteString("\n")

		p := m.Data

		// Calculate total value by summing all equity components
		var totalValueFloat float64
		var cashValue string
		for _, eq := range p.Equity {
			if val, err := strconv.ParseFloat(eq.Value, 64); err == nil {
				totalValueFloat += val
			}
			if eq.Type == "CASH" {
				cashValue = eq.Value
			}
		}
		totalValue := fmt.Sprintf("%.2f", totalValueFloat)

		// Calculate day P/L from positions
		var totalDayGain float64
		for _, pos := range p.Positions {
			if val, err := strconv.ParseFloat(pos.PositionDailyGain.GainValue, 64); err == nil {
				totalDayGain += val
			}
		}
		dayChange := fmt.Sprintf("%.2f", totalDayGain)

		b.WriteString(LabelStyle.Render("Total Value: "))
		b.WriteString(ValueStyle.Render("$" + totalValue))
		b.WriteString("  ")
		b.WriteString(LabelStyle.Render("Cash: "))
		b.WriteString(ValueStyle.Render("$" + cashValue))
		b.WriteString("  ")
		b.WriteString(LabelStyle.Render("Day P/L: "))
		if totalDayGain >= 0 {
			b.WriteString(GreenStyle.Render("+$" + dayChange))
		} else {
			b.WriteString(RedStyle.Render("-$" + dayChange[1:]))
		}
		b.WriteString("\n")

		b.WriteString(LabelStyle.Render("Buying Power: "))
		b.WriteString(ValueStyle.Render("$" + p.BuyingPower.BuyingPower))
		b.WriteString("  ")
		b.WriteString(LabelStyle.Render("Options BP: "))
		b.WriteString(ValueStyle.Render("$" + p.BuyingPower.OptionsBuyingPower))
		b.WriteString("\n\n")

		// Positions Table
		if len(p.Positions) == 0 {
			b.WriteString(LabelStyle.Render("No positions"))
		} else {
			b.WriteString(SummaryStyle.Render("Positions"))
			b.WriteString(LabelStyle.Render(fmt.Sprintf(" (%d)", len(p.Positions))))
			b.WriteString("\n")
			b.WriteString(m.Table.View())
		}

		// Last updated
		b.WriteString("\n")
		b.WriteString(LabelStyle.Render(fmt.Sprintf("Updated: %s", m.LastUpdated.Format("3:04:05 PM"))))
	}

	return b.String()
}

// FetchPortfolio returns a command that fetches portfolio data.
func FetchPortfolio(cfg *config.Config, store keyring.Store) tea.Cmd {
	return func() tea.Msg {
		if cfg.AccountUUID == "" {
			return PortfolioErrorMsg{Err: fmt.Errorf("no account configured")}
		}

		token, err := api.GetAuthToken(store, cfg.APIBaseURL, false)
		if err != nil {
			return PortfolioErrorMsg{Err: err}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		client := api.NewClient(cfg.APIBaseURL, token)
		path := fmt.Sprintf("/userapigateway/trading/%s/portfolio/v2", cfg.AccountUUID)
		resp, err := client.Get(ctx, path)
		if err != nil {
			return PortfolioErrorMsg{Err: fmt.Errorf("failed to fetch portfolio: %w", err)}
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			return PortfolioErrorMsg{Err: fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))}
		}

		var portfolio Portfolio
		if err := json.NewDecoder(resp.Body).Decode(&portfolio); err != nil {
			return PortfolioErrorMsg{Err: fmt.Errorf("failed to decode response: %w", err)}
		}

		return PortfolioLoadedMsg{Portfolio: portfolio}
	}
}
