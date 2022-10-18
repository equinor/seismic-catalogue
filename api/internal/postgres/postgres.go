package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/equinor/seismic-catalogue/api/internal/database"
)

func formatStringArray(in []string) string {
	return "'" + strings.Join(in, "', '") + "'"
}

/** Postgres implementation of database.Adapter */
type Adapter struct {
	pool *pgxpool.Pool
}

func (c *Adapter) Close() {
	c.pool.Close()
}

func (c *Adapter) GetCubes(fields []string) ([]database.CubeEntry, error) {
	sql := fmt.Sprintf(
		"SELECT * FROM catalogue.cube WHERE field IN (%s)",
		formatStringArray(fields),
	)

	rows, err := c.pool.Query(context.Background(), sql)
	if err != nil {
		return nil, fmt.Errorf("query failed with: %v", err)
	}

	return pgx.CollectRows(rows, pgx.RowToStructByPos[database.CubeEntry])
}

func NewAdapter(connString string) (*Adapter, error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("unable to parse config: %v\n", err)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %v\n", err)
	}
	return &Adapter{pool: pool}, nil
}
