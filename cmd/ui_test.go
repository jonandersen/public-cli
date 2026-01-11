package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestUICommandExists(t *testing.T) {
	// Verify the ui command is registered
	cmd := rootCmd.Commands()
	var found bool
	for _, c := range cmd {
		if c.Name() == "ui" {
			found = true
			break
		}
	}
	assert.True(t, found, "ui command should be registered")
}

func TestUICommandDescription(t *testing.T) {
	// Find the ui command
	var uiCmd *cobra.Command
	for _, c := range rootCmd.Commands() {
		if c.Name() == "ui" {
			uiCmd = c
			break
		}
	}
	assert.NotNil(t, uiCmd)
	assert.Equal(t, "ui", uiCmd.Use)
	assert.Contains(t, uiCmd.Short, "Interactive")
}

func TestNewModel(t *testing.T) {
	m := newModel()
	assert.NotNil(t, m)
	assert.Equal(t, viewPortfolio, m.currentView)
}

func TestModelInit(t *testing.T) {
	m := newModel()
	cmd := m.Init()
	// Init should return nil (no initial command)
	assert.Nil(t, cmd)
}

func TestModelView(t *testing.T) {
	m := newModel()
	m.width = 80
	m.height = 24
	m.ready = true
	view := m.View()

	// Should contain header
	assert.Contains(t, view, "pub")
	// Should contain footer with key hints
	assert.Contains(t, view, "q")
}
