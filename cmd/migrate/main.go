package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	"wbtest/internal/config"
	"wbtest/internal/migrations"
)

func main() {
	var (
		command    = flag.String("cmd", "status", "Migration command: status, up, down")
		configFile = flag.String("config", ".env", "Configuration file")
	)
	flag.Parse()

	// Загружаем конфигурацию
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Подключаемся к базе данных
	db, err := connectDB(cfg.DatabaseURL())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Создаем мигратор
	migrator := migrations.NewMigrator(db, "schema_migrations")

	// Загружаем миграции
	if err := migrations.LoadMigrationsFromFiles(migrator, "migrations"); err != nil {
		log.Fatalf("Failed to load migrations: %v", err)
	}

	ctx := context.Background()

	switch *command {
	case "status":
		if err := showStatus(ctx, migrator); err != nil {
			log.Fatalf("Failed to show status: %v", err)
		}
	case "up":
		if err := migrateUp(ctx, migrator); err != nil {
			log.Fatalf("Failed to migrate up: %v", err)
		}
	case "down":
		if err := migrateDown(ctx, migrator); err != nil {
			log.Fatalf("Failed to migrate down: %v", err)
		}
	default:
		fmt.Printf("Unknown command: %s\n", *command)
		fmt.Println("Available commands: status, up, down")
		os.Exit(1)
	}
}

func showStatus(ctx context.Context, migrator *migrations.Migrator) error {
	statuses, err := migrator.GetStatus(ctx)
	if err != nil {
		return err
	}

	fmt.Println("Migration Status:")
	fmt.Println("=================")

	for _, status := range statuses {
		statusText := "PENDING"
		if status.Applied {
			statusText = "APPLIED"
			if status.AppliedAt != nil {
				statusText += fmt.Sprintf(" (%s)", status.AppliedAt.Format("2006-01-02 15:04:05"))
			}
		}

		fmt.Printf("%3d | %-30s | %s\n", status.Version, status.Name, statusText)
	}

	return nil
}

func migrateUp(ctx context.Context, migrator *migrations.Migrator) error {
	fmt.Println("Running migrations...")

	if err := migrator.Migrate(ctx); err != nil {
		return err
	}

	fmt.Println("Migrations completed successfully!")
	return nil
}

func migrateDown(ctx context.Context, migrator *migrations.Migrator) error {
	fmt.Println("Rolling back last migration...")

	if err := migrator.Rollback(ctx); err != nil {
		return err
	}

	fmt.Println("Rollback completed successfully!")
	return nil
}

func connectDB(databaseURL string) (*sql.DB, error) {
	// В реальном проекте здесь будет подключение к PostgreSQL
	// Для демонстрации возвращаем nil
	return nil, fmt.Errorf("database connection not implemented in this demo")
}
