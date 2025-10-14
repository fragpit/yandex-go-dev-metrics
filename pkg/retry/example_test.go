package retry

import (
	"context"
	"errors"
	"fmt"
)

func ExampleRetrier_Do_databasePing() {
	isRetryable := func(err error) bool {
		return errors.Is(err, errConnectionException) ||
			errors.Is(err, errOperatorIntervention)
	}

	retrier := NewRetrier(isRetryable)

	db := &mockDB{
		pingErrors: []error{
			errConnectionException,
			nil,
		},
	}

	op := func(ctx context.Context) error {
		return db.Ping(ctx)
	}

	ctx := context.Background()
	err := retrier.Do(ctx, op)

	fmt.Printf("Ping successful: %v, Attempts: %d\n", err == nil, db.pingCount)

	// Output:
	// Ping successful: true, Attempts: 2
}
