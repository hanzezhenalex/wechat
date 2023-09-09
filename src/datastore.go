package src

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"
)

const (
	defaultUsername  = "root"
	defaultPassword  = "sergey"
	defaultMysqlHost = "localhost"
	defaultMysqlPort = 3306
)

var datastoreTracer = logrus.WithField("comp", "datastore")

type Record struct {
	Hash      string    `json:"hash"`
	Username  string    `json:"username"`
	Leader    string    `json:"leader"`
	CreatedAt time.Time `json:"createdAt"`
}

type DataStore interface {
	GetRecord(ctx context.Context, user string, leader string, from, to time.Time) ([]Record, error)
	CreateRecordIfNotExist(ctx context.Context, record Record) (bool, error)
	Close() error
}

type MysqlDatastore struct {
	db *sql.DB
}

func NewMysqlDatastore(ctx context.Context) (*MysqlDatastore, error) {
	timeout := 10 * time.Minute
	datastoreCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	datastoreTracer.Infof("creating mysql datastore, timeout=%s", timeout.String())

	// "root:<password>@tcp(127.0.0.1:3306)/123begin"
	db, err := sql.Open("mysql", fmt.Sprintf("%s:<%s>@tcp(%s:%d)/",
		defaultUsername, defaultPassword, defaultMysqlHost, defaultMysqlPort))
	if err != nil {
		return nil, fmt.Errorf("fail to open mysql db: %w", err)
	}
	if err := db.PingContext(datastoreCtx); err != nil {
		return nil, fmt.Errorf("fail to ping mysql: %w", err)
	}

	store := &MysqlDatastore{db: db}
	if err := store.Prepare(datastoreCtx); err != nil {
		return nil, fmt.Errorf("fail to prepare mysql: %w", err)
	}
	return store, err
}

func (store *MysqlDatastore) Prepare(ctx context.Context) error {
	if err := store.PrepareDatabaseAndTables(ctx); err != nil {
		return fmt.Errorf("fail to prepare tables, %w", err)
	}

	if err := store.PrepareUsers(ctx); err != nil {
		return fmt.Errorf("fail to prepare users, %w", err)
	}
	return nil
}

func (store *MysqlDatastore) PrepareDatabaseAndTables(ctx context.Context) error {
	datastoreTracer.Info("start to prepare database")

	const (
		createDB           = `CREATE DATABASE IF NOT EXISTS wechat DEFAULT CHARACTER SET utf-8`
		createTableRecords = `
			CREATE TABLE IF NOT EXISTS records(
			    hash VARCHAR(250) NOT NULL PRIMARY KEY,
			    created_at VARCHAR(50) NOT NULL,
			    graph_url VARCHAR NOT NULL,
			    FOREIGN KEY (username) REFERENCES users(username),
			    FOREIGN KEY (leader) REFERENCES users(username),
			) ENGINE=Innodb DEFAULT CHARSET=utf8;
		`
		createTableUsers = `
			CREATE TABLE IF NOT EXISTS users(
			    username VARCHAR(50) NOT NULL PRIMARY KEY,
			    leader VARCHAR(50),
			    deleted BOOLEAN DEFAULT false,
			) ENGINE=Innodb DEFAULT CHARSET=utf8;
		`
	)

	if _, err := store.db.ExecContext(ctx, createDB); err != nil {
		return fmt.Errorf("fail to create database, %w", err)
	}
	if _, err := store.db.ExecContext(ctx, createTableUsers); err != nil {
		return fmt.Errorf("fail to create records table, %w", err)
	}
	if _, err := store.db.ExecContext(ctx, createTableRecords); err != nil {
		return fmt.Errorf("fail to create records table, %w", err)
	}
	return nil
}

func (store *MysqlDatastore) PrepareUsers(ctx context.Context) error {
	return nil
}

func (store *MysqlDatastore) GetRecordByLeader(ctx context.Context, leader string, from, to time.Time) ([]Record, error) {
	return nil, nil
}

func (store *MysqlDatastore) Close() error {
	return store.db.Close()
}
