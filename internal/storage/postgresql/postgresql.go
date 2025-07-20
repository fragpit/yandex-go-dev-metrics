package postgresql

import (
	"context"
	"database/sql"

	_ "github.com/lib/pq"

	"github.com/fragpit/yandex-go-dev-metrics/internal/model"
	"github.com/fragpit/yandex-go-dev-metrics/internal/repository"
)

var _ repository.Repository = (*Storage)(nil)

type Storage struct {
	DB *sql.DB
}

func NewStorage(ctx context.Context, dbDSN string) (*Storage, error) {
	db, err := sql.Open("postgres", dbDSN)
	if err != nil {
		return nil, err
	}

	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	dbCreateQuery := `
	CREATE TABLE IF NOT EXISTS metrics (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			value TEXT NOT NULL
	);
	`

	if _, err := db.ExecContext(ctx, dbCreateQuery); err != nil {
		return nil, err
	}

	return &Storage{
		DB: db,
	}, nil
}

func (s *Storage) GetMetrics(
	ctx context.Context,
) (map[string]model.Metric, error) {
	return nil, nil
}

func (s *Storage) GetMetric(
	ctx context.Context,
	name string,
) (model.Metric, error) {
	q := `SELECT id, type, value FROM metrics WHERE id == $1`
	row := s.DB.QueryRowContext(ctx, q)

	var id, metricType, value string
	if err := row.Scan(&id, &metricType, &value); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
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
	return nil
}

func (s *Storage) Initialize(metrics []model.Metric) error {
	return nil
}

func (s *Storage) Ping(ctx context.Context) error {
	if s.DB == nil {
		return nil
	}
	return s.DB.PingContext(ctx)
}

func (s *Storage) Close() error {
	if s.DB != nil {
		return s.DB.Close()
	}
	return nil
}
