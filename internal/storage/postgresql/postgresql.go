package postgresql

import (
	"context"
	"errors"
	"fmt"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fragpit/yandex-go-dev-metrics/internal/model"
	"github.com/fragpit/yandex-go-dev-metrics/internal/repository"
	"github.com/fragpit/yandex-go-dev-metrics/pkg/retry"
)

var _ repository.Repository = (*Storage)(nil)

type Storage struct {
	DB      *pgxpool.Pool
	retrier *retry.Retrier
}

func NewStorage(ctx context.Context, dbDSN string) (*Storage, error) {
	db, err := pgxpool.New(ctx, dbDSN)
	if err != nil {
		return nil, fmt.Errorf("error creating pgxpool: %w", err)
	}

	if err := db.Ping(ctx); err != nil {
		return nil, fmt.Errorf("db ping error: %w", err)
	}

	if err := runMigrations(ctx, db); err != nil {
		return nil, fmt.Errorf("error running migrations: %w", err)
	}

	isRetryable := func(err error) bool {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			return pgerrcode.IsConnectionException(pgErr.Code) ||
				pgerrcode.IsOperatorIntervention(pgErr.Code)
		}

		var connErr *pgconn.ConnectError
		return errors.As(err, &connErr)
	}

	retrier := retry.New(isRetryable)

	return &Storage{
		DB:      db,
		retrier: retrier,
	}, nil
}

func (s *Storage) GetMetrics(
	ctx context.Context,
) (map[string]model.Metric, error) {
	q := `SELECT id, type, value FROM metrics`
	rows, err := s.DB.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("error querying db: %w", err)
	}
	defer rows.Close()

	metrics := make(map[string]model.Metric)
	for rows.Next() {
		var id, metricType, value string
		if err := rows.Scan(&id, &metricType, &value); err != nil {
			return nil, fmt.Errorf("error reading values: %w", err)
		}

		metric, err := model.NewMetric(id, model.MetricType(metricType))
		if err != nil {
			return nil, err
		}

		if err := metric.SetValue(value); err != nil {
			return nil, err
		}

		metrics[id] = metric
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return metrics, nil
}

func (s *Storage) GetMetric(
	ctx context.Context,
	name string,
) (model.Metric, error) {
	q := `SELECT id, type, value FROM metrics WHERE id = $1`
	row := s.DB.QueryRow(ctx, q, name)

	var id, metricType, value string
	if err := row.Scan(&id, &metricType, &value); err != nil {
		return nil, fmt.Errorf("error reading values: %w", err)
	}

	metric, err := model.NewMetric(id, model.MetricType(metricType))
	if err != nil {
		return nil, err
	}

	if err := metric.SetValue(value); err != nil {
		return nil, err
	}

	return metric, nil
}

func (s *Storage) SetOrUpdateMetric(
	ctx context.Context,
	metric model.Metric,
) error {
	var q string

	if metric.GetType() == "counter" {
		q = `
		INSERT INTO metrics (id, type, value)
		VALUES ($1, $2, $3)
		ON CONFLICT (id) DO
		UPDATE SET value = CAST(metrics.value AS BIGINT) +
							CAST(EXCLUDED.value AS BIGINT)
		`
	} else {
		q = `
		INSERT INTO metrics (id, type, value)
		VALUES ($1, $2, $3)
		ON CONFLICT (id) DO
		UPDATE SET value = EXCLUDED.value
		`
	}

	_, err := s.DB.Exec(
		ctx,
		q,
		metric.GetID(),
		metric.GetType(),
		metric.GetValue(),
	)
	if err != nil {
		return fmt.Errorf("error querying db: %w", err)
	}

	return nil
}

func (s *Storage) SetOrUpdateMetricBatch(
	ctx context.Context,
	metrics []model.Metric,
) error {
	qCounter := `
		INSERT INTO metrics (id, type, value)
		VALUES ($1, $2, $3)
		ON CONFLICT (id) DO
		UPDATE SET value = CAST(metrics.value AS BIGINT) +
							CAST(EXCLUDED.value AS BIGINT)
    `

	qGauge := `
		INSERT INTO metrics (id, type, value)
		VALUES ($1, $2, $3)
		ON CONFLICT (id) DO UPDATE
		SET value = EXCLUDED.value
    `

	b := &pgx.Batch{}

	for _, m := range metrics {
		var q string
		if m.GetType() == "counter" {
			q = qCounter
		} else {
			q = qGauge
		}

		b.Queue(q, m.GetID(), m.GetType(), m.GetValue())
	}

	br := s.DB.SendBatch(ctx, b)
	defer br.Close()

	for i := 0; i < b.Len(); i++ {
		_, err := br.Exec()
		if err != nil {
			return fmt.Errorf(
				"error executing batch command %d: %w",
				i,
				err,
			)
		}
	}

	return nil
}

func (s *Storage) Initialize(metrics []model.Metric) error {
	return nil
}

func (s *Storage) Reset() error {
	return nil
}

func (s *Storage) Ping(ctx context.Context) error {
	if s.DB == nil {
		return fmt.Errorf("database connection is not initialized")
	}

	op := func(ctx context.Context) error {
		return s.DB.Ping(ctx)
	}

	if err := s.retrier.Do(ctx, op); err != nil {
		return err
	}

	return nil
}

func (s *Storage) Close(ctx context.Context) error {
	if s.DB != nil {
		s.DB.Close()
		return nil
	}

	return nil
}
