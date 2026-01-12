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

	"github.com/jonandersen/pub/internal/api"
	"github.com/jonandersen/pub/internal/config"
	"github.com/jonandersen/pub/internal/keyring"
)

// OrdersState represents the loading state of orders data.
type OrdersState int

const (
	OrdersStateLoading OrdersState = iota
	OrdersStateLoaded
	OrdersStateError
)

// OrdersMode represents the input mode of the orders view.
type OrdersMode int

const (
	OrdersModeNormal OrdersMode = iota
	OrdersModeCanceling
)

// OrdersModel holds the state for the orders view.
type OrdersModel struct {
	State         OrdersState
	Orders        []Order
	Err           error
	LastUpdated   time.Time
	Table         table.Model
	Mode          OrdersMode
	CancelOrderID string
	CancelSymbol  string
}

// NewOrdersModel creates a new orders model.
func NewOrdersModel() *OrdersModel {
	cols := []table.Column{
		{Title: "Symbol", Width: 8},
		{Title: "Side", Width: 5},
		{Title: "Type", Width: 8},
		{Title: "Status", Width: 12},
		{Title: "Qty", Width: 6},
		{Title: "Filled", Width: 6},
		{Title: "Price", Width: 10},
		{Title: "Created", Width: 12},
	}

	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
		table.WithHeight(10),
	)
	t.SetStyles(TableStyles())

	return &OrdersModel{
		State:  OrdersStateLoading,
		Orders: []Order{},
		Table:  t,
		Mode:   OrdersModeNormal,
	}
}

// SetHeight sets the table height.
func (m *OrdersModel) SetHeight(height int) {
	m.Table.SetHeight(height)
}

// Update handles messages for the orders view.
// Returns the model, command, and whether the event was handled.
func (m *OrdersModel) Update(msg tea.Msg, cfg *config.Config, store keyring.Store) (*OrdersModel, tea.Cmd, bool) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case OrdersLoadedMsg:
		m.State = OrdersStateLoaded
		m.Orders = msg.Orders
		m.LastUpdated = time.Now()
		m.Err = nil
		m.updateTable()
		return m, nil, true

	case OrdersErrorMsg:
		m.State = OrdersStateError
		m.Err = msg.Err
		return m, nil, true

	case OrderCancelledMsg:
		// Remove the cancelled order from our list
		newOrders := make([]Order, 0, len(m.Orders))
		for _, o := range m.Orders {
			if o.OrderID != msg.OrderID {
				newOrders = append(newOrders, o)
			}
		}
		m.Orders = newOrders
		m.updateTable()
		m.Mode = OrdersModeNormal
		m.CancelOrderID = ""
		m.CancelSymbol = ""
		// Refresh to get latest status
		return m, FetchOrders(cfg, store), true

	case OrderCancelErrorMsg:
		m.Err = msg.Err
		m.Mode = OrdersModeNormal
		m.CancelOrderID = ""
		m.CancelSymbol = ""
		return m, nil, true

	case tea.KeyMsg:
		switch m.Mode {
		case OrdersModeCanceling:
			switch msg.String() {
			case "y", "Y":
				// Cancel the order
				cmd = CancelOrder(m.CancelOrderID, cfg, store)
				return m, cmd, true
			case "n", "N", "esc":
				m.Mode = OrdersModeNormal
				m.CancelOrderID = ""
				m.CancelSymbol = ""
				return m, nil, true
			}
			return m, nil, true

		case OrdersModeNormal:
			switch msg.String() {
			case "c", "x", "d":
				// Cancel selected order
				if len(m.Orders) > 0 {
					idx := m.Table.Cursor()
					if idx >= 0 && idx < len(m.Orders) {
						order := m.Orders[idx]
						if order.Status == "NEW" || order.Status == "PARTIALLY_FILLED" || order.Status == "PENDING" {
							m.CancelOrderID = order.OrderID
							m.CancelSymbol = order.Instrument.Symbol
							m.Mode = OrdersModeCanceling
						}
					}
				}
				return m, nil, true
			}
		}
	}

	// Pass to table in normal mode
	if m.Mode == OrdersModeNormal {
		m.Table, cmd = m.Table.Update(msg)
		return m, cmd, false
	}

	return m, nil, false
}

// updateTable updates the table rows from orders data.
func (m *OrdersModel) updateTable() {
	rows := make([]table.Row, 0, len(m.Orders))
	for _, order := range m.Orders {
		price := "-"
		if order.LimitPrice != "" {
			price = "$" + order.LimitPrice
		} else if order.StopPrice != "" {
			price = "$" + order.StopPrice
		}

		// Format created time
		created := order.CreatedAt
		if t, err := time.Parse(time.RFC3339, order.CreatedAt); err == nil {
			created = t.Format("01/02 15:04")
		}

		rows = append(rows, table.Row{
			order.Instrument.Symbol,
			order.Side,
			order.Type,
			order.Status,
			order.Quantity,
			order.FilledQuantity,
			price,
			created,
		})
	}
	m.Table.SetRows(rows)
}

// View renders the orders view.
func (m *OrdersModel) View() string {
	var b strings.Builder

	// Handle cancel confirmation mode
	if m.Mode == OrdersModeCanceling {
		b.WriteString(WarningStyle.Render(fmt.Sprintf("Cancel order for %s?", m.CancelSymbol)))
		b.WriteString("\n\n")
		b.WriteString(LabelStyle.Render("Order ID: "))
		b.WriteString(ValueStyle.Render(m.CancelOrderID[:8] + "..."))
		b.WriteString("\n\n")
		b.WriteString(LabelStyle.Render("Press Y to confirm, N to cancel"))
		return b.String()
	}

	// Normal mode - show orders
	switch m.State {
	case OrdersStateLoading:
		b.WriteString("Loading orders...")
		return b.String()

	case OrdersStateError:
		b.WriteString(ErrorStyle.Render(fmt.Sprintf("Error: %v", m.Err)))
		b.WriteString("\n\nPress 'r' to retry")
		return b.String()

	case OrdersStateLoaded:
		b.WriteString(SummaryStyle.Render("Open Orders"))
		b.WriteString(LabelStyle.Render(fmt.Sprintf(" (%d)", len(m.Orders))))
		b.WriteString("\n\n")

		if len(m.Orders) == 0 {
			b.WriteString(LabelStyle.Render("No open orders"))
			b.WriteString("\n\n")
			b.WriteString(LabelStyle.Render("Place orders using the Trade view [4]"))
		} else {
			b.WriteString(m.Table.View())
			b.WriteString("\n")
			b.WriteString(LabelStyle.Render(fmt.Sprintf("Updated: %s", m.LastUpdated.Format("3:04:05 PM"))))
		}
	}

	return b.String()
}

// SelectedOrder returns the currently selected order, if any.
func (m *OrdersModel) SelectedOrder() *Order {
	if m.Mode != OrdersModeNormal || len(m.Orders) == 0 {
		return nil
	}
	idx := m.Table.Cursor()
	if idx >= 0 && idx < len(m.Orders) {
		return &m.Orders[idx]
	}
	return nil
}

// FetchOrders returns a command that fetches open orders.
func FetchOrders(cfg *config.Config, store keyring.Store) tea.Cmd {
	return func() tea.Msg {
		if cfg.AccountUUID == "" {
			return OrdersErrorMsg{Err: fmt.Errorf("no account configured")}
		}

		token, err := getAuthToken(store, cfg.APIBaseURL, false)
		if err != nil {
			return OrdersErrorMsg{Err: err}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		client := api.NewClient(cfg.APIBaseURL, token)
		path := fmt.Sprintf("/userapigateway/trading/%s/portfolio/v2", cfg.AccountUUID)
		resp, err := client.Get(ctx, path)
		if err != nil {
			return OrdersErrorMsg{Err: fmt.Errorf("failed to fetch orders: %w", err)}
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			return OrdersErrorMsg{Err: fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))}
		}

		var ordersResp OrdersResponse
		if err := json.NewDecoder(resp.Body).Decode(&ordersResp); err != nil {
			return OrdersErrorMsg{Err: fmt.Errorf("failed to decode response: %w", err)}
		}

		return OrdersLoadedMsg(ordersResp)
	}
}

// CancelOrder returns a command that cancels an order.
func CancelOrder(orderID string, cfg *config.Config, store keyring.Store) tea.Cmd {
	return func() tea.Msg {
		if cfg.AccountUUID == "" {
			return OrderCancelErrorMsg{Err: fmt.Errorf("no account configured")}
		}

		token, err := getAuthToken(store, cfg.APIBaseURL, false)
		if err != nil {
			return OrderCancelErrorMsg{Err: err}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		client := api.NewClient(cfg.APIBaseURL, token)
		path := fmt.Sprintf("/userapigateway/trading/%s/order/%s", cfg.AccountUUID, orderID)
		resp, err := client.Delete(ctx, path)
		if err != nil {
			return OrderCancelErrorMsg{Err: fmt.Errorf("failed to cancel order: %w", err)}
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			return OrderCancelErrorMsg{Err: fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))}
		}

		return OrderCancelledMsg{OrderID: orderID}
	}
}
