package migrations

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

func setupTestDB(t *testing.T) *sql.DB {
	// В реальном проекте здесь будет подключение к тестовой БД
	// Для тестов используем in-memory SQLite или мок
	t.Skip("Skipping database tests - requires test database setup")
	return nil
}

func TestNewMigrator(t *testing.T) {
	db := setupTestDB(t)

	migrator := NewMigrator(db, "test_migrations")
	if migrator == nil {
		t.Error("Migrator is nil")
	}

	if migrator.table != "test_migrations" {
		t.Errorf("Expected table 'test_migrations', got '%s'", migrator.table)
	}

	if migrator.db != db {
		t.Error("Database connection is not set correctly")
	}
}

func TestNewMigratorDefaultTable(t *testing.T) {
	db := setupTestDB(t)

	migrator := NewMigrator(db, "")
	if migrator.table != "schema_migrations" {
		t.Errorf("Expected default table 'schema_migrations', got '%s'", migrator.table)
	}
}

func TestAddMigration(t *testing.T) {
	db := setupTestDB(t)
	migrator := NewMigrator(db, "test_migrations")

	migrator.AddMigration(2, "second_migration", "CREATE TABLE test2 (id INT);", "DROP TABLE test2;")
	migrator.AddMigration(1, "first_migration", "CREATE TABLE test1 (id INT);", "DROP TABLE test1;")
	migrator.AddMigration(3, "third_migration", "CREATE TABLE test3 (id INT);", "DROP TABLE test3;")

	if len(migrator.migrations) != 3 {
		t.Errorf("Expected 3 migrations, got %d", len(migrator.migrations))
	}

	// Проверяем что миграции отсортированы по версии
	expectedVersions := []int{1, 2, 3}
	for i, migration := range migrator.migrations {
		if migration.Version != expectedVersions[i] {
			t.Errorf("Expected version %d at index %d, got %d", expectedVersions[i], i, migration.Version)
		}
	}
}

func TestLoadMigrationsFromFiles(t *testing.T) {
	db := setupTestDB(t)
	migrator := NewMigrator(db, "test_migrations")

	err := LoadMigrationsFromFiles(migrator, "test_migrations")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(migrator.migrations) == 0 {
		t.Error("Expected migrations to be loaded")
	}

	// Проверяем что первая миграция создает таблицу orders
	found := false
	for _, migration := range migrator.migrations {
		if migration.Version == 1 && strings.Contains(migration.UpSQL, "CREATE TABLE IF NOT EXISTS orders") {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected to find orders table creation migration")
	}
}

func TestMigrationStatus(t *testing.T) {
	tests := []struct {
		name      string
		version   int
		applied   bool
		appliedAt *time.Time
	}{
		{
			name:    "applied migration",
			version: 1,
			applied: true,
			appliedAt: func() *time.Time {
				t := time.Now()
				return &t
			}(),
		},
		{
			name:    "not applied migration",
			version: 2,
			applied: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := MigrationStatus{
				Version:   tt.version,
				Name:      "test_migration",
				Applied:   tt.applied,
				AppliedAt: tt.appliedAt,
			}

			if status.Version != tt.version {
				t.Errorf("Expected version %d, got %d", tt.version, status.Version)
			}

			if status.Applied != tt.applied {
				t.Errorf("Expected applied %v, got %v", tt.applied, status.Applied)
			}

			if tt.applied && status.AppliedAt == nil {
				t.Error("Expected AppliedAt to be set for applied migration")
			}

			if !tt.applied && status.AppliedAt != nil {
				t.Error("Expected AppliedAt to be nil for not applied migration")
			}
		})
	}
}

func TestMigrationStruct(t *testing.T) {
	now := time.Now()

	migration := Migration{
		Version:   1,
		Name:      "test_migration",
		UpSQL:     "CREATE TABLE test (id INT);",
		DownSQL:   "DROP TABLE test;",
		AppliedAt: &now,
	}

	if migration.Version != 1 {
		t.Errorf("Expected version 1, got %d", migration.Version)
	}

	if migration.Name != "test_migration" {
		t.Errorf("Expected name 'test_migration', got '%s'", migration.Name)
	}

	if migration.UpSQL != "CREATE TABLE test (id INT);" {
		t.Errorf("Expected UpSQL 'CREATE TABLE test (id INT);', got '%s'", migration.UpSQL)
	}

	if migration.DownSQL != "DROP TABLE test;" {
		t.Errorf("Expected DownSQL 'DROP TABLE test;', got '%s'", migration.DownSQL)
	}

	if migration.AppliedAt == nil {
		t.Error("Expected AppliedAt to be set")
	}

	if migration.AppliedAt != &now {
		t.Error("AppliedAt should point to the same time")
	}
}
