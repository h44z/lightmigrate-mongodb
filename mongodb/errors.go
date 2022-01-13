package mongodb

import "fmt"

var (
	// ErrNoDatabaseName signals a missing database name.
	ErrNoDatabaseName = fmt.Errorf("no database name")
	// ErrDatabaseLocked signals that the database is already locked by another migration process.
	ErrDatabaseLocked = fmt.Errorf("database is locked")
)
