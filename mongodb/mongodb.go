package mongodb

import (
	"context"
	"fmt"
	"github.com/h44z/lightmigrate"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"io"
	"io/ioutil"
	"os"
	"sync/atomic"
	"time"
)

type versionInfo struct {
	Version int64 `bson:"version"`
	Dirty   bool  `bson:"dirty"`
}

type lockObj struct {
	Key       int       `bson:"locking_key"`
	Pid       int       `bson:"pid"`
	Hostname  string    `bson:"hostname"`
	CreatedAt time.Time `bson:"created_at"`
}

type lockFilter struct {
	Key int `bson:"locking_key"`
}

type driver struct {
	client   *mongo.Client
	cfg      *config
	migDb    *mongo.Database // where migration info is stored
	lockFlag int32           // must be accessed by atomic.XXX functions!

	logger  lightmigrate.Logger
	verbose bool
}

// DriverOption is a function that can be used within the driver constructor to
// modify the driver object.
type DriverOption func(svc *driver)

// NewDriver instantiates a new MongoDB driver. A MongoDB client and the database name are required arguments.
func NewDriver(client *mongo.Client, database string, opts ...DriverOption) (lightmigrate.MigrationDriver, error) {
	if database == "" {
		return nil, ErrNoDatabaseName
	}

	cfg := &config{
		DatabaseName:         database,
		MigrationsCollection: DefaultMigrationsCollection,
		TransactionMode:      false,
		Locking:              LockingConfig{}, // no locking
	}

	d := &driver{
		client: client,
		cfg:    cfg,
	}

	for _, opt := range opts {
		opt(d)
	}

	// setup migration database
	d.migDb = d.client.Database(d.cfg.DatabaseName)

	// setup locking
	if d.cfg.Locking.Enabled {
		err := d.prepareLockCollection()
		if err != nil {
			return nil, err
		}
	}

	return d, nil
}

// WithLogger sets the logging instance used by the driver.
func WithLogger(logger lightmigrate.Logger) DriverOption {
	return func(d *driver) {
		d.logger = logger
	}
}

// WithVerboseLogging sets the verbose flag of the driver.
func WithVerboseLogging(verbose bool) DriverOption {
	return func(d *driver) {
		d.verbose = verbose
	}
}

// WithMigrationCollection allows to specify the name of the collection that contains the migration state.
func WithMigrationCollection(migrationCollection string) DriverOption {
	return func(d *driver) {
		d.cfg.MigrationsCollection = migrationCollection
	}
}

// WithTransactions allows enabling or disabling MongoDB transactions for the migration process.
func WithTransactions(transactions bool) DriverOption {
	return func(d *driver) {
		d.cfg.TransactionMode = transactions
	}
}

// WithLocking can be used to configure the locking behaviour of the MongoDB migration driver.
// See LockingConfig for details.
func WithLocking(lockConfig LockingConfig) DriverOption {
	return func(d *driver) {
		if lockConfig.CollectionName == "" {
			lockConfig.CollectionName = DefaultLockingCollection
		}
		if lockConfig.IndexName == "" {
			lockConfig.IndexName = DefaultLockIndexName
		}

		d.cfg.Locking = lockConfig
	}
}

func (d *driver) Close() error {
	return nil // nothing to cleanup
}

// Lock utilizes advisory locking on the LockingConfig.CollectionName collection
// This uses a unique index on the `locking_key` field.
func (d *driver) Lock() error {
	if !d.cfg.Locking.Enabled {
		return nil
	}

	// check if already locked
	if atomic.LoadInt32(&d.lockFlag) == 1 {
		return nil
	}

	pid := os.Getpid()
	hostname, err := os.Hostname()
	if err != nil {
		hostname = fmt.Sprintf("unknown-host-%d", pid) // use pid as fallback
	}

	newLockObj := lockObj{
		Key:       lockKeyUniqueValue,
		Pid:       pid,
		Hostname:  hostname,
		CreatedAt: time.Now(),
	}

	ctx, cancelFunc := context.WithTimeout(context.Background(), contextWaitTimeout)
	defer cancelFunc()
	_, err = d.migDb.Collection(d.cfg.Locking.CollectionName).InsertOne(ctx, newLockObj)
	if err != nil {
		return ErrDatabaseLocked
	}

	atomic.StoreInt32(&d.lockFlag, 1)

	return nil
}

func (d *driver) Unlock() error {
	if !d.cfg.Locking.Enabled {
		return nil
	}

	// check if already unlocked
	if atomic.LoadInt32(&d.lockFlag) == 0 {
		return nil
	}

	filter := lockFilter{
		Key: lockKeyUniqueValue,
	}

	ctx, cancelFunc := context.WithTimeout(context.Background(), contextWaitTimeout)
	defer cancelFunc()
	_, err := d.migDb.Collection(d.cfg.Locking.CollectionName).DeleteMany(ctx, filter)
	if err != nil {
		return err
	}

	atomic.StoreInt32(&d.lockFlag, 0)

	return nil
}

func (d *driver) GetVersion() (version uint64, dirty bool, err error) {
	var versionInfo versionInfo
	err = d.migDb.Collection(d.cfg.MigrationsCollection).FindOne(context.TODO(), bson.M{}).Decode(&versionInfo)
	switch {
	case err == mongo.ErrNoDocuments:
		return lightmigrate.NoMigrationVersion, false, nil
	case err != nil:
		return 0, false, &lightmigrate.DriverError{OrigErr: err, Msg: "failed to get migration version"}
	default:
		return uint64(versionInfo.Version), versionInfo.Dirty, nil
	}
}

func (d *driver) SetVersion(version uint64, dirty bool) error {
	migrationsCollection := d.migDb.Collection(d.cfg.MigrationsCollection)
	if err := migrationsCollection.Drop(context.TODO()); err != nil {
		return &lightmigrate.DriverError{OrigErr: err, Msg: "drop migrations collection failed"}
	}
	_, err := migrationsCollection.InsertOne(context.TODO(), versionInfo{
		Version: int64(version),
		Dirty:   dirty,
	})
	if err != nil {
		return &lightmigrate.DriverError{OrigErr: err, Msg: "save version failed"}
	}
	return nil
}

func (d *driver) RunMigration(migration io.Reader) error {
	migr, err := ioutil.ReadAll(migration)
	if err != nil {
		return err
	}

	var cmds []bson.D
	err = bson.UnmarshalExtJSON(migr, true, &cmds)
	if err != nil {
		return fmt.Errorf("unmarshaling json error: %s", err)
	}
	if d.cfg.TransactionMode {
		if err := d.executeCommandsWithTransaction(context.TODO(), cmds); err != nil {
			return err
		}
	} else {
		if err := d.executeCommands(context.TODO(), cmds); err != nil {
			return err
		}
	}

	return nil
}

func (d *driver) Reset() error {
	migrationsCollection := d.migDb.Collection(d.cfg.MigrationsCollection)
	if err := migrationsCollection.Drop(context.TODO()); err != nil {
		return &lightmigrate.DriverError{OrigErr: err, Msg: "drop migrations collection failed"}
	}
	return nil
}

func (d *driver) executeCommandsWithTransaction(ctx context.Context, cmds []bson.D) error {
	err := d.client.UseSession(ctx, func(sessionContext mongo.SessionContext) error {
		if err := sessionContext.StartTransaction(); err != nil {
			return &lightmigrate.DriverError{OrigErr: err, Msg: "failed to start transaction"}
		}
		if err := d.executeCommands(sessionContext, cmds); err != nil {
			// When command execution failed, MongoDB has aborted the transaction
			// Calling abortTransaction will return an error that the transaction is already aborted
			return err
		}
		if err := sessionContext.CommitTransaction(sessionContext); err != nil {
			return &lightmigrate.DriverError{OrigErr: err, Msg: "failed to commit transaction"}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (d *driver) executeCommands(ctx context.Context, cmds []bson.D) error {
	for _, cmd := range cmds {
		err := d.migDb.RunCommand(ctx, cmd).Err()
		if err != nil {
			return &lightmigrate.DriverError{OrigErr: err, Msg: fmt.Sprintf("failed to execute command: %v", cmd)}
		}
	}
	return nil
}

// prepareLockCollection ensures that there exists a unique index for the locking key
func (d *driver) prepareLockCollection() error {
	indexes := d.migDb.Collection(d.cfg.Locking.CollectionName).Indexes()

	indexOptions := options.Index().SetUnique(true).SetName(d.cfg.Locking.IndexName)
	_, err := indexes.CreateOne(context.TODO(), mongo.IndexModel{
		Options: indexOptions,
		Keys:    lockFilter{Key: -1},
	})
	if err != nil {
		return err
	}
	return nil
}
