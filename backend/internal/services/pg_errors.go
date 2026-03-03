package services

import (
	"errors"

	"github.com/lib/pq"
)

// isPgUniqueViolation reports whether err is a PostgreSQL unique‑constraint
// violation (error code 23505).
func isPgUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505"
}
