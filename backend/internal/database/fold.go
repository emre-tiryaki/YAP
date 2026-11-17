package database

import (
	"database/sql/driver"
	"strings"

	"modernc.org/sqlite"
)

// FoldFunction is the SQL function that case-folds text the same way Fold
// does. SQLite's own lower() and LIKE only fold ASCII letters, so searches use
// this function to match "Ödev" with "ödev".
const FoldFunction = "yap_fold"

// Fold case-folds s for case-insensitive comparisons.
func Fold(s string) string {
	return strings.ToLower(s)
}

func init() {
	err := sqlite.RegisterDeterministicScalarFunction(FoldFunction, 1,
		func(_ *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
			switch v := args[0].(type) {
			case string:
				return Fold(v), nil
			case []byte:
				return Fold(string(v)), nil
			default:
				return v, nil
			}
		})
	if err != nil {
		panic(err)
	}
}
