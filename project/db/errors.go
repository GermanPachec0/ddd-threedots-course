package db

import (
	"errors"

	"github.com/lib/pq"
)

const (
	postgresUniqueValueViolationErrorCode = "23505"
)

func isErrorUniqueViolation(err error) bool {
	var psqlErr *pq.Error
	return errors.As(err, &psqlErr) && psqlErr.Code == postgresUniqueValueViolationErrorCode
}
