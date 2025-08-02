package postgresql

import (
	"context"
	"database/sql"
	"embed"
	"errors"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/lib/pq"

	"github.com/fragpit/yandex-go-dev-metrics/internal/model"
	"github.com/fragpit/yandex-go-dev-metrics/internal/repository"
)

var _ repository.Repository = (*Storage)(nil)

type Storage struct {
	DB *sql.DB
}

//go:embed migrations/*.sql
var migrationsFS embed.FS

func NewStorage(ctx context.Context, dbDSN string) (*Storage, error) {
	db, err := sql.Open("postgres", dbDSN)
	if err != nil {
		return nil, err
	}

	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return nil, err
	}

	// m, err := migrate.NewWithDatabaseInstance(
	// 	"file:///migrations",
	// 	"postgresql",
	// 	driver,
	// )
	// if err != nil {
	// 	return nil, err
	// }

	sourceDriver, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return nil, err
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, "postgresql", driver)
	if err != nil {
		return nil, err
	}

	if err := m.Up(); err != nil {
		if !errors.Is(err, migrate.ErrNoChange) {
			return nil, err
		}
	}

	return &Storage{
		DB: db,
	}, nil
}

func (s *Storage) GetMetrics(
	ctx context.Context,
) (map[string]model.Metric, error) {
	q := `SELECT id, type, value FROM metrics`
	rows, err := s.DB.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	metrics := make(map[string]model.Metric)
	for rows.Next() {
		var id, metricType, value string
		if err := rows.Scan(&id, &metricType, &value); err != nil {
			return nil, err
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
	row := s.DB.QueryRowContext(ctx, q, name)

	var id, metricType, value string
	if err := row.Scan(&id, &metricType, &value); err != nil {
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

	_, err := s.DB.ExecContext(
		ctx,
		q,
		metric.GetID(),
		metric.GetType(),
		metric.GetValue(),
	)
	if err != nil {
		return err
	}

	return nil
}

func (s *Storage) SetOrUpdateMetricBatch(
	ctx context.Context,
	metrics []model.Metric,
) error {
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}

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
	ON CONFLICT (id) DO
	UPDATE SET value = EXCLUDED.value
	`

	for _, m := range metrics {
		var q string
		if m.GetType() == "counter" {
			q = qCounter
		} else {
			q = qGauge
		}

		_, err := tx.ExecContext(
			ctx,
			q,
			m.GetID(),
			m.GetType(),
			m.GetValue(),
		)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
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
