package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRootCmd_JSONFlagExists(t *testing.T) {
	// Reset the flag for testing
	jsonOutput = false

	cmd := rootCmd
	flag := cmd.PersistentFlags().Lookup("json")

	assert.NotNil(t, flag, "--json flag should exist")
	assert.Equal(t, "false", flag.DefValue)
	assert.Equal(t, "Output in JSON format", flag.Usage)
}

func TestRootCmd_JSONFlagShorthand(t *testing.T) {
	cmd := rootCmd
	flag := cmd.PersistentFlags().ShorthandLookup("j")

	assert.NotNil(t, flag, "-j shorthand should exist")
	assert.Equal(t, "json", flag.Name)
}

func TestRootCmd_GetJSONMode(t *testing.T) {
	// Test default value
	jsonOutput = false
	assert.False(t, GetJSONMode())

	// Test when set
	jsonOutput = true
	assert.True(t, GetJSONMode())

	// Reset
	jsonOutput = false
}

func TestRootCmd_Version(t *testing.T) {
	var out bytes.Buffer
	cmd := rootCmd
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--version"})

	_ = cmd.Execute()

	output := out.String()
	assert.Contains(t, output, "pub version")
}
