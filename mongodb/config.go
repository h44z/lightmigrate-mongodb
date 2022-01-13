package mongodb

import "time"

const DefaultMigrationsCollection = "schema_migrations"
const DefaultLockingCollection = "migrate_advisory_lock" // the collection to use for advisory locking by default.
const lockKeyUniqueValue = 0                             // the unique value to lock on. If multiple clients try to insert the same key, it will fail (locked).
const DefaultLockIndexName = "lock_unique_key"           // the default name of the index which adds unique constraint to the locking_key field.
const contextWaitTimeout = 5 * time.Second               // how long to wait for the request to mongo to block/wait for.

type config struct {
	DatabaseName         string
	MigrationsCollection string
	TransactionMode      bool
	Locking              LockingConfig
}

type LockingConfig struct {
	CollectionName string
	IndexName      string
	Enabled        bool
}
