package mongodb

import (
	"go.mongodb.org/mongo-driver/mongo"
	"log"
	"testing"
)

func TestNewDriver(t *testing.T) {
	_, err := NewDriver(&mongo.Client{}, "db", WithVerboseLogging(true))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewDriver_WithLocking(t *testing.T) {
	_, err := NewDriver(&mongo.Client{}, "db", WithLocking(LockingConfig{
		Enabled: true,
	}))
	if err == nil { // cannot prepare migration db
		t.Fatalf("expected error, got: %v", err)
	}
}

func TestNewDriver_NoDb(t *testing.T) {
	_, err := NewDriver(nil, "")
	if err == nil {
		t.Fatalf("expected error, got: %v", err)
	}
}

func TestNewDriver_NoClient(t *testing.T) {
	_, err := NewDriver(nil, "db")
	if err == nil {
		t.Fatalf("expected error, got: %v", err)
	}
}

func TestWithLocking(t *testing.T) {
	d := &driver{cfg: &config{}}
	lockCfg := LockingConfig{
		CollectionName: "a",
		IndexName:      "b",
		Enabled:        true,
	}

	WithLocking(lockCfg)(d)
	if d.cfg.Locking != lockCfg {
		t.Fatalf("failed to set lock config")
	}
}

func TestWithLogger(t *testing.T) {
	d := &driver{}

	WithLogger(log.Default())(d)
	if d.logger != log.Default() {
		t.Fatalf("failed to set logger")
	}
}

func TestWithMigrationCollection(t *testing.T) {
	d := &driver{cfg: &config{}}

	WithMigrationCollection("name")(d)
	if d.cfg.MigrationsCollection != "name" {
		t.Fatalf("failed to set migration collection name")
	}
}

func TestWithTransactions(t *testing.T) {
	d := &driver{cfg: &config{}}

	WithTransactions(true)(d)
	if d.cfg.TransactionMode != true {
		t.Fatalf("failed to set transaction mode")
	}
}

func TestWithVerboseLogging(t *testing.T) {
	d := &driver{}

	WithVerboseLogging(true)(d)
	if d.verbose != true {
		t.Fatalf("failed to set verbose flag")
	}
}

func Test_driver_Close(t *testing.T) {
	d := &driver{}
	err := d.Close()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func Test_driver_GetVersion(t *testing.T) {
	// TODO: implement
}

func Test_driver_Lock(t *testing.T) {
	// TODO: implement
}

func Test_driver_Reset(t *testing.T) {
	// TODO: implement
}

func Test_driver_RunMigration(t *testing.T) {
	// TODO: implement
}

func Test_driver_SetVersion(t *testing.T) {
	// TODO: implement
}

func Test_driver_Unlock(t *testing.T) {
	// TODO: implement
}

func Test_driver_executeCommands(t *testing.T) {
	// TODO: implement
}

func Test_driver_executeCommandsWithTransaction(t *testing.T) {
	// TODO: implement
}

func Test_driver_prepareLockCollection(t *testing.T) {
	// TODO: implement
}
