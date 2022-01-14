package mongodb

import (
	"bytes"
	"context"
	"github.com/h44z/lightmigrate"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"log"
	"sync/atomic"
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
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("Success", func(mt *mtest.T) {
		d, err := NewDriver(mt.Client, "test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expectedVersion := versionInfo{
			Version: 5,
			Dirty:   false,
		}

		mt.AddMockResponses(mtest.CreateCursorResponse(1, "test.schema_migrations", mtest.FirstBatch, bson.D{
			{Key: "_id", Value: primitive.NewObjectID()},
			{Key: "version", Value: expectedVersion.Version},
			{Key: "dirty", Value: expectedVersion.Dirty},
		}))
		version, dirty, err := d.GetVersion()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if version != uint64(expectedVersion.Version) {
			t.Fatalf("unexpected version: %d", version)
		}
		if dirty != expectedVersion.Dirty {
			t.Fatalf("unexpected dirty state: %t", dirty)
		}
	})

	mt.Run("EmptyDb", func(mt *mtest.T) {
		d, err := NewDriver(mt.Client, "test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		mt.AddMockResponses(mtest.CreateCursorResponse(0, "test.schema_migrations", mtest.FirstBatch))

		version, dirty, err := d.GetVersion()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if version != lightmigrate.NoMigrationVersion {
			t.Fatalf("unexpected version: %d", version)
		}
		if dirty {
			t.Fatalf("unexpected dirty state: %t", dirty)
		}
	})

	mt.Run("DbError", func(mt *mtest.T) {
		d, err := NewDriver(mt.Client, "test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		mt.AddMockResponses(mtest.CreateCommandErrorResponse(mtest.CommandError{
			Message: "Something is wrong",
			Code:    666,
		}))

		_, _, err = d.GetVersion()
		if err == nil {
			t.Fatalf("expected error, got: %v", err)
		}
	})
}

func Test_driver_Lock(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("Success", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateSuccessResponse()) // prepare lock table (index success response)

		d, err := NewDriver(mt.Client, "test", WithLocking(LockingConfig{Enabled: true}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		err = d.Lock()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// should be locked
		if atomic.LoadInt32(&d.(*driver).lockFlag) != 1 {
			t.Fatalf("not locked")
		}
	})

	mt.Run("Error", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateSuccessResponse()) // prepare lock table (index success response)

		d, err := NewDriver(mt.Client, "test", WithLocking(LockingConfig{Enabled: true}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		mt.AddMockResponses(mtest.CreateCommandErrorResponse(mtest.CommandError{
			Message: "Something is wrong",
			Code:    666,
		}))

		err = d.Lock()
		if err != ErrDatabaseLocked {
			t.Fatalf("expected ErrDatabaseLocked error, got: %v", err)
		}

		// should not be locked
		if atomic.LoadInt32(&d.(*driver).lockFlag) != 0 {
			t.Fatalf("unexpected lock")
		}
	})
}

func Test_driver_Lock_Disabled(t *testing.T) {
	d := driver{cfg: &config{}}
	err := d.Lock()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// should not be locked
	if atomic.LoadInt32(&d.lockFlag) == 1 {
		t.Fatalf("unexpected lock")
	}
}

func Test_driver_Lock_AlreadyLocked(t *testing.T) {
	d := driver{cfg: &config{Locking: LockingConfig{Enabled: true}}, lockFlag: 1}
	err := d.Lock()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// should be locked
	if atomic.LoadInt32(&d.lockFlag) != 1 {
		t.Fatalf("not locked")
	}
}

func Test_driver_Reset(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("Success", func(mt *mtest.T) {
		d, err := NewDriver(mt.Client, "test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		err = d.Reset()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	mt.Run("Error", func(mt *mtest.T) {
		d, err := NewDriver(mt.Client, "test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		mt.AddMockResponses(bson.D{{Key: "ok", Value: 0}})

		err = d.Reset()
		if err == nil {
			t.Fatalf("expected error, got: %v", err)
		}
	})
}

func Test_driver_RunMigration(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("NoTransactionsSuccess", func(mt *mtest.T) {
		d, err := NewDriver(mt.Client, "test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		err = d.(*driver).RunMigration(bytes.NewReader([]byte("[{}]")))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	mt.Run("NoTransactionsError", func(mt *mtest.T) {
		d, err := NewDriver(mt.Client, "test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		mt.AddMockResponses(bson.D{{Key: "ok", Value: 0}})

		err = d.(*driver).RunMigration(bytes.NewReader([]byte("[{}]")))
		if err == nil {
			t.Fatalf("expected error, got: %v", err)
		}
	})

	mt.Run("TransactionsSuccess", func(mt *mtest.T) {
		d, err := NewDriver(mt.Client, "test", WithTransactions(true))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		mt.AddMockResponses(mtest.CreateSuccessResponse())
		mt.AddMockResponses(mtest.CreateSuccessResponse()) // Commit

		err = d.(*driver).RunMigration(bytes.NewReader([]byte("[{}]")))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	mt.Run("TransactionsError", func(mt *mtest.T) {
		d, err := NewDriver(mt.Client, "test", WithTransactions(true))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		mt.AddMockResponses(bson.D{{Key: "ok", Value: 0}})

		err = d.(*driver).RunMigration(bytes.NewReader([]byte("[{}]")))
		if err == nil {
			t.Fatalf("expected error, got: %v", err)
		}
	})
}

func Test_driver_SetVersion(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("Success", func(mt *mtest.T) {
		d, err := NewDriver(mt.Client, "test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		mt.AddMockResponses(bson.D{{Key: "ok", Value: 1}, {Key: "acknowledged", Value: true}, {Key: "n", Value: 1}})
		mt.AddMockResponses(mtest.CreateSuccessResponse())

		err = d.SetVersion(5, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	mt.Run("DropError", func(mt *mtest.T) {
		d, err := NewDriver(mt.Client, "test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		mt.AddMockResponses(mtest.CreateCommandErrorResponse(mtest.CommandError{
			Message: "Something is wrong",
			Code:    666,
		}))

		err = d.SetVersion(5, false)
		if err == nil {
			t.Fatalf("expected error, got: %v", err)
		}
	})

	mt.Run("InsertError", func(mt *mtest.T) {
		d, err := NewDriver(mt.Client, "test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		mt.AddMockResponses(bson.D{{Key: "ok", Value: 1}, {Key: "acknowledged", Value: true}, {Key: "n", Value: 1}})
		mt.AddMockResponses(mtest.CreateCommandErrorResponse(mtest.CommandError{
			Message: "Something is wrong",
			Code:    666,
		}))

		err = d.SetVersion(5, false)
		if err == nil {
			t.Fatalf("expected error, got: %v", err)
		}
	})
}

func Test_driver_Unlock(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("Success", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateSuccessResponse()) // prepare lock table (index success response)

		d, err := NewDriver(mt.Client, "test", WithLocking(LockingConfig{Enabled: true}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		d.(*driver).lockFlag = 1 // simulate locked driver

		mt.AddMockResponses(bson.D{{Key: "ok", Value: 1}, {Key: "acknowledged", Value: true}, {Key: "n", Value: 1}}) // n = 1 doc deleted

		err = d.Unlock()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// should not be locked
		if atomic.LoadInt32(&d.(*driver).lockFlag) != 0 {
			t.Fatalf("unexptected lock")
		}
	})

	mt.Run("Error", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateSuccessResponse()) // prepare lock table (index success response)

		d, err := NewDriver(mt.Client, "test", WithLocking(LockingConfig{Enabled: true}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		d.(*driver).lockFlag = 1 // simulate locked driver

		mt.AddMockResponses(bson.D{{Key: "ok", Value: 0}, {Key: "acknowledged", Value: false}, {Key: "n", Value: 0}})

		err = d.Unlock()
		if err == nil {
			t.Fatalf("expected error, got: %v", err)
		}

		// should still be locked
		if atomic.LoadInt32(&d.(*driver).lockFlag) != 1 {
			t.Fatalf("not locked")
		}
	})
}

func Test_driver_Unlock_Disabled(t *testing.T) {
	d := driver{cfg: &config{}}
	err := d.Unlock()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// should not be locked
	if atomic.LoadInt32(&d.lockFlag) != 0 {
		t.Fatalf("unexptected lock")
	}
}

func Test_driver_Unlock_AlreadyUnlocked(t *testing.T) {
	d := driver{cfg: &config{Locking: LockingConfig{Enabled: true}}, lockFlag: 0}
	err := d.Unlock()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// should not be locked
	if atomic.LoadInt32(&d.lockFlag) != 0 {
		t.Fatalf("unexptected lock")
	}
}

func Test_driver_executeCommands(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("Success", func(mt *mtest.T) {
		d, err := NewDriver(mt.Client, "test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		mt.AddMockResponses(mtest.CreateSuccessResponse())
		mt.AddMockResponses(mtest.CreateSuccessResponse())

		err = d.(*driver).executeCommands(context.Background(), []bson.D{{}, {}})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	mt.Run("Error", func(mt *mtest.T) {
		d, err := NewDriver(mt.Client, "test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		mt.AddMockResponses(mtest.CreateSuccessResponse())
		mt.AddMockResponses(bson.D{{Key: "ok", Value: 0}}) // second one failed

		err = d.(*driver).executeCommands(context.Background(), []bson.D{{}, {}})
		if err == nil {
			t.Fatalf("expected error, got: %v", err)
		}
	})
}

func Test_driver_executeCommandsWithTransaction(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("Success", func(mt *mtest.T) {
		d, err := NewDriver(mt.Client, "test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		mt.AddMockResponses(mtest.CreateSuccessResponse())
		mt.AddMockResponses(mtest.CreateSuccessResponse())
		mt.AddMockResponses(mtest.CreateSuccessResponse()) // commit transaction

		err = d.(*driver).executeCommandsWithTransaction(context.Background(), []bson.D{{}, {}})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	mt.Run("ExecError", func(mt *mtest.T) {
		d, err := NewDriver(mt.Client, "test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		mt.AddMockResponses(bson.D{{Key: "ok", Value: 0}}) // first command failed

		err = d.(*driver).executeCommandsWithTransaction(context.Background(), []bson.D{{}, {}})
		if err == nil {
			t.Fatalf("expected error, got: %v", err)
		}
	})

	mt.Run("CommitError", func(mt *mtest.T) {
		d, err := NewDriver(mt.Client, "test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		mt.AddMockResponses(mtest.CreateSuccessResponse())
		mt.AddMockResponses(mtest.CreateSuccessResponse())
		mt.AddMockResponses(bson.D{{Key: "ok", Value: 0}}) // commit transaction error

		err = d.(*driver).executeCommandsWithTransaction(context.Background(), []bson.D{{}, {}})
		if err == nil {
			t.Fatalf("expected error, got: %v", err)
		}
	})

}

func Test_driver_prepareLockCollection(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("Success", func(mt *mtest.T) {
		d, err := NewDriver(mt.Client, "test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		mt.AddMockResponses(bson.D{{Key: "ok", Value: 1}, {Key: "acknowledged", Value: true}, {Key: "n", Value: 1}}) // n = 1 index created

		err = d.(*driver).prepareLockCollection()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	mt.Run("Error", func(mt *mtest.T) {
		d, err := NewDriver(mt.Client, "test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		mt.AddMockResponses(bson.D{{Key: "ok", Value: 0}, {Key: "acknowledged", Value: false}, {Key: "n", Value: 0}}) // n = 0 index created

		err = d.(*driver).prepareLockCollection()
		if err == nil {
			t.Fatalf("expected error, got: %v", err)
		}
	})
}
