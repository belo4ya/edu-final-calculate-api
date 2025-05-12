package sqlz

import "database/sql"

func Some[T any](v T) sql.Null[T] {
	return sql.Null[T]{V: v, Valid: true}
}
