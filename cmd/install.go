package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/colorlog"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/migrations"
	"github.com/abhinavxd/libredesk/internal/user"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/stuffbin"
)

// Install checks if the schema is already installed, prompts for confirmation, and installs the schema if needed.
// idempotent install skips the installation if the database schema is already installed.
func install(ctx context.Context, db *sqlx.DB, fs stuffbin.FileSystem, idempotentInstall, prompt bool) error {
	schemaInstalled, err := checkSchema(db)
	if err != nil {
		log.Fatalf("error checking existing db schema: %v", err)
	}

	// Make sure the system user password is strong enough.
	password := os.Getenv("LIBREDESK_SYSTEM_USER_PASSWORD")
	if password != "" && !user.IsStrongPassword(password) && !schemaInstalled {
		log.Fatalf("system user password is not strong, %s", user.PasswordHint)
	}

	if !idempotentInstall {
		log.Println("running first time setup...")
		colorlog.Red(fmt.Sprintf("WARNING: This will wipe your entire database - '%s'", ko.String("db.database")))
	}

	if prompt {
		log.Print("Continue (y/n)? ")
		var ok string
		fmt.Scanf("%s", &ok)
		if !strings.EqualFold(ok, "y") {
			log.Fatalf("installation cancelled")
		}
	}

	if idempotentInstall {
		if schemaInstalled {
			log.Println("skipping installation as schema is already installed")
			os.Exit(0)
		}
	} else {
		log.Println("installing database schema...")
		time.Sleep(5 * time.Second)
	}

	// Install schema.
	if err := installSchema(db, fs); err != nil {
		log.Fatalf("error installing schema: %v", err)
	}

	// Run post-schema migrations so fresh installs include columns not yet in schema.sql.
	if err := migrations.V2_7_0(db, fs, ko); err != nil {
		log.Fatalf("error running v2.7.0 migration: %v", err)
	}

	log.Println("database schema installed successfully")

	// Create system user.
	if err := user.CreateSystemUser(ctx, password, db); err != nil {
		log.Fatalf("error creating system user: %v", err)
	}
	return nil
}

// setSystemUserPass prompts for pass and sets system user password.
func setSystemUserPass(ctx context.Context, db *sqlx.DB) {
	user.ChangeSystemUserPassword(ctx, db)
}

// checkSchema verifies if the DB schema is already installed by querying a table.
func checkSchema(db *sqlx.DB) (bool, error) {
	if _, err := db.Exec(`SELECT * FROM settings LIMIT 1`); err != nil {
		if dbutil.IsTableNotExistError(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// installSchema reads the schema file and installs it in the database.
func installSchema(db *sqlx.DB, fs stuffbin.FileSystem) error {
	q, err := fs.Read("/schema.sql")
	if err != nil {
		return err
	}
	_, err = db.Exec(string(q))
	return err
}
