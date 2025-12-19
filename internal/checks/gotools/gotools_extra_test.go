package gotools

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/mrz1836/go-pre-commit/internal/shared"
)

func TestFumptCheck_Run_ContextCancelled_Extra(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	check := NewFumptCheck()
	err := check.Run(ctx, []string{"test.go"})
	assert.Error(t, err)
}

func TestLintCheck_Run_ContextCancelled_Extra(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	check := NewLintCheck()
	err := check.Run(ctx, []string{"test.go"})
	assert.Error(t, err)
}

func TestModTidyCheck_Run_ContextCancelled_Extra(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	check := NewModTidyCheck()
	err := check.Run(ctx, []string{"go.mod"})
	assert.Error(t, err)
}

func TestFumptCheck_Run_NoFiles_Extra(t *testing.T) {
	ctx := context.Background()
	check := NewFumptCheck()
	err := check.Run(ctx, []string{})
	assert.NoError(t, err)
}

func TestLintCheck_Run_NoFiles_Extra(t *testing.T) {
	ctx := context.Background()
	check := NewLintCheck()
	err := check.Run(ctx, []string{})
	assert.NoError(t, err)
}

func TestModTidyCheck_Run_NoFiles_Extra(t *testing.T) {
	ctx := context.Background()
	check := NewModTidyCheck()
	err := check.Run(ctx, []string{})
	assert.NoError(t, err)
}

func TestNewFumptCheckWithConfig_Extra(t *testing.T) {
	sharedCtx := shared.NewContext()
	timeout := 10 * time.Second
	check := NewFumptCheckWithConfig(sharedCtx, timeout)
	assert.Equal(t, timeout, check.timeout)
	assert.Equal(t, sharedCtx, check.sharedCtx)
}
