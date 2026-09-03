package printer

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type testStruct struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

func TestPrint_JSON(t *testing.T) {
	var buf bytes.Buffer
	data := testStruct{ID: "1", Name: "Test", Status: "ok"}

	err := Print(data, FormatJSON, &buf)
	require.NoError(t, err)

	var result testStruct
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "Test", result.Name)
}

func TestPrint_YAML(t *testing.T) {
	var buf bytes.Buffer
	data := testStruct{ID: "1", Name: "Test", Status: "ok"}

	err := Print(data, FormatYAML, &buf)
	require.NoError(t, err)

	var result testStruct
	err = yaml.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "Test", result.Name)
}

func TestPrint_Table_Struct(t *testing.T) {
	var buf bytes.Buffer
	data := testStruct{ID: "abc-123", Name: "MyApp", Status: "running"}

	err := Print(data, FormatTable, &buf)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "id")
	assert.Contains(t, output, "abc-123")
	assert.Contains(t, output, "MyApp")
}

func TestPrint_Table_Slice(t *testing.T) {
	var buf bytes.Buffer
	data := []testStruct{
		{ID: "1", Name: "App1", Status: "running"},
		{ID: "2", Name: "App2", Status: "pending"},
	}

	err := Print(data, FormatTable, &buf)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "App1")
	assert.Contains(t, output, "App2")
	assert.Contains(t, output, "running")
	assert.Contains(t, output, "pending")
}

func TestPrint_Table_EmptySlice(t *testing.T) {
	var buf bytes.Buffer
	data := []testStruct{}

	err := Print(data, FormatTable, &buf)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No data")
}

func TestPrint_Table_LongValueTruncation(t *testing.T) {
	var buf bytes.Buffer
	type longStruct struct {
		Description string `json:"description"`
	}
	data := []longStruct{
		{Description: string(make([]byte, 100))},
	}

	err := Print(data, FormatTable, &buf)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "...")
}
