package migrations

import (
	"context"
	"fmt"
	"sort"
	"time"

	apperrors "wbtest/internal/errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Migration представляет одну миграцию
type Migration struct {
	Version   int
	Name      string
	UpSQL     string
	DownSQL   string
	AppliedAt *time.Time
}

// Migrator управляет миграциями базы данных
type Migrator struct {
	db         *pgxpool.Pool
	table      string
	migrations []Migration
}

// NewMigrator создает новый мигратор
func NewMigrator(db *pgxpool.Pool, tableName string) *Migrator {
	if tableName == "" {
		tableName = "schema_migrations"
	}

	return &Migrator{
		db:         db,
		table:      tableName,
		migrations: make([]Migration, 0),
	}
}

// AddMigration добавляет миграцию
func (m *Migrator) AddMigration(version int, name, upSQL, downSQL string) {
	migration := Migration{
		Version: version,
		Name:    name,
		UpSQL:   upSQL,
		DownSQL: downSQL,
	}

	m.migrations = append(m.migrations, migration)

	// Сортируем миграции по версии
	sort.Slice(m.migrations, func(i, j int) bool {
		return m.migrations[i].Version < m.migrations[j].Version
	})
}

// Initialize создает таблицу миграций если она не существует
func (m *Migrator) Initialize(ctx context.Context) error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			version INTEGER PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`, m.table)

	_, err := m.db.Exec(ctx, query)
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrorTypeDatabase, "failed to create migrations table")
	}

	return nil
}

// GetAppliedMigrations возвращает список примененных миграций
func (m *Migrator) GetAppliedMigrations(ctx context.Context) (map[int]*Migration, error) {
	query := fmt.Sprintf("SELECT version, name, applied_at FROM %s ORDER BY version", m.table)

	rows, err := m.db.Query(ctx, query)
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrorTypeDatabase, "failed to query applied migrations")
	}
	defer rows.Close()

	applied := make(map[int]*Migration)
	for rows.Next() {
		var version int
		var name string
		var appliedAt time.Time

		if err := rows.Scan(&version, &name, &appliedAt); err != nil {
			return nil, apperrors.Wrap(err, apperrors.ErrorTypeDatabase, "failed to scan migration row")
		}

		applied[version] = &Migration{
			Version:   version,
			Name:      name,
			AppliedAt: &appliedAt,
		}
	}

	if err := rows.Err(); err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrorTypeDatabase, "error iterating migration rows")
	}

	return applied, nil
}

// Migrate применяет все непримененные миграции
func (m *Migrator) Migrate(ctx context.Context) error {
	if err := m.Initialize(ctx); err != nil {
		return err
	}

	applied, err := m.GetAppliedMigrations(ctx)
	if err != nil {
		return err
	}

	var toApply []Migration
	for _, migration := range m.migrations {
		if _, exists := applied[migration.Version]; !exists {
			toApply = append(toApply, migration)
		}
	}

	if len(toApply) == 0 {
		return nil // Нет миграций для применения
	}

	// Применяем миграции в транзакции
	tx, err := m.db.Begin(ctx)
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrorTypeDatabase, "failed to begin migration transaction")
	}
	defer tx.Rollback(ctx)

	for _, migration := range toApply {
		if err := m.applyMigration(ctx, tx, migration); err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return apperrors.Wrap(err, apperrors.ErrorTypeDatabase, "failed to commit migration transaction")
	}

	return nil
}

// Rollback откатывает последнюю миграцию
func (m *Migrator) Rollback(ctx context.Context) error {
	applied, err := m.GetAppliedMigrations(ctx)
	if err != nil {
		return err
	}

	if len(applied) == 0 {
		return nil // Нет миграций для отката
	}

	// Находим последнюю примененную миграцию
	var lastVersion int
	for version := range applied {
		if version > lastVersion {
			lastVersion = version
		}
	}

	// Находим миграцию для отката
	var rollbackMigration *Migration
	for _, migration := range m.migrations {
		if migration.Version == lastVersion {
			rollbackMigration = &migration
			break
		}
	}

	if rollbackMigration == nil {
		return apperrors.New(apperrors.ErrorTypeDatabase, fmt.Sprintf("migration version %d not found", lastVersion))
	}

	if rollbackMigration.DownSQL == "" {
		return apperrors.New(apperrors.ErrorTypeDatabase, fmt.Sprintf("no rollback SQL for migration %d", lastVersion))
	}

	// Выполняем откат в транзакции
	tx, err := m.db.Begin(ctx)
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrorTypeDatabase, "failed to begin rollback transaction")
	}
	defer tx.Rollback(ctx)

	// Выполняем SQL отката
	if _, err := tx.Exec(ctx, rollbackMigration.DownSQL); err != nil {
		return apperrors.Wrap(err, apperrors.ErrorTypeDatabase, fmt.Sprintf("failed to execute rollback SQL for migration %d", lastVersion))
	}

	// Удаляем запись о миграции
	query := fmt.Sprintf("DELETE FROM %s WHERE version = $1", m.table)
	if _, err := tx.Exec(ctx, query, lastVersion); err != nil {
		return apperrors.Wrap(err, apperrors.ErrorTypeDatabase, "failed to remove migration record")
	}

	if err := tx.Commit(ctx); err != nil {
		return apperrors.Wrap(err, apperrors.ErrorTypeDatabase, "failed to commit rollback transaction")
	}

	return nil
}

// GetStatus возвращает статус миграций
func (m *Migrator) GetStatus(ctx context.Context) ([]MigrationStatus, error) {
	applied, err := m.GetAppliedMigrations(ctx)
	if err != nil {
		return nil, err
	}

	var statuses []MigrationStatus
	for _, migration := range m.migrations {
		status := MigrationStatus{
			Version: migration.Version,
			Name:    migration.Name,
			Applied: false,
		}

		if applied, exists := applied[migration.Version]; exists {
			status.Applied = true
			status.AppliedAt = applied.AppliedAt
		}

		statuses = append(statuses, status)
	}

	return statuses, nil
}

// MigrationStatus представляет статус миграции
type MigrationStatus struct {
	Version   int
	Name      string
	Applied   bool
	AppliedAt *time.Time
}

// applyMigration применяет одну миграцию
func (m *Migrator) applyMigration(ctx context.Context, tx pgx.Tx, migration Migration) error {
	// Выполняем SQL миграции
	if _, err := tx.Exec(ctx, migration.UpSQL); err != nil {
		return apperrors.Wrap(err, apperrors.ErrorTypeDatabase, fmt.Sprintf("failed to execute migration %d: %s", migration.Version, migration.Name))
	}

	// Записываем информацию о примененной миграции
	query := fmt.Sprintf("INSERT INTO %s (version, name) VALUES ($1, $2)", m.table)
	if _, err := tx.Exec(ctx, query, migration.Version, migration.Name); err != nil {
		return apperrors.Wrap(err, apperrors.ErrorTypeDatabase, "failed to record migration")
	}

	return nil
}

// LoadMigrationsFromFiles загружает миграции из файлов
func LoadMigrationsFromFiles(migrator *Migrator, migrationsDir string) error {
	// Это упрощенная версия - в реальном проекте можно использовать go:embed
	// или файловую систему для загрузки SQL файлов

	// Пример миграций
	migrator.AddMigration(1, "create_orders_table", `
		CREATE TABLE IF NOT EXISTS orders (
			id SERIAL PRIMARY KEY,
			order_uid VARCHAR(255) UNIQUE NOT NULL,
			track_number VARCHAR(255) NOT NULL,
			entry VARCHAR(255) NOT NULL,
			delivery JSONB NOT NULL,
			payment JSONB NOT NULL,
			items JSONB NOT NULL,
			locale VARCHAR(10),
			internal_signature VARCHAR(255),
			customer_id VARCHAR(255),
			delivery_service VARCHAR(255),
			shardkey VARCHAR(255),
			sm_id INTEGER,
			date_created TIMESTAMP,
			oof_shard VARCHAR(255),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		
		CREATE INDEX IF NOT EXISTS idx_orders_order_uid ON orders(order_uid);
		CREATE INDEX IF NOT EXISTS idx_orders_track_number ON orders(track_number);
	`, `
		DROP TABLE IF EXISTS orders;
	`)

	migrator.AddMigration(2, "create_orders_indexes", `
		CREATE INDEX IF NOT EXISTS idx_orders_date_created ON orders(date_created);
		CREATE INDEX IF NOT EXISTS idx_orders_customer_id ON orders(customer_id);
	`, `
		DROP INDEX IF EXISTS idx_orders_date_created;
		DROP INDEX IF EXISTS idx_orders_customer_id;
	`)

	return nil
}
