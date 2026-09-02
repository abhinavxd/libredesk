package dbutil

import (
	"errors"

	"github.com/lib/pq"
)

// IsForeignKeyError checks if the given error is a PostgreSQL foreign key violation (error code 23503)
func IsForeignKeyError(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23503"
}

// IsUniqueViolationError checks if the given error is a PostgreSQL unique violation (error code 23505)
func IsUniqueViolationError(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505"
}

// IsTableNotExistError checks if the given error is a PostgreSQL table does not exist error (error code 42P01)
func IsTableNotExistError(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "42P01"
}

// PageOffset returns the row offset for a 1-based page, 0 when pageSize is 0 (fetch all).
func PageOffset(page, pageSize int) int {
	if pageSize <= 0 || page < 1 {
		return 0
	}
	return (page - 1) * pageSize
}
