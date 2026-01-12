package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/jonandersen/pub/internal/config"
	"github.com/jonandersen/pub/internal/keyring"
	"github.com/jonandersen/pub/internal/tui"
)

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
  ↑/↓     Navigate positions
  r       Refresh data
  q/esc   Quit the application`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load CLI config
			cfg, err := config.Load(config.ConfigPath())
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Load TUI config
			uiCfg, err := tui.LoadConfig()
			if err != nil {
				return fmt.Errorf("failed to load UI config: %w", err)
			}

			// Create keyring store
			store := keyring.NewEnvStore(keyring.NewSystemStore())

			p := tea.NewProgram(tui.New(cfg, uiCfg, store), tea.WithAltScreen())
			_, err = p.Run()
			return err
		},
	}

	uiCmd.SilenceUsage = true
	rootCmd.AddCommand(uiCmd)
}
