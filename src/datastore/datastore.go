package datastore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"
)

const (
	defaultDatabase = "wechat"

	defaultUsername  = "sergey"
	defaultPassword  = "sergey"
	defaultMysqlHost = "localhost"
	defaultMysqlPort = 3306
)

var (
	datastoreTracer    = logrus.WithField("comp", "datastore")
	DefaultMysqlConfig = Config{
		Username: defaultUsername,
		Password: defaultPassword,
		Host:     defaultMysqlHost,
		Port:     defaultMysqlPort,
	}
)

type Config struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
}

func (cfg Config) dns() string {
	// "username:password@tcp(host:post)/dbname"
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&loc=Local",
		cfg.Username, cfg.Password, cfg.Host, cfg.Port, defaultDatabase)
}

type Record struct {
	Hash      string    `json:"hash"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"createdAt"`
	GraphUrl  string    `json:"graph_url"`
}

type DataStore interface {
	GetRecordByLeader(ctx context.Context, leader string, from, to time.Time) ([]Record, error)
	CreateRecordIfNotExist(ctx context.Context, record Record) (bool, error)
	CreateUser(ctx context.Context, username, leader string) error
	Close() error
}

type MysqlDatastore struct {
	db *sql.DB
}

func NewMysqlDatastore(ctx context.Context, cfg Config, cleanup bool) (*MysqlDatastore, error) {
	timeout := 10 * time.Minute
	datastoreCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	datastoreTracer.Infof("creating mysql datastore, timeout=%s", timeout.String())

	db, err := sql.Open("mysql", cfg.dns())
	if err != nil {
		return nil, fmt.Errorf("fail to open mysql db: %w", err)
	}

	datastoreTracer.Debug("ping db")
	if err := db.PingContext(datastoreCtx); err != nil {
		return nil, fmt.Errorf("fail to ping mysql: %w", err)
	}

	store := &MysqlDatastore{db: db}
	if err := store.prepareTables(datastoreCtx, cleanup); err != nil {
		return nil, fmt.Errorf("fail to prepare mysql: %w", err)
	}
	datastoreTracer.Info("mysql datastore created successfully")
	return store, nil
}

func (store *MysqlDatastore) prepareTables(ctx context.Context, cleanup bool) error {
	datastoreTracer.Debug("start to prepare database")

	const (
		createTableRecords = `
			CREATE TABLE IF NOT EXISTS records(
			    hash VARCHAR(250) NOT NULL PRIMARY KEY,
			    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6), 
			    /* 
			     * https://stackoverflow.com/questions/219569/best-database-field-type-for-a-url
			     * Lowest common denominator max URL length among popular web browsers: 2,083
			     */
			    graph_url VARCHAR(2083) NOT NULL,  
			    username VARCHAR(50) NOT NULL,
			    CONSTRAINT FOREIGN KEY(username) REFERENCES users(username)
			) ENGINE=Innodb DEFAULT CHARACTER SET=utf8;
		`
		createTableUsers = `
			CREATE TABLE IF NOT EXISTS users(
			    username VARCHAR(50) NOT NULL PRIMARY KEY,
			    leader VARCHAR(50),
			    deleted BOOLEAN DEFAULT false
			) ENGINE=Innodb DEFAULT CHARACTER SET=utf8;
		`
	)

	if cleanup {
		datastoreTracer.Info("clean up tables")
		if _, err := store.db.ExecContext(ctx, `DROP TABLE records`); err != nil {
			return fmt.Errorf("fail to drop records table, %w", err)
		}
		if _, err := store.db.ExecContext(ctx, `DROP TABLE users`); err != nil {
			return fmt.Errorf("fail to drop users table, %w", err)
		}
	}

	datastoreTracer.Debug("creating users table")
	if _, err := store.db.ExecContext(ctx, createTableUsers); err != nil {
		return fmt.Errorf("fail to create records table, %w", err)
	}

	datastoreTracer.Debug("creating records table")
	if _, err := store.db.ExecContext(ctx, createTableRecords); err != nil {
		return fmt.Errorf("fail to create records table, %w", err)
	}
	return nil
}

func (store *MysqlDatastore) GetRecordByLeader(ctx context.Context, leader string, from, to time.Time) ([]Record, error) {
	const (
		queryByLeader = `
			SELECT 
			    target_records.username AS username, 
			    target_records.graph_url AS graph_url, 
			    target_records.created_at AS created_at
			FROM 
			    (
					SELECT
						username
					FROM
						users AS u
					WHERE 
						leader = ?
				)	AS target_users
			LEFT JOIN 
				(
				    SELECT
				        username, graph_url, created_at
				    FROM 
				        records
				    WHERE 
				        created_at >= ?
						AND created_at <= ?
				) AS target_records
			ON
				target_records.username = target_users.username
			WHERE 
			    target_records.graph_url IS NOT NULL
		`
	)
	rows, err := store.db.QueryContext(ctx, queryByLeader, leader, from, to)
	if err != nil {
		return nil, fmt.Errorf("fail to query db, %w", err)
	}

	var records []Record
	for rows.Next() {
		var username, graphUrl string
		var createdAt time.Time

		if err := rows.Scan(&username, &graphUrl, &createdAt); err != nil {
			return nil, fmt.Errorf("fail scan records from rows, %w", err)
		}
		records = append(records, Record{
			CreatedAt: createdAt,
			Username:  username,
			GraphUrl:  graphUrl,
		})
	}
	return records, nil
}

func (store *MysqlDatastore) CreateRecordIfNotExist(ctx context.Context, record Record) (bool, error) {
	const insertRecord = `
		INSERT IGNORE INTO 
			records(hash, graph_url, username)
		VALUES 
		    (?, ?, ?)
	`

	rows, err := store.db.ExecContext(ctx, insertRecord, record.Hash, record.GraphUrl, record.Username)
	if err != nil {
		return false, fmt.Errorf("fail to exec insert sql, %w", err)
	}

	if affected, err := rows.RowsAffected(); err != nil {
		return false, fmt.Errorf("fail to read insert result, %w", err)
	} else if affected == 0 {
		return false, nil
	}
	return true, nil
}

func (store *MysqlDatastore) CreateUser(ctx context.Context, username, leader string) error {
	const insertUser = `
		INSERT IGNORE INTO 
			users(username, leader)
		VALUES 
		    (?, ?)
	`
	rows, err := store.db.ExecContext(ctx, insertUser, username, leader)
	if err != nil {
		return fmt.Errorf("fail to insert user, %w", err)
	}
	if affected, err := rows.RowsAffected(); err != nil {
		return fmt.Errorf("fail to read insert result, %w", err)
	} else if affected == 0 {
		return errors.New("user exists")
	}
	return nil
}

func (store *MysqlDatastore) Close() error {
	return store.db.Close()
}
