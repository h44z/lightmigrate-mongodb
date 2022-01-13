package mongodb

import "fmt"

var (
	// ErrNoDatabaseName signals a missing database name.
	ErrNoDatabaseName = fmt.Errorf("no database name")
	// ErrNoDatabaseClient signals a missing database client.
	ErrNoDatabaseClient = fmt.Errorf("no database client")
	// ErrDatabaseLocked signals that the database is already locked by another migration process.
	ErrDatabaseLocked = fmt.Errorf("database is locked")
)
