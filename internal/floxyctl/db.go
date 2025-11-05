package floxyctl

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/term"

	floxy "github.com/rom8726/floxy-pro"
)

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
}

func ConnectDB(ctx context.Context, config DBConfig) (*pgxpool.Pool, error) {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(config.User, config.Password),
		Host:   fmt.Sprintf("%s:%s", config.Host, config.Port),
		Path:   "/" + config.Database,
	}

	q := u.Query()
	q.Set("sslmode", "disable")
	q.Set("search_path", "workflows")
	u.RawQuery = q.Encode()

	connStr := u.String()

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}

func ReadPassword() (string, error) {
	_, _ = fmt.Fprint(os.Stderr, "Password: ")
	password, err := term.ReadPassword(syscall.Stdin)
	_, _ = fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", fmt.Errorf("failed to read password: %w", err)
	}

	return string(password), nil
}

func CreateEngineFromDB(ctx context.Context, pool *pgxpool.Pool) (*floxy.Engine, error) {
	if err := floxy.RunMigrations(ctx, pool); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	engine := floxy.NewEngine(pool)

	return engine, nil
}
