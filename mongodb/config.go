package mongodb

import "time"

// DefaultMigrationsCollection is the collection to use for migration state by default.
const DefaultMigrationsCollection = "schema_migrations"

// DefaultLockingCollection is the collection to use for advisory locking by default.
const DefaultLockingCollection = "migrate_advisory_lock"

// lockKeyUniqueValue is the unique value to lock on. If multiple clients try to insert the same key, it will fail (locked).
const lockKeyUniqueValue = 0

// DefaultLockIndexName is the default name of the index which adds unique constraint to the locking_key field.
const DefaultLockIndexName = "lock_unique_key"

// contextWaitTimeout describes how long to wait for the request to mongo to block/wait for.
const contextWaitTimeout = 5 * time.Second

type config struct {
	DatabaseName         string
	MigrationsCollection string
	TransactionMode      bool
	Locking              LockingConfig
}

// LockingConfig can be used to configure the locking behaviour of the MongoDB migration driver.
type LockingConfig struct {
	// CollectionName is the collection name where the lock object will be stored. Defaults to DefaultLockingCollection.
	CollectionName string
	// IndexName is the name of the unique index that is required for the locking process.
	// Defaults to DefaultLockIndexName.
	IndexName string
	// Enabled flag can be used to enable or disable locking, by default it is disabled.
	Enabled bool
}
