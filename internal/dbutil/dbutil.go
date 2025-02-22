package dbutil

import "github.com/lib/pq"

// IsForeignKeyError checks if the given error is a PostgreSQL foreign key violation (error code 23503)
func IsForeignKeyError(err error) bool {
	if err == nil {
		return false
	}
	if pqErr, ok := err.(*pq.Error); ok {
		return pqErr.Code == "23503"
	}
	return false
}

// IsUniqueViolationError checks if the given error is a PostgreSQL unique violation (error code 23505)
func IsUniqueViolationError(err error) bool {
	if err == nil {
		return false
	}
	if pqErr, ok := err.(*pq.Error); ok {
		return pqErr.Code == "23505"
	}
	return false
}
