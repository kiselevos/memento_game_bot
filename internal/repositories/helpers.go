package repositories

import (
	"database/sql"
	"fmt"
)

// ensureRowsAffected возвращает ошибку, если UPDATE/DELETE не затронул ни одной строки.
func ensureRowsAffected(res sql.Result, notFoundMsg string) error {
	n, err := res.RowsAffected()
	if err != nil {
		return nil
	}
	if n == 0 {
		return fmt.Errorf(notFoundMsg)
	}
	return nil
}

// nullifyEmpty возвращает nil при пустой строке
func nullifyEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
