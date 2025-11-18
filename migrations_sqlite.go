package floxy

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

//go:embed migrations_sqlite/*.sql
var sqliteMigrationFiles embed.FS

// RunSQLiteMigrations executes embedded SQLite migrations in lexical order.
// Note: SQLite DDL operations are auto-committed, so we don't use a transaction.
func RunSQLiteMigrations(ctx context.Context, db *sql.DB) error {
	entries, err := fs.ReadDir(sqliteMigrationFiles, "migrations_sqlite")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}
	if len(entries) == 0 {
		return nil
	}

	// Sort by filename to ensure deterministic order (e.g., 0001_..., 0002_...)
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		b, err := sqliteMigrationFiles.ReadFile("migrations_sqlite/" + e.Name())
		if err != nil {
			return fmt.Errorf("read migration %s: %w", e.Name(), err)
		}
		content := string(b)
		// Very simple split by semicolon; adequate for our DDL files
		stmts := splitSQLStatements(content)
		for _, stmt := range stmts {
			if _, err := db.ExecContext(ctx, stmt); err != nil {
				return fmt.Errorf("exec migration %s: %w", e.Name(), err)
			}
		}
	}

	// Verify that critical tables exist after migrations
	var tableName string
	requiredTables := []string{"workflow_definitions", "workflow_instances", "workflow_steps", "queue"}
	for _, table := range requiredTables {
		err := db.QueryRowContext(ctx, "SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&tableName)
		if err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("migration verification failed: table %s does not exist", table)
			}
			return fmt.Errorf("verify migration: %w", err)
		}
	}

	return nil
}

func splitSQLStatements(sqlText string) []string {
	parts := strings.Split(sqlText, ";")
	res := make([]string, 0, len(parts))
	for _, p := range parts {
		stmt := strings.TrimSpace(p)
		if stmt == "" {
			continue
		}
		// Skip pure comment-only lines
		lines := strings.Split(stmt, "\n")
		allComment := true
		for _, ln := range lines {
			l := strings.TrimSpace(ln)
			if l == "" {
				continue
			}
			if !strings.HasPrefix(l, "--") {
				allComment = false
				break
			}
		}
		if allComment {
			continue
		}
		res = append(res, stmt)
	}
	return res
}
