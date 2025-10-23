package audit

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const (
	DefaultTimeout = 5 * time.Second
)

type Event struct {
	Timestamp int64    `json:"ts"`
	Metrics   []string `json:"metrics"`
	IPAddress string   `json:"ip_address"`
}

// Observer is an interface for components that want to receive audit events.
type Observer interface {
	Notify(ctx context.Context, event Event) error
}

type Auditor struct {
	observers []Observer
}

func NewAuditor() *Auditor {
	return &Auditor{}
}

// Add registers a new observer to receive audit events.
func (a *Auditor) Add(observer Observer) {
	a.observers = append(a.observers, observer)
}

// LogEvent notifies all observers about the event.
func (a *Auditor) LogEvent(
	ctx context.Context,
	metrics []string,
	ipAddress string,
) error {
	if len(a.observers) == 0 {
		return nil
	}

	event := Event{
		Timestamp: time.Now().Unix(),
		Metrics:   metrics,
		IPAddress: ipAddress,
	}

	var errs []error
	for i, obs := range a.observers {
		if err := obs.Notify(ctx, event); err != nil {
			errs = append(errs, fmt.Errorf(
				"observer %d (%T) failed: %w",
				i,
				obs,
				err,
			))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}
