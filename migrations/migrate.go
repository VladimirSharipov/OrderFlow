package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Migration struct {
	Version   int
	Name      string
	UpSQL     string
	DownSQL   string
	CreatedAt time.Time
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run migrations/migrate.go <database_url> <command> [version]")
		fmt.Println("Commands: up, down, status")
		fmt.Println("Examples:")
		fmt.Println("  go run migrations/migrate.go 'postgres://user:pass@localhost/db' up")
		fmt.Println("  go run migrations/migrate.go 'postgres://user:pass@localhost/db' down 1")
		fmt.Println("  go run migrations/migrate.go 'postgres://user:pass@localhost/db' status")
		os.Exit(1)
	}

	dbURL := os.Args[1]
	command := os.Args[2]

	// Используем драйвер pgx (stdlib)
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Проверяем соединение
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// Создаем таблицу миграций если её нет
	if err := createMigrationsTable(db); err != nil {
		log.Fatalf("Failed to create migrations table: %v", err)
	}

	switch command {
	case "up":
		if err := runUpMigrations(db); err != nil {
			log.Fatalf("Failed to run up migrations: %v", err)
		}
	case "down":
		if len(os.Args) < 4 {
			log.Fatal("Version required for down migration")
		}
		version, err := strconv.Atoi(os.Args[3])
		if err != nil {
			log.Fatalf("Invalid version: %v", err)
		}
		if err := runDownMigration(db, version); err != nil {
			log.Fatalf("Failed to run down migration: %v", err)
		}
	case "status":
		if err := showStatus(db); err != nil {
			log.Fatalf("Failed to show status: %v", err)
		}
	default:
		log.Fatalf("Unknown command: %s", command)
	}
}

func createMigrationsTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS migrations (
			version INTEGER PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`
	_, err := db.Exec(query)
	return err
}

func runUpMigrations(db *sql.DB) error {
	migrations := getMigrations()
	appliedMigrations, err := getAppliedMigrations(db)
	if err != nil {
		return err
	}

	for _, migration := range migrations {
		if !appliedMigrations[migration.Version] {
			log.Printf("Applying migration %d: %s", migration.Version, migration.Name)

			tx, err := db.Begin()
			if err != nil {
				return fmt.Errorf("failed to begin transaction: %v", err)
			}

			// Выполняем миграцию
			if _, err := tx.Exec(migration.UpSQL); err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to execute migration %d: %v", migration.Version, err)
			}

			// Записываем в таблицу миграций
			if _, err := tx.Exec("INSERT INTO migrations (version, name) VALUES ($1, $2)",
				migration.Version, migration.Name); err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to record migration %d: %v", migration.Version, err)
			}

			if err := tx.Commit(); err != nil {
				return fmt.Errorf("failed to commit migration %d: %v", migration.Version, err)
			}

			log.Printf("Successfully applied migration %d: %s", migration.Version, migration.Name)
		}
	}

	return nil
}

func runDownMigration(db *sql.DB, targetVersion int) error {
	migrations := getMigrations()
	appliedMigrations, err := getAppliedMigrations(db)
	if err != nil {
		return err
	}

	// Находим миграцию для отката
	var targetMigration *Migration
	for _, migration := range migrations {
		if migration.Version == targetVersion {
			targetMigration = &migration
			break
		}
	}

	if targetMigration == nil {
		return fmt.Errorf("migration version %d not found", targetVersion)
	}

	if !appliedMigrations[targetVersion] {
		return fmt.Errorf("migration version %d is not applied", targetVersion)
	}

	log.Printf("Rolling back migration %d: %s", targetMigration.Version, targetMigration.Name)

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}

	// Выполняем down миграцию
	if _, err := tx.Exec(targetMigration.DownSQL); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to execute down migration %d: %v", targetMigration.Version, err)
	}

	// Удаляем запись из таблицы миграций
	if _, err := tx.Exec("DELETE FROM migrations WHERE version = $1", targetMigration.Version); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to remove migration record %d: %v", targetMigration.Version, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit down migration %d: %v", targetMigration.Version, err)
	}

	log.Printf("Successfully rolled back migration %d: %s", targetMigration.Version, targetMigration.Name)
	return nil
}

func showStatus(db *sql.DB) error {
	migrations := getMigrations()
	appliedMigrations, err := getAppliedMigrations(db)
	if err != nil {
		return err
	}

	fmt.Printf("%-10s %-20s %-15s %s\n", "Version", "Name", "Status", "Applied At")
	fmt.Println(strings.Repeat("-", 70))

	for _, migration := range migrations {
		status := "Pending"
		appliedAt := "-"

		if appliedMigrations[migration.Version] {
			status = "Applied"
			// Получаем время применения
			var appliedTime time.Time
			err := db.QueryRow("SELECT applied_at FROM migrations WHERE version = $1",
				migration.Version).Scan(&appliedTime)
			if err == nil {
				appliedAt = appliedTime.Format("2006-01-02 15:04:05")
			}
		}

		fmt.Printf("%-10d %-20s %-15s %s\n", migration.Version, migration.Name, status, appliedAt)
	}

	return nil
}

func getAppliedMigrations(db *sql.DB) (map[int]bool, error) {
	rows, err := db.Query("SELECT version FROM migrations ORDER BY version")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[int]bool)
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = true
	}

	return applied, rows.Err()
}

func getMigrations() []Migration {
	return []Migration{
		{
			Version: 1,
			Name:    "001_init",
			UpSQL: `
				CREATE TABLE orders (
					order_uid VARCHAR PRIMARY KEY,
					track_number VARCHAR,
					entry VARCHAR,
					locale VARCHAR,
					internal_signature VARCHAR,
					customer_id VARCHAR,
					delivery_service VARCHAR,
					shardkey VARCHAR,
					sm_id INT,
					date_created TIMESTAMP,
					oof_shard VARCHAR
				);

				CREATE TABLE delivery (
					order_uid VARCHAR PRIMARY KEY REFERENCES orders(order_uid) ON DELETE CASCADE,
					name VARCHAR,
					phone VARCHAR,
					zip VARCHAR,
					city VARCHAR,
					address VARCHAR,
					region VARCHAR,
					email VARCHAR
				);

				CREATE TABLE payment (
					transaction VARCHAR PRIMARY KEY,
					order_uid VARCHAR REFERENCES orders(order_uid) ON DELETE CASCADE,
					request_id VARCHAR,
					currency VARCHAR,
					provider VARCHAR,
					amount INT,
					payment_dt BIGINT,
					bank VARCHAR,
					delivery_cost INT,
					goods_total INT,
					custom_fee INT
				);

				CREATE TABLE items (
					id SERIAL PRIMARY KEY,
					order_uid VARCHAR REFERENCES orders(order_uid) ON DELETE CASCADE,
					chrt_id INT,
					track_number VARCHAR,
					price INT,
					rid VARCHAR,
					name VARCHAR,
					sale INT,
					size VARCHAR,
					total_price INT,
					nm_id INT,
					brand VARCHAR,
					status INT
				);
			`,
			DownSQL: `
				DROP TABLE IF EXISTS items;
				DROP TABLE IF EXISTS payment;
				DROP TABLE IF EXISTS delivery;
				DROP TABLE IF EXISTS orders;
			`,
		},
		{
			Version: 2,
			Name:    "002_add_indexes",
			UpSQL: `
				CREATE INDEX IF NOT EXISTS idx_orders_customer_id ON orders(customer_id);
				CREATE INDEX IF NOT EXISTS idx_orders_date_created ON orders(date_created);
				CREATE INDEX IF NOT EXISTS idx_items_order_uid ON items(order_uid);
				CREATE INDEX IF NOT EXISTS idx_payment_order_uid ON payment(order_uid);
			`,
			DownSQL: `
				DROP INDEX IF EXISTS idx_orders_customer_id;
				DROP INDEX IF EXISTS idx_orders_date_created;
				DROP INDEX IF EXISTS idx_items_order_uid;
				DROP INDEX IF EXISTS idx_payment_order_uid;
			`,
		},
	}
}
