package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStatusCmd_CommandStructure(t *testing.T) {
	// Create CLI app and command builder
	app := NewCLIApp("test", "test-commit", "test-date")
	builder := NewCommandBuilder(app)
	statusCmd := builder.BuildStatusCmd()

	// Test basic command properties
	assert.Equal(t, "status", statusCmd.Use)
	assert.Contains(t, statusCmd.Short, "Show installation status")

	// Test that the command has proper structure
	assert.NotNil(t, statusCmd.RunE, "RunE function should be set")
}
