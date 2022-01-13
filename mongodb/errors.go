package mongodb

import "fmt"

var (
	ErrNoDatabaseName = fmt.Errorf("no database name")
	ErrDatabaseLocked = fmt.Errorf("database is locked")
)
