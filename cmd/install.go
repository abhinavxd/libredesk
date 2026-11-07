package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	auth_ "github.com/abhinavxd/libredesk/internal/auth"
	"github.com/abhinavxd/libredesk/internal/colorlog"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/user"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/stuffbin"
)

// install preserves an existing schema unless --reset-database explicitly allows replacement.
func install(ctx context.Context, db *sqlx.DB, fs stuffbin.FileSystem, idempotentInstall, prompt bool) error {
	schemaInstalled, err := checkSchema(db)
	if err != nil {
		return fmt.Errorf("checking database schema: %w", err)
	}
	reset := ko.Bool("reset-database")
	if schemaInstalled && !reset {
		log.Println("database already installed. Existing data is unchanged. Use --upgrade to apply migrations.")
		return nil
	}
	if !reset {
		var occupied bool
		if err := db.GetContext(ctx, &occupied, `SELECT EXISTS (SELECT 1 FROM pg_tables WHERE schemaname = current_schema())`); err != nil {
			return fmt.Errorf("checking database contents: %w", err)
		}
		if occupied {
			return fmt.Errorf("database contains tables. Installation stopped to preserve existing data. Use an empty database or explicitly pass --reset-database")
		}
	}

	// Make sure the system user password is strong enough.
	password := os.Getenv("LIBREDESK_SYSTEM_USER_PASSWORD")
	if password != "" && !user.IsStrongPassword(password) {
		return fmt.Errorf("system user password is not strong, %s", user.PasswordHint)
	}

	if reset {
		colorlog.Red(fmt.Sprintf("WARNING: This will wipe your entire database - '%s'", ko.String("db.database")))
	}

	if prompt && (!idempotentInstall || reset) {
		log.Print("Continue (y/n)? ")
		var ok string
		fmt.Scanf("%s", &ok)
		if !strings.EqualFold(ok, "y") {
			return fmt.Errorf("installation cancelled")
		}
	}

	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := installSchema(tx, fs); err != nil {
		return fmt.Errorf("installing database schema: %w", err)
	}

	if err := user.CreateSystemUser(ctx, password, tx); err != nil {
		return fmt.Errorf("creating system user: %w", err)
	}
	if err := initializeSetup(tx, password == ""); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	log.Println("database schema installed successfully")
	return nil
}

// setSystemUserPass prompts for pass and sets system user password.
func setSystemUserPass(ctx context.Context, db *sqlx.DB) {
	rd := initRedis()
	defer rd.Close()
	id, err := user.ChangeSystemUserPassword(ctx, db)
	if err != nil {
		log.Fatalf("error changing system user password: %v", err)
	}
	if err := auth_.DestroyUserSessions(ctx, rd, id); err != nil {
		log.Fatalf("error destroying system user sessions: %v", err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE settings SET value = '{"required":false}' WHERE key = 'setup.bootstrap'`); err != nil {
		log.Fatalf("error closing browser setup: %v", err)
	}
	fmt.Println("password updated successfully. Login with email 'System' and the new password.")
}

// checkSchema verifies if the DB schema is already installed by querying a table.
func checkSchema(db *sqlx.DB) (bool, error) {
	if _, err := db.Exec(`SELECT 1 FROM settings LIMIT 1`); err != nil {
		if dbutil.IsTableNotExistError(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// installSchema reads the schema file and installs it in the database.
func installSchema(db sqlx.Ext, fs stuffbin.FileSystem) error {
	q, err := fs.Read("/schema.sql")
	if err != nil {
		return err
	}
	_, err = db.Exec(string(q))
	return err
}
