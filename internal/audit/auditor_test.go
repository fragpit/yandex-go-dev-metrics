package audit

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockObserver struct {
	called bool
	err    error
}

func (m *mockObserver) Notify(ctx context.Context, event Event) error {
	m.called = true
	return m.err
}

func TestNewAuditor(t *testing.T) {
	auditor := NewAuditor()
	assert.NotNil(t, auditor)
	assert.Empty(t, auditor.observers)
}

func TestAuditor_Add(t *testing.T) {
	auditor := NewAuditor()
	observer := &mockObserver{}

	auditor.Add(observer)

	assert.Equal(t, 1, len(auditor.observers))
}

func TestAuditor_LogEvent(t *testing.T) {
	t.Run("no observers", func(t *testing.T) {
		auditor := NewAuditor()
		err := auditor.LogEvent(
			context.Background(),
			[]string{"metric1"},
			"127.0.0.1",
		)
		assert.NoError(t, err)
	})

	t.Run("with observer", func(t *testing.T) {
		auditor := NewAuditor()
		observer := &mockObserver{}
		auditor.Add(observer)

		err := auditor.LogEvent(
			context.Background(),
			[]string{"metric1"},
			"127.0.0.1",
		)
		assert.NoError(t, err)
		assert.True(t, observer.called)
	})

	t.Run("observer returns error", func(t *testing.T) {
		auditor := NewAuditor()
		observer := &mockObserver{err: assert.AnError}
		auditor.Add(observer)

		err := auditor.LogEvent(
			context.Background(),
			[]string{"metric1"},
			"127.0.0.1",
		)
		assert.Error(t, err)
		assert.True(t, observer.called)
	})
}
