package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/redis/go-redis/v9"
)

func checkInstallation(ctx context.Context, config *koanf.Koanf) error {
	var failures []error
	if len(config.String("app.encryption_key")) != 32 || config.String("app.encryption_key") == sampleEncKey {
		failures = append(failures, fmt.Errorf("app.encryption_key is missing or still uses the sample. Generate it with openssl rand -hex 16 and keep it unchanged after setup"))
	}
	if err := checkUploadDirectory(config); err != nil {
		failures = append(failures, err)
	}
	db, err := sqlx.Open("postgres", databaseDSN(config)+" connect_timeout=3")
	if err == nil {
		defer db.Close()
		checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err = db.PingContext(checkCtx)
		cancel()
	}
	if err != nil {
		failures = append(failures, fmt.Errorf("PostgreSQL is unavailable. Check db.host, db.port, db.user, db.password, and db.database: %w", err))
	} else {
		installed, err := checkSchema(db)
		if err != nil {
			failures = append(failures, fmt.Errorf("database schema cannot be read. Check the database user's permissions: %w", err))
		} else if !installed {
			failures = append(failures, fmt.Errorf("database schema is missing. Run libredesk --install against an empty database"))
		} else {
			if _, pending, err := getPendingMigrations(db); err != nil {
				failures = append(failures, fmt.Errorf("database migrations cannot be checked: %w", err))
			} else if len(pending) > 0 {
				failures = append(failures, fmt.Errorf("database upgrades are pending. Back up your database, then run libredesk --upgrade"))
			}
			var rootURL string
			if err := db.Get(&rootURL, `SELECT value #>> '{}' FROM settings WHERE key = 'app.root_url'`); err != nil {
				failures = append(failures, fmt.Errorf("public address cannot be read from settings: %w", err))
			} else {
				config.Set("app.root_url", rootURL)
			}
		}
	}
	options := &redis.Options{Addr: config.String("redis.address"), Username: config.String("redis.user"), Password: config.String("redis.password"), DB: config.Int("redis.db")}
	if redisURL := config.String("redis.url"); redisURL != "" {
		options, err = redis.ParseURL(redisURL)
	}
	if err != nil && options == nil {
		failures = append(failures, fmt.Errorf("redis.url is invalid. Correct its connection URL"))
	} else {
		options.DialTimeout = 3 * time.Second
		options.MaxRetries = -1
		client := redis.NewClient(options)
		checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		redisErr := client.Ping(checkCtx).Err()
		cancel()
		client.Close()
		if redisErr != nil {
			failures = append(failures, fmt.Errorf("Redis is unavailable. Check redis.address, redis.password, and redis.url: %w", redisErr))
		}
	}
	if err := checkCookieConfiguration(config); err != nil {
		failures = append(failures, err)
	}
	if len(failures) == 0 {
		fmt.Println("Configuration and dependency checks passed. Local upload storage was checked when enabled.")
	}
	return errors.Join(failures...)
}

func checkUploadDirectory(config *koanf.Koanf) error {
	if config.String("upload.provider") != "fs" {
		return nil
	}
	dir := config.String("upload.fs.upload_path")
	if dir == "" {
		dir = "."
	}
	file, err := os.CreateTemp(dir, ".libredesk-write-check-*")
	if err != nil {
		return fmt.Errorf("upload folder %q is missing or not writable. Create it and give the libredesk process write permission: %w", dir, err)
	}
	name := file.Name()
	closeErr := file.Close()
	removeErr := os.Remove(name)
	return errors.Join(closeErr, removeErr)
}

func checkCookieConfiguration(config *koanf.Koanf) error {
	rootURL := config.String("app.root_url")
	if rootURL == "" {
		return nil
	}
	parsed, err := url.Parse(rootURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") {
		return fmt.Errorf("app.root_url is invalid. Set your full address, including https://")
	}
	local := parsed.Hostname() == "localhost" || parsed.Hostname() == "127.0.0.1" || parsed.Hostname() == "::1"
	if parsed.Scheme == "http" && !local && !config.Bool("app.server.disable_secure_cookies") {
		return fmt.Errorf("HTTP cannot use secure session cookies at this address. Configure HTTPS before signing in")
	}
	if parsed.Scheme == "https" && config.Bool("app.server.disable_secure_cookies") {
		return fmt.Errorf("secure session cookies are disabled on HTTPS. Set app.server.disable_secure_cookies to false")
	}
	return nil
}

func databaseDSN(config *koanf.Koanf) string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s %s",
		config.String("db.host"), config.Int("db.port"), config.String("db.user"), config.String("db.password"),
		config.String("db.database"), config.String("db.ssl_mode"), config.String("db.params"))
}
