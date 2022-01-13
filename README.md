# LightMigrate - MongoDB migration driver

[![codecov](https://codecov.io/gh/h44z/lightmigrate-mongodb/branch/main/graph/badge.svg?token=N7H27SQUUW)](https://codecov.io/gh/h44z/lightmigrate-mongodb)
[![License: MIT](https://img.shields.io/badge/license-MIT-green.svg)](https://opensource.org/licenses/MIT)
[![GoDoc](https://pkg.go.dev/badge/github.com/h44z/lightmigrate-mongodb)](https://pkg.go.dev/github.com/h44z/lightmigrate-mongodb)
![GitHub last commit](https://img.shields.io/github/last-commit/h44z/lightmigrate-mongodb)
[![Go Report Card](https://goreportcard.com/badge/github.com/h44z/lightmigrate-mongodb)](https://goreportcard.com/report/github.com/h44z/lightmigrate-mongodb)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/h44z/lightmigrate-mongodb)
![GitHub code size in bytes](https://img.shields.io/github/languages/code-size/h44z/lightmigrate-mongodb)
[![GitHub Release](https://img.shields.io/github/release/h44z/lightmigrate-mongodb.svg)](https://github.com/h44z/lightmigrate-mongodb/releases)

This module is part of the [LightMigrate](https://github.com/h44z/lightmigrate) library.
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
