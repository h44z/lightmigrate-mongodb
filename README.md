# LightMigrate - MongoDB migration driver

This module is part of the [LightMigrate](github.com/h44z/lightmigrate) library.
It provides a migration driver for MongoDB.

## Features
 * Driver work with mongo through [db.runCommands](https://docs.mongodb.com/manual/reference/command/)
 * Migrations support json format. It contains array of commands for `db.runCommand`. Every command is executed in separate request to the database. 
 * All keys have to be in quotes `"`
 * [Examples](./examples)

## Configuration Options

Configuration options can be passed to the constructor using the `With<Config-Option>` functions.

| Config Value           | Defaults          | Description                                                                                                                         |
|------------------------|-------------------|-------------------------------------------------------------------------------------------------------------------------------------|
| `MigrationsCollection` | schema_migrations | Name of the migrations collection.                                                                                                  |
| `Transactions`         | false             | If set to `true` wrap commands in [transaction](https://docs.mongodb.com/manual/core/transactions). Available only for replica set. |
| `Locking`              | disabled / empty  | The locking configuration, see Locking Config table below.                                                                          |
| `Logger`               | log.Default()     | The logger instance that should be used.                                                                                            |
| `VerboseLogging`       | false             | If set to true, more log messages will be printed.                                                                                  |


| Locking Config Value   | Defaults              | Description                                          |
|------------------------|-----------------------|------------------------------------------------------|
| `MigrationsCollection` | migrate_advisory_lock | Name of the locking collection.                      |
| `IndexName`            | lock_unique_key       | Name of the unique index for the locking collection. |
| `Enabled`              | false                 | A boolean flag to enable the database locking.       |
