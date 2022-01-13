package main

import (
	"context"
	"github.com/h44z/lightmigrate"
	"github.com/h44z/lightmigrate-mongodb/mongodb"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"log"
	"os"
	"time"
)

func main() {
	mongoClient, err := newMongoClient("mongodb://user:secret@123.123.123.123:27017/", 10*time.Second)
	if err != nil {
		log.Fatalf("unable to setup mongodb client: %v", err)
	}

	fsys := os.DirFS("examples")

	source, err := lightmigrate.NewFsSource(fsys, "example-migrations")
	if err != nil {
		log.Fatalf("unable to setup source: %v", err)
	}
	defer source.Close()

	driver, err := mongodb.NewDriver(mongoClient, "testdb",
		//mongodb.WithTransactions(true),
		mongodb.WithLocking(mongodb.LockingConfig{
			Enabled: true,
		}))
	if err != nil {
		log.Fatalf("unable to setup driver: %v", err)
	}
	defer driver.Close()

	migrator, err := lightmigrate.NewMigrator(source, driver, lightmigrate.WithVerboseLogging(true))
	if err != nil {
		log.Fatalf("unable to setup migrator: %v", err)
	}

	err = migrator.Migrate(2) // Migrate to schema version 2 (the only possible version in the example-migrations folder)
	if err != nil {
		log.Fatalf("migration error: %v", err)
	}
}

func newMongoClient(url string, timeout time.Duration) (*mongo.Client, error) {
	client, err := mongo.NewClient(options.Client().ApplyURI(url))
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		return nil, err
	}

	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return nil, err
	}

	return client, nil
}
