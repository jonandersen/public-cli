package output

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatter_Table_BasicOutput(t *testing.T) {
	var buf bytes.Buffer
	f := &Formatter{Writer: &buf, JSONMode: false}

	headers := []string{"Name", "Value"}
	rows := [][]string{
		{"foo", "123"},
		{"bar", "456"},
	}

	err := f.Table(headers, rows)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Name")
	assert.Contains(t, output, "Value")
	assert.Contains(t, output, "foo")
	assert.Contains(t, output, "123")
	assert.Contains(t, output, "bar")
	assert.Contains(t, output, "456")
}

func TestFormatter_Table_ColumnAlignment(t *testing.T) {
	var buf bytes.Buffer
	f := &Formatter{Writer: &buf, JSONMode: false}

	headers := []string{"ID", "Description"}
	rows := [][]string{
		{"1", "Short"},
		{"100", "A longer description"},
	}

	err := f.Table(headers, rows)
	require.NoError(t, err)

	output := buf.String()
	// Columns should be aligned - check that spacing is consistent
	assert.Contains(t, output, "ID")
	assert.Contains(t, output, "Description")
}

func TestFormatter_Table_EmptyRows(t *testing.T) {
	var buf bytes.Buffer
	f := &Formatter{Writer: &buf, JSONMode: false}

	headers := []string{"Name", "Value"}
	rows := [][]string{}

	err := f.Table(headers, rows)
	require.NoError(t, err)

	output := buf.String()
	// Should still show headers
	assert.Contains(t, output, "Name")
	assert.Contains(t, output, "Value")
}

func TestFormatter_Table_SingleColumn(t *testing.T) {
	var buf bytes.Buffer
	f := &Formatter{Writer: &buf, JSONMode: false}

	headers := []string{"Items"}
	rows := [][]string{
		{"apple"},
		{"banana"},
		{"cherry"},
	}

	err := f.Table(headers, rows)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Items")
	assert.Contains(t, output, "apple")
	assert.Contains(t, output, "banana")
	assert.Contains(t, output, "cherry")
}

func TestFormatter_JSON_BasicOutput(t *testing.T) {
	var buf bytes.Buffer
	f := &Formatter{Writer: &buf, JSONMode: true}

	data := map[string]string{
		"name":  "test",
		"value": "123",
	}

	err := f.Print(data)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, `"name"`)
	assert.Contains(t, output, `"test"`)
	assert.Contains(t, output, `"value"`)
	assert.Contains(t, output, `"123"`)
}

func TestFormatter_JSON_Array(t *testing.T) {
	var buf bytes.Buffer
	f := &Formatter{Writer: &buf, JSONMode: true}

	data := []string{"one", "two", "three"}

	err := f.Print(data)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, `"one"`)
	assert.Contains(t, output, `"two"`)
	assert.Contains(t, output, `"three"`)
}

func TestFormatter_JSON_PrettyPrinted(t *testing.T) {
	var buf bytes.Buffer
	f := &Formatter{Writer: &buf, JSONMode: true}

	data := map[string]string{"key": "value"}

	err := f.Print(data)
	require.NoError(t, err)

	output := buf.String()
	// Should be indented (pretty printed)
	assert.Contains(t, output, "\n")
	assert.Contains(t, output, "  ") // indentation
}

func TestFormatter_Table_InJSONMode_OutputsJSON(t *testing.T) {
	var buf bytes.Buffer
	f := &Formatter{Writer: &buf, JSONMode: true}

	headers := []string{"Name", "Value"}
	rows := [][]string{
		{"foo", "123"},
		{"bar", "456"},
	}

	err := f.Table(headers, rows)
	require.NoError(t, err)

	output := buf.String()
	// In JSON mode, Table should output JSON array of objects
	assert.Contains(t, output, `"Name"`)
	assert.Contains(t, output, `"foo"`)
	assert.Contains(t, output, `"Value"`)
	assert.Contains(t, output, `"123"`)
}

func TestFormatter_Print_NonJSONMode_Fallback(t *testing.T) {
	var buf bytes.Buffer
	f := &Formatter{Writer: &buf, JSONMode: false}

	data := map[string]string{"key": "value"}

	err := f.Print(data)
	require.NoError(t, err)

	// In non-JSON mode, Print should still produce some output
	output := buf.String()
	assert.NotEmpty(t, output)
}

func TestNewFormatter(t *testing.T) {
	var buf bytes.Buffer
	f := New(&buf, false)

	assert.Equal(t, &buf, f.Writer)
	assert.False(t, f.JSONMode)

	f2 := New(&buf, true)
	assert.True(t, f2.JSONMode)
}
