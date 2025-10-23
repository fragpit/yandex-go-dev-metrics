package server

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServerComponents(t *testing.T) {
	t.Run("context creation", func(t *testing.T) {
		ctx := context.Background()
		assert.NotNil(t, ctx)
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		assert.NotNil(t, ctx)
		assert.NotNil(t, cancel)
		cancel()
		assert.Error(t, ctx.Err())
	})
}
