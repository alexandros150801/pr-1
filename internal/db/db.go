package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"pr-1/internal/utils"
)

// DB — пул соединений к PostgreSQL.
type DB struct {
	Pool *pgxpool.Pool
	Log  *utils.Logger
}

// Connect устанавливает соединение с БД.
func Connect(ctx context.Context, cfg *utils.Config, log *utils.Logger) (*DB, error) {
	dsn := fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.Name, cfg.Database.User, cfg.Database.Password)

	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("ошибка конфигурации подключения: %w", err)
	}
	poolCfg.MaxConns = 10

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("не удалось создать пул соединений: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("не удалось подключиться к БД: %w", err)
	}
	log.Info("подключение к БД установлено (%s:%d/%s)", cfg.Database.Host, cfg.Database.Port, cfg.Database.Name)
	return &DB{Pool: pool, Log: log}, nil
}

// Close закрывает пул соединений.
func (d *DB) Close() {
	if d.Pool != nil {
		d.Pool.Close()
	}
}
