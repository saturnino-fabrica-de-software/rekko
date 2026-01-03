package repository

import "strings"

// isUniqueViolation checks if the error is a unique constraint violation
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}

	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "23505") ||
		strings.Contains(errMsg, "unique") ||
		strings.Contains(errMsg, "duplicate key")
}
