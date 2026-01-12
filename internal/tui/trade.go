package tui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"

	"github.com/jonandersen/pub/internal/api"
	"github.com/jonandersen/pub/internal/config"
	"github.com/jonandersen/pub/internal/keyring"
)

// TradeState represents the current state of the trade view.
type TradeState int

const (
	TradeStateIdle TradeState = iota
	TradeStateFetchingQuote
	TradeStateReady
	TradeStateSubmitting
	TradeStateSuccess
	TradeStateError
)

// TradeMode represents the input mode of the trade view.
type TradeMode int

const (
	TradeModeForm TradeMode = iota
	TradeModeConfirm
)

// TradeSide represents buy or sell.
type TradeSide int

const (
	TradeSideBuy TradeSide = iota
	TradeSideSell
)

func (s TradeSide) String() string {
	if s == TradeSideBuy {
		return "BUY"
	}
	return "SELL"
}

// TradeOrderType represents market or limit order.
type TradeOrderType int

const (
	TradeOrderTypeMarket TradeOrderType = iota
	TradeOrderTypeLimit
)

func (t TradeOrderType) String() string {
	if t == TradeOrderTypeMarket {
		return "MARKET"
	}
	return "LIMIT"
}

// TradeField represents the currently focused input field.
type TradeField int

const (
	TradeFieldSymbol TradeField = iota
	TradeFieldSide
	TradeFieldOrderType
	TradeFieldQuantity
	TradeFieldLimitPrice
)

// TradeModel holds the state for the trade view.
type TradeModel struct {
	State     TradeState
	Mode      TradeMode
	Err       error
	LastError string

	// Form fields
	FocusedField    TradeField
	SymbolInput     textinput.Model
	QuantityInput   textinput.Model
	LimitPriceInput textinput.Model

	// Selections
	Side      TradeSide
	OrderType TradeOrderType

	// Quote data for the symbol
	Quote       *Quote
	QuoteLoaded bool

	// Order result
	OrderID     string
	OrderSymbol string
}

// NewTradeModel creates a new trade model.
func NewTradeModel() *TradeModel {
	symbolInput := textinput.New()
	symbolInput.Placeholder = "AAPL"
	symbolInput.CharLimit = 10
	symbolInput.Width = 15
	symbolInput.Focus()

	quantityInput := textinput.New()
	quantityInput.Placeholder = "0"
	quantityInput.CharLimit = 10
	quantityInput.Width = 15

	limitPriceInput := textinput.New()
	limitPriceInput.Placeholder = "0.00"
	limitPriceInput.CharLimit = 12
	limitPriceInput.Width = 15

	return &TradeModel{
		State:           TradeStateIdle,
		Mode:            TradeModeForm,
		FocusedField:    TradeFieldSymbol,
		SymbolInput:     symbolInput,
		QuantityInput:   quantityInput,
		LimitPriceInput: limitPriceInput,
		Side:            TradeSideBuy,
		OrderType:       TradeOrderTypeMarket,
	}
}

// SetSymbol sets the symbol to trade (called when jumping from watchlist).
func (m *TradeModel) SetSymbol(symbol string) {
	m.SymbolInput.SetValue(strings.ToUpper(symbol))
	m.FocusedField = TradeFieldQuantity
	m.SymbolInput.Blur()
	m.QuantityInput.Focus()
	// Reset quote - will need to fetch
	m.Quote = nil
	m.QuoteLoaded = false
	m.State = TradeStateIdle
}

// IsTextFieldFocused returns true if focus is on a text input field.
func (m *TradeModel) IsTextFieldFocused() bool {
	switch m.FocusedField {
	case TradeFieldSymbol, TradeFieldQuantity, TradeFieldLimitPrice:
		return true
	default:
		return false
	}
}

// Update handles messages for the trade view.
func (m *TradeModel) Update(msg tea.Msg, cfg *config.Config, store keyring.Store) (*TradeModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case TradeQuoteMsg:
		m.State = TradeStateReady
		m.Quote = &msg.Quote
		m.QuoteLoaded = true
		m.Err = nil
		return m, nil

	case TradeQuoteErrorMsg:
		m.State = TradeStateError
		m.Err = msg.Err
		m.QuoteLoaded = false
		return m, nil

	case TradeOrderPlacedMsg:
		m.State = TradeStateSuccess
		m.OrderID = msg.OrderID
		m.OrderSymbol = msg.Symbol
		m.Mode = TradeModeForm
		return m, nil

	case TradeOrderErrorMsg:
		m.State = TradeStateError
		m.Err = msg.Err
		m.Mode = TradeModeForm
		return m, nil

	case tea.KeyMsg:
		// Handle confirmation mode
		if m.Mode == TradeModeConfirm {
			switch msg.String() {
			case "y", "Y", "enter":
				m.State = TradeStateSubmitting
				return m, PlaceOrder(m, cfg, store)
			case "n", "N", "esc":
				m.Mode = TradeModeForm
				return m, nil
			}
			return m, nil
		}

		// Handle form mode
		switch msg.String() {
		case "esc":
			// Escape blurs text fields so number keys work for navigation
			if m.IsTextFieldFocused() {
				m.blurAll()
				m.FocusedField = TradeFieldSide
				return m, nil
			}
			return m, nil

		case "tab", "down":
			m.nextField()
			return m, textinput.Blink

		case "shift+tab", "up":
			m.prevField()
			return m, textinput.Blink

		case "enter":
			// If on symbol field and symbol is entered, fetch quote
			if m.FocusedField == TradeFieldSymbol && m.SymbolInput.Value() != "" {
				m.State = TradeStateFetchingQuote
				return m, FetchTradeQuote(m.SymbolInput.Value(), cfg, store)
			}
			// If form is valid, show confirmation
			if m.isFormValid() {
				m.Mode = TradeModeConfirm
			}
			return m, nil

		case "left", "right":
			// Toggle side or order type with arrow keys
			switch m.FocusedField {
			case TradeFieldSide:
				if m.Side == TradeSideBuy {
					m.Side = TradeSideSell
				} else {
					m.Side = TradeSideBuy
				}
			case TradeFieldOrderType:
				if m.OrderType == TradeOrderTypeMarket {
					m.OrderType = TradeOrderTypeLimit
				} else {
					m.OrderType = TradeOrderTypeMarket
				}
			}
			return m, nil

		case " ":
			// Space toggles side or order type
			switch m.FocusedField {
			case TradeFieldSide:
				if m.Side == TradeSideBuy {
					m.Side = TradeSideSell
				} else {
					m.Side = TradeSideBuy
				}
				return m, nil
			case TradeFieldOrderType:
				if m.OrderType == TradeOrderTypeMarket {
					m.OrderType = TradeOrderTypeLimit
				} else {
					m.OrderType = TradeOrderTypeMarket
				}
				return m, nil
			}

		case "ctrl+n":
			// Reset form for new order
			m.reset()
			return m, nil
		}

		// Pass to focused text input
		switch m.FocusedField {
		case TradeFieldSymbol:
			m.SymbolInput, cmd = m.SymbolInput.Update(msg)
			cmds = append(cmds, cmd)
			// Clear quote when symbol changes
			if m.QuoteLoaded {
				m.Quote = nil
				m.QuoteLoaded = false
				m.State = TradeStateIdle
			}
		case TradeFieldQuantity:
			m.QuantityInput, cmd = m.QuantityInput.Update(msg)
			cmds = append(cmds, cmd)
		case TradeFieldLimitPrice:
			m.LimitPriceInput, cmd = m.LimitPriceInput.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// nextField moves focus to the next field.
func (m *TradeModel) nextField() {
	m.blurAll()

	switch m.FocusedField {
	case TradeFieldSymbol:
		m.FocusedField = TradeFieldSide
	case TradeFieldSide:
		m.FocusedField = TradeFieldOrderType
	case TradeFieldOrderType:
		m.FocusedField = TradeFieldQuantity
	case TradeFieldQuantity:
		if m.OrderType == TradeOrderTypeLimit {
			m.FocusedField = TradeFieldLimitPrice
		} else {
			m.FocusedField = TradeFieldSymbol
		}
	case TradeFieldLimitPrice:
		m.FocusedField = TradeFieldSymbol
	}

	m.focusCurrent()
}

// prevField moves focus to the previous field.
func (m *TradeModel) prevField() {
	m.blurAll()

	switch m.FocusedField {
	case TradeFieldSymbol:
		if m.OrderType == TradeOrderTypeLimit {
			m.FocusedField = TradeFieldLimitPrice
		} else {
			m.FocusedField = TradeFieldQuantity
		}
	case TradeFieldSide:
		m.FocusedField = TradeFieldSymbol
	case TradeFieldOrderType:
		m.FocusedField = TradeFieldSide
	case TradeFieldQuantity:
		m.FocusedField = TradeFieldOrderType
	case TradeFieldLimitPrice:
		m.FocusedField = TradeFieldQuantity
	}

	m.focusCurrent()
}

func (m *TradeModel) blurAll() {
	m.SymbolInput.Blur()
	m.QuantityInput.Blur()
	m.LimitPriceInput.Blur()
}

func (m *TradeModel) focusCurrent() {
	switch m.FocusedField {
	case TradeFieldSymbol:
		m.SymbolInput.Focus()
	case TradeFieldQuantity:
		m.QuantityInput.Focus()
	case TradeFieldLimitPrice:
		m.LimitPriceInput.Focus()
	}
}

func (m *TradeModel) reset() {
	m.State = TradeStateIdle
	m.Mode = TradeModeForm
	m.Err = nil
	m.SymbolInput.SetValue("")
	m.QuantityInput.SetValue("")
	m.LimitPriceInput.SetValue("")
	m.Side = TradeSideBuy
	m.OrderType = TradeOrderTypeMarket
	m.Quote = nil
	m.QuoteLoaded = false
	m.OrderID = ""
	m.OrderSymbol = ""
	m.FocusedField = TradeFieldSymbol
	m.blurAll()
	m.SymbolInput.Focus()
}

func (m *TradeModel) isFormValid() bool {
	symbol := strings.TrimSpace(m.SymbolInput.Value())
	if symbol == "" {
		return false
	}

	qty := strings.TrimSpace(m.QuantityInput.Value())
	if qty == "" {
		return false
	}
	qtyVal, err := strconv.ParseFloat(qty, 64)
	if err != nil || qtyVal <= 0 {
		return false
	}

	if m.OrderType == TradeOrderTypeLimit {
		price := strings.TrimSpace(m.LimitPriceInput.Value())
		if price == "" {
			return false
		}
		priceVal, err := strconv.ParseFloat(price, 64)
		if err != nil || priceVal <= 0 {
			return false
		}
	}

	return true
}

// estimatedCost calculates the estimated cost of the order.
func (m *TradeModel) estimatedCost() string {
	qty := strings.TrimSpace(m.QuantityInput.Value())
	if qty == "" {
		return "-"
	}
	qtyVal, err := strconv.ParseFloat(qty, 64)
	if err != nil || qtyVal <= 0 {
		return "-"
	}

	var price float64
	if m.OrderType == TradeOrderTypeLimit {
		priceStr := strings.TrimSpace(m.LimitPriceInput.Value())
		if priceStr == "" {
			return "-"
		}
		price, err = strconv.ParseFloat(priceStr, 64)
		if err != nil {
			return "-"
		}
	} else if m.Quote != nil && m.Quote.Last != "" {
		price, err = strconv.ParseFloat(m.Quote.Last, 64)
		if err != nil {
			return "-"
		}
	} else {
		return "-"
	}

	cost := qtyVal * price
	return fmt.Sprintf("$%.2f", cost)
}

// View renders the trade view.
func (m *TradeModel) View() string {
	var b strings.Builder

	// Show success message
	if m.State == TradeStateSuccess {
		b.WriteString(GreenStyle.Render("Order Placed Successfully!"))
		b.WriteString("\n\n")
		b.WriteString(LabelStyle.Render("Order ID: "))
		b.WriteString(ValueStyle.Render(m.OrderID))
		b.WriteString("\n")
		b.WriteString(LabelStyle.Render("Symbol:   "))
		b.WriteString(ValueStyle.Render(m.OrderSymbol))
		b.WriteString("\n\n")
		b.WriteString(LabelStyle.Render("Press Ctrl+N to place another order"))
		return b.String()
	}

	// Show confirmation dialog
	if m.Mode == TradeModeConfirm {
		return m.renderConfirmation()
	}

	// Show form
	b.WriteString(SummaryStyle.Render("Place Order"))
	b.WriteString("\n\n")

	// Error message
	if m.State == TradeStateError && m.Err != nil {
		b.WriteString(ErrorStyle.Render(fmt.Sprintf("Error: %v", m.Err)))
		b.WriteString("\n\n")
	}

	// Symbol input
	symbolStyle := LabelStyle
	if m.FocusedField == TradeFieldSymbol {
		symbolStyle = ValueStyle
	}
	b.WriteString(symbolStyle.Render("Symbol:     "))
	b.WriteString(m.renderTextInput(m.SymbolInput, m.FocusedField == TradeFieldSymbol))
	if m.State == TradeStateFetchingQuote {
		b.WriteString(LabelStyle.Render("  (fetching quote...)"))
	} else if m.QuoteLoaded && m.Quote != nil {
		b.WriteString(LabelStyle.Render(fmt.Sprintf("  $%s", m.Quote.Last)))
	}
	b.WriteString("\n\n")

	// Side toggle (BUY/SELL)
	sideStyle := LabelStyle
	if m.FocusedField == TradeFieldSide {
		sideStyle = ValueStyle
	}
	b.WriteString(sideStyle.Render("Side:       "))
	b.WriteString(m.renderToggle([]string{"BUY", "SELL"}, int(m.Side), m.FocusedField == TradeFieldSide))
	b.WriteString("\n\n")

	// Order type toggle (MARKET/LIMIT)
	typeStyle := LabelStyle
	if m.FocusedField == TradeFieldOrderType {
		typeStyle = ValueStyle
	}
	b.WriteString(typeStyle.Render("Type:       "))
	b.WriteString(m.renderToggle([]string{"MARKET", "LIMIT"}, int(m.OrderType), m.FocusedField == TradeFieldOrderType))
	b.WriteString("\n\n")

	// Quantity input
	qtyStyle := LabelStyle
	if m.FocusedField == TradeFieldQuantity {
		qtyStyle = ValueStyle
	}
	b.WriteString(qtyStyle.Render("Quantity:   "))
	b.WriteString(m.renderTextInput(m.QuantityInput, m.FocusedField == TradeFieldQuantity))
	b.WriteString("\n\n")

	// Limit price input (only for limit orders)
	if m.OrderType == TradeOrderTypeLimit {
		priceStyle := LabelStyle
		if m.FocusedField == TradeFieldLimitPrice {
			priceStyle = ValueStyle
		}
		b.WriteString(priceStyle.Render("Limit Price: "))
		b.WriteString(m.renderTextInput(m.LimitPriceInput, m.FocusedField == TradeFieldLimitPrice))
		b.WriteString("\n\n")
	}

	// Estimated cost
	b.WriteString(LabelStyle.Render("Est. Cost:  "))
	cost := m.estimatedCost()
	if cost != "-" {
		b.WriteString(ValueStyle.Render(cost))
	} else {
		b.WriteString(LabelStyle.Render(cost))
	}
	b.WriteString("\n\n")

	// Submit hint
	if m.isFormValid() {
		b.WriteString(KeyStyle.Render("Enter"))
		b.WriteString(LabelStyle.Render(" to review order"))
	} else {
		b.WriteString(LabelStyle.Render("Fill in all fields to place order"))
	}

	return b.String()
}

func (m *TradeModel) renderTextInput(input textinput.Model, focused bool) string {
	style := lipgloss.NewStyle()
	if focused {
		style = style.
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(0, 1)
	} else {
		style = style.
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorMuted).
			Padding(0, 1)
	}
	return style.Render(input.View())
}

func (m *TradeModel) renderToggle(options []string, selected int, focused bool) string {
	var parts []string
	for i, opt := range options {
		style := lipgloss.NewStyle().Padding(0, 1)
		if i == selected {
			switch opt {
			case "BUY":
				style = style.Background(ColorGreen).Foreground(lipgloss.Color("0")).Bold(true)
			case "SELL":
				style = style.Background(ColorRed).Foreground(lipgloss.Color("0")).Bold(true)
			default:
				style = style.Background(ColorPrimary).Foreground(lipgloss.Color("0")).Bold(true)
			}
		} else {
			style = style.Foreground(ColorMuted)
		}
		parts = append(parts, style.Render(opt))
	}

	content := strings.Join(parts, " ")
	if focused {
		return lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Render(content)
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorMuted).
		Render(content)
}

func (m *TradeModel) renderConfirmation() string {
	var b strings.Builder

	b.WriteString(WarningStyle.Render("Confirm Order"))
	b.WriteString("\n\n")

	symbol := strings.ToUpper(m.SymbolInput.Value())
	qty := m.QuantityInput.Value()

	// Side with color
	sideStr := m.Side.String()
	if m.Side == TradeSideBuy {
		b.WriteString(GreenStyle.Render(sideStr))
	} else {
		b.WriteString(RedStyle.Render(sideStr))
	}
	b.WriteString(" ")
	b.WriteString(ValueStyle.Render(qty))
	b.WriteString(" shares of ")
	b.WriteString(ValueStyle.Render(symbol))
	b.WriteString("\n\n")

	b.WriteString(LabelStyle.Render("Order Type: "))
	b.WriteString(ValueStyle.Render(m.OrderType.String()))
	b.WriteString("\n")

	if m.OrderType == TradeOrderTypeLimit {
		b.WriteString(LabelStyle.Render("Limit Price: "))
		b.WriteString(ValueStyle.Render("$" + m.LimitPriceInput.Value()))
		b.WriteString("\n")
	}

	b.WriteString(LabelStyle.Render("Est. Cost:  "))
	b.WriteString(ValueStyle.Render(m.estimatedCost()))
	b.WriteString("\n\n")

	if m.State == TradeStateSubmitting {
		b.WriteString(LabelStyle.Render("Submitting order..."))
	} else {
		b.WriteString(KeyStyle.Render("Y"))
		b.WriteString(LabelStyle.Render(" to confirm, "))
		b.WriteString(KeyStyle.Render("N"))
		b.WriteString(LabelStyle.Render(" to cancel"))
	}

	return b.String()
}

// Message types for trade operations

// TradeQuoteMsg is sent when a quote is fetched for the trade form.
type TradeQuoteMsg struct {
	Quote Quote
}

// TradeQuoteErrorMsg is sent when fetching a quote fails.
type TradeQuoteErrorMsg struct {
	Err error
}

// TradeOrderPlacedMsg is sent when an order is placed successfully.
type TradeOrderPlacedMsg struct {
	OrderID string
	Symbol  string
}

// TradeOrderErrorMsg is sent when placing an order fails.
type TradeOrderErrorMsg struct {
	Err error
}

// FetchTradeQuote returns a command that fetches a quote for the trade form.
func FetchTradeQuote(symbol string, cfg *config.Config, store keyring.Store) tea.Cmd {
	return func() tea.Msg {
		if cfg.AccountUUID == "" {
			return TradeQuoteErrorMsg{Err: fmt.Errorf("no account configured")}
		}

		token, err := getAuthToken(store, cfg.APIBaseURL)
		if err != nil {
			return TradeQuoteErrorMsg{Err: err}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		reqBody := QuoteRequest{
			Instruments: []QuoteInstrument{
				{Symbol: strings.ToUpper(symbol), Type: "EQUITY"},
			},
		}
		body, err := json.Marshal(reqBody)
		if err != nil {
			return TradeQuoteErrorMsg{Err: fmt.Errorf("failed to encode request: %w", err)}
		}

		client := api.NewClient(cfg.APIBaseURL, token)
		path := fmt.Sprintf("/userapigateway/marketdata/%s/quotes", cfg.AccountUUID)
		resp, err := client.Post(ctx, path, bytes.NewReader(body))
		if err != nil {
			return TradeQuoteErrorMsg{Err: fmt.Errorf("failed to fetch quote: %w", err)}
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != 200 {
			respBody, _ := io.ReadAll(resp.Body)
			return TradeQuoteErrorMsg{Err: fmt.Errorf("API error: %d - %s", resp.StatusCode, string(respBody))}
		}

		var quotesResp QuotesResponse
		if err := json.NewDecoder(resp.Body).Decode(&quotesResp); err != nil {
			return TradeQuoteErrorMsg{Err: fmt.Errorf("failed to decode response: %w", err)}
		}

		if len(quotesResp.Quotes) == 0 {
			return TradeQuoteErrorMsg{Err: fmt.Errorf("no quote found for %s", symbol)}
		}

		quote := quotesResp.Quotes[0]
		if quote.Outcome != "SUCCESS" {
			return TradeQuoteErrorMsg{Err: fmt.Errorf("invalid symbol: %s", symbol)}
		}

		return TradeQuoteMsg{Quote: quote}
	}
}

// PlaceOrder returns a command that places an order.
func PlaceOrder(m *TradeModel, cfg *config.Config, store keyring.Store) tea.Cmd {
	return func() tea.Msg {
		if cfg.AccountUUID == "" {
			return TradeOrderErrorMsg{Err: fmt.Errorf("no account configured")}
		}

		if !cfg.TradingEnabled {
			return TradeOrderErrorMsg{Err: fmt.Errorf("trading is disabled - enable in config")}
		}

		token, err := getAuthToken(store, cfg.APIBaseURL)
		if err != nil {
			return TradeOrderErrorMsg{Err: err}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		symbol := strings.ToUpper(strings.TrimSpace(m.SymbolInput.Value()))
		orderID := uuid.New().String()

		orderReq := map[string]any{
			"orderId": orderID,
			"instrument": map[string]string{
				"symbol": symbol,
				"type":   "EQUITY",
			},
			"orderSide": m.Side.String(),
			"orderType": m.OrderType.String(),
			"expiration": map[string]string{
				"timeInForce": "DAY",
			},
			"quantity": m.QuantityInput.Value(),
		}

		if m.OrderType == TradeOrderTypeLimit {
			orderReq["limitPrice"] = m.LimitPriceInput.Value()
		}

		body, err := json.Marshal(orderReq)
		if err != nil {
			return TradeOrderErrorMsg{Err: fmt.Errorf("failed to encode request: %w", err)}
		}

		client := api.NewClient(cfg.APIBaseURL, token)
		path := fmt.Sprintf("/userapigateway/trading/%s/order", cfg.AccountUUID)
		resp, err := client.Post(ctx, path, bytes.NewReader(body))
		if err != nil {
			return TradeOrderErrorMsg{Err: fmt.Errorf("failed to place order: %w", err)}
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != 200 {
			respBody, _ := io.ReadAll(resp.Body)
			return TradeOrderErrorMsg{Err: fmt.Errorf("API error: %d - %s", resp.StatusCode, string(respBody))}
		}

		return TradeOrderPlacedMsg{OrderID: orderID, Symbol: symbol}
	}
}
