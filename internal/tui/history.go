package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/jonandersen/public-cli/internal/api"
	"github.com/jonandersen/public-cli/internal/config"
	"github.com/jonandersen/public-cli/internal/keyring"
)

// HistoryState represents the loading state of history data.
type HistoryState int

const (
	HistoryStateLoading HistoryState = iota
	HistoryStateLoaded
	HistoryStateError
)

// HistoryModel holds the state for the history view.
type HistoryModel struct {
	State        HistoryState
	Transactions []Transaction
	Err          error
	LastUpdated  time.Time
	Table        table.Model

	// Pagination
	NextToken   string
	HasMore     bool
	LoadingMore bool

	// Detail panel
	ShowDetail  bool
	DetailIndex int
}

// NewHistoryModel creates a new history model.
func NewHistoryModel() *HistoryModel {
	cols := []table.Column{
		{Title: "Date", Width: 12},
		{Title: "Type", Width: 12},
		{Title: "Symbol", Width: 10},
		{Title: "Side", Width: 6},
		{Title: "Qty", Width: 8},
		{Title: "Amount", Width: 12},
		{Title: "Description", Width: 30},
	}

	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
		table.WithHeight(10),
	)
	t.SetStyles(TableStyles())

	return &HistoryModel{
		State:        HistoryStateLoading,
		Transactions: []Transaction{},
		Table:        t,
	}
}

// SetHeight sets the table height.
func (m *HistoryModel) SetHeight(height int) {
	m.Table.SetHeight(height)
}

// Update handles messages for the history view.
// Returns the model, command, and whether the event was handled.
func (m *HistoryModel) Update(msg tea.Msg) (*HistoryModel, tea.Cmd, bool) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case HistoryLoadedMsg:
		if m.LoadingMore {
			// Append to existing transactions
			m.Transactions = append(m.Transactions, msg.Transactions...)
			m.LoadingMore = false
		} else {
			m.Transactions = msg.Transactions
		}
		m.State = HistoryStateLoaded
		m.LastUpdated = time.Now()
		m.Err = nil
		m.NextToken = msg.NextToken
		m.HasMore = msg.NextToken != ""
		m.updateTable()
		return m, nil, true

	case HistoryErrorMsg:
		m.State = HistoryStateError
		m.Err = msg.Err
		m.LoadingMore = false
		return m, nil, true

	case tea.KeyMsg:
		// Handle detail view toggle
		if m.ShowDetail {
			switch msg.String() {
			case "esc", "enter", "q":
				m.ShowDetail = false
				return m, nil, true
			case "up", "k":
				if m.DetailIndex > 0 {
					m.DetailIndex--
				}
				return m, nil, true
			case "down", "j":
				if m.DetailIndex < len(m.Transactions)-1 {
					m.DetailIndex++
				}
				return m, nil, true
			}
			return m, nil, true
		}

		// Normal mode
		switch msg.String() {
		case "enter":
			// Show detail for selected transaction
			if len(m.Transactions) > 0 {
				m.DetailIndex = m.Table.Cursor()
				m.ShowDetail = true
			}
			return m, nil, true
		}
	}

	// Pass to table for navigation
	m.Table, cmd = m.Table.Update(msg)
	return m, cmd, false
}

// updateTable updates the table rows from history data.
func (m *HistoryModel) updateTable() {
	rows := make([]table.Row, 0, len(m.Transactions))
	for _, txn := range m.Transactions {
		// Format timestamp
		date := formatHistoryDate(txn.Timestamp)

		// Use subtype if available, otherwise type
		txnType := txn.Type
		if txn.SubType != "" {
			txnType = txn.SubType
		}

		// Format amount
		amount := formatHistoryAmount(txn.NetAmount)

		// Truncate description
		desc := txn.Description
		if len(desc) > 30 {
			desc = desc[:27] + "..."
		}

		rows = append(rows, table.Row{
			date,
			txnType,
			txn.Symbol,
			txn.Side,
			txn.Quantity,
			amount,
			desc,
		})
	}
	m.Table.SetRows(rows)
}

// View renders the history view.
func (m *HistoryModel) View() string {
	var b strings.Builder

	// Show detail panel if active
	if m.ShowDetail && m.DetailIndex < len(m.Transactions) {
		return m.renderDetail()
	}

	switch m.State {
	case HistoryStateLoading:
		b.WriteString("Loading history...")
		return b.String()

	case HistoryStateError:
		b.WriteString(ErrorStyle.Render(fmt.Sprintf("Error: %v", m.Err)))
		b.WriteString("\n\nPress 'r' to retry")
		return b.String()

	case HistoryStateLoaded:
		b.WriteString(SummaryStyle.Render("Transaction History"))
		b.WriteString(LabelStyle.Render(fmt.Sprintf(" (%d)", len(m.Transactions))))
		b.WriteString("\n\n")

		if len(m.Transactions) == 0 {
			b.WriteString(LabelStyle.Render("No transactions found"))
		} else {
			b.WriteString(m.Table.View())

			// Show pagination status
			b.WriteString("\n")
			if m.HasMore {
				if m.LoadingMore {
					b.WriteString(LabelStyle.Render("Loading more..."))
				} else {
					b.WriteString(LabelStyle.Render("More transactions available"))
				}
				b.WriteString("  ")
			}
			b.WriteString(LabelStyle.Render(fmt.Sprintf("Updated: %s", m.LastUpdated.Format("3:04:05 PM"))))
		}
	}

	return b.String()
}

// renderDetail renders the detail panel for a selected transaction.
func (m *HistoryModel) renderDetail() string {
	var b strings.Builder
	txn := m.Transactions[m.DetailIndex]

	b.WriteString(SummaryStyle.Render("Transaction Details"))
	b.WriteString("\n\n")

	// Format timestamp
	date := txn.Timestamp
	if t, err := time.Parse(time.RFC3339, txn.Timestamp); err == nil {
		date = t.Format("Jan 2, 2006 3:04:05 PM")
	}

	// Transaction type
	txnType := txn.Type
	if txn.SubType != "" {
		txnType = fmt.Sprintf("%s (%s)", txn.Type, txn.SubType)
	}

	// Build detail rows
	rows := []struct {
		label string
		value string
	}{
		{"ID", txn.ID},
		{"Date", date},
		{"Type", txnType},
		{"Symbol", txn.Symbol},
		{"Security Type", txn.SecurityType},
		{"Side", txn.Side},
		{"Quantity", txn.Quantity},
		{"Net Amount", formatHistoryAmount(txn.NetAmount)},
		{"Principal", formatHistoryAmount(txn.PrincipalAmount)},
		{"Fees", formatHistoryAmount(txn.Fees)},
		{"Direction", txn.Direction},
		{"Description", txn.Description},
	}

	for _, row := range rows {
		if row.value == "" {
			continue
		}
		b.WriteString(LabelStyle.Render(fmt.Sprintf("%-15s", row.label+":")))
		b.WriteString(" ")
		// Color amounts
		if row.label == "Net Amount" || row.label == "Principal" {
			if len(row.value) > 0 && row.value[0] == '-' {
				b.WriteString(RedStyle.Render(row.value))
			} else {
				b.WriteString(GreenStyle.Render(row.value))
			}
		} else {
			b.WriteString(ValueStyle.Render(row.value))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(LabelStyle.Render("Press Enter or Esc to close"))

	return b.String()
}

// FooterKeys returns the footer keys for the history view.
func (m *HistoryModel) FooterKeys(keys []struct{ key, desc string }) []struct{ key, desc string } {
	if m.ShowDetail {
		return []struct{ key, desc string }{
			{"↑/↓", "prev/next"},
			{"enter/esc", "close"},
		}
	}
	keys = append(keys, struct{ key, desc string }{"↑/↓", "navigate"})
	keys = append(keys, struct{ key, desc string }{"enter", "details"})
	keys = append(keys, struct{ key, desc string }{"esc", "toolbar"})
	keys = append(keys, struct{ key, desc string }{"r", "refresh"})
	return keys
}

// SelectedTransaction returns the currently selected transaction, if any.
func (m *HistoryModel) SelectedTransaction() *Transaction {
	if len(m.Transactions) == 0 {
		return nil
	}
	idx := m.Table.Cursor()
	if idx >= 0 && idx < len(m.Transactions) {
		return &m.Transactions[idx]
	}
	return nil
}

// FetchHistory returns a command that fetches transaction history.
func FetchHistory(cfg *config.Config, store keyring.Store) tea.Cmd {
	return FetchHistoryWithToken(cfg, store, "")
}

// FetchHistoryWithToken returns a command that fetches history with optional pagination token.
func FetchHistoryWithToken(cfg *config.Config, store keyring.Store, nextToken string) tea.Cmd {
	return func() tea.Msg {
		if cfg.AccountUUID == "" {
			return HistoryErrorMsg{Err: fmt.Errorf("no account configured")}
		}

		token, err := getAuthToken(store, cfg.APIBaseURL, false)
		if err != nil {
			return HistoryErrorMsg{Err: err}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		client := api.NewClient(cfg.APIBaseURL, token)
		path := fmt.Sprintf("/userapigateway/trading/%s/history", cfg.AccountUUID)

		// Build query parameters
		queryParams := make(map[string]string)
		queryParams["pageSize"] = "50" // Default page size
		if nextToken != "" {
			queryParams["nextToken"] = nextToken
		}

		resp, err := client.GetWithParams(ctx, path, queryParams)
		if err != nil {
			return HistoryErrorMsg{Err: fmt.Errorf("failed to fetch history: %w", err)}
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			return HistoryErrorMsg{Err: fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))}
		}

		var historyResp HistoryResponse
		if err := json.NewDecoder(resp.Body).Decode(&historyResp); err != nil {
			return HistoryErrorMsg{Err: fmt.Errorf("failed to decode response: %w", err)}
		}

		return HistoryLoadedMsg{
			Transactions: historyResp.Transactions,
			NextToken:    historyResp.NextToken,
		}
	}
}

// formatHistoryDate formats an ISO timestamp to a readable date.
func formatHistoryDate(timestamp string) string {
	t, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return timestamp
	}
	return t.Format("01/02 15:04")
}

// formatHistoryAmount formats an amount string with currency symbol.
func formatHistoryAmount(amount string) string {
	if amount == "" {
		return ""
	}
	if amount[0] == '-' {
		return "-$" + amount[1:]
	}
	return "$" + amount
}
