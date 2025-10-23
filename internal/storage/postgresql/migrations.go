package postgresql

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/tern/v2/migrate"
)

// runMigrations applies database migrations using the tern library.
func runMigrations(ctx context.Context, conn *pgxpool.Pool) error {
	poolConn, err := conn.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("error creating pool connection: %w", err)
	}
	defer poolConn.Release()

	m, err := migrate.NewMigrator(ctx, poolConn.Conn(), "metrics_migrations")
	if err != nil {
		return fmt.Errorf("error migrations init: %w", err)
	}

	m.Migrations = []*migrate.Migration{
		{
			Sequence: 1,
			Name:     "create metrics table",
			UpSQL: `
			CREATE TABLE IF NOT EXISTS metrics (
					id TEXT PRIMARY KEY,
					type TEXT NOT NULL,
					value TEXT NOT NULL
			);
			`,
			DownSQL: `DROP TABLE IF EXISTS metrics;`,
		},
	}

	if err := m.Migrate(ctx); err != nil {
		return fmt.Errorf("error applying migrations: %w", err)
	}

	return nil
}
