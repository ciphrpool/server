package services

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/joho/godotenv/autoload"
)

type Database struct {
	Pool *pgxpool.Pool
}

func DefaultDatabase() Database {
	return Database{
		Pool: nil,
	}
}

func (db *Database) Connect(password string) error {
	address := os.Getenv("DATABASE_ADDRESS")
	uri := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", "mcs", password, address, "basepool")
	config, err := pgxpool.ParseConfig(uri)
	if err != nil {
		return fmt.Errorf("failed to connect to PostgresDB: %w", err)
	}
	config.MaxConns = 10
	config.MinConns = 2
	config.MaxConnLifetime = 1 * time.Hour
	config.MaxConnIdleTime = 30 * time.Minute
	config.HealthCheckPeriod = 1 * time.Minute

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return fmt.Errorf("failed to ping database: %w", err)
	}

	db.Pool = pool
	slog.Info("Db connection succeeded")
	return nil
}

func StringToUUID(s string) (pgtype.UUID, error) {
	var pgUUID pgtype.UUID

	// Parse the string into a UUID
	parsedUUID, err := uuid.Parse(s)
	if err != nil {
		return pgUUID, err
	}

	// Convert to pgtype.UUID
	pgUUID.Bytes = parsedUUID
	pgUUID.Valid = true

	return pgUUID, nil
}

func StringToTimestampz(s string) (pgtype.Timestamptz, error) {
	// Parse the string to time.Time
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return pgtype.Timestamptz{}, err
	}

	// Convert to pgtype.Timestamptz
	var ts pgtype.Timestamptz
	ts.Time = t
	ts.Valid = true

	return ts, nil
}

func UUIDToString(id pgtype.UUID) string {
	if !id.Valid {
		return ""
	}
	return uuid.UUID(id.Bytes).String()
}

func (s *Database) Health() bool {
	if s.Pool == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	return s.Pool.Ping(ctx) == nil
}
