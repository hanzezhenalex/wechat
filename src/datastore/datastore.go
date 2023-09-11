package datastore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/hanzezhenalex/wechat/src"

	_ "github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"
)

const (
	confirmedStr         = "confirmed"
	waitingForConfirmStr = "waitingForConfirm"
	deniedStr            = "denied"
	autoDeniedStr        = "autoDenied"
	unknownStr           = "unknown"
)

const (
	unknown RecordStatus = iota + 1
	autoDenied
	denied
	waitingForConfirm
	confirmed
)

type RecordStatus int

func (rs RecordStatus) String() string {
	switch rs {
	case autoDenied:
		return autoDeniedStr
	case denied:
		return deniedStr
	case waitingForConfirm:
		return waitingForConfirmStr
	case confirmed:
		return confirmedStr
	}
	return unknownStr
}

func ToRecordStatus(s string) RecordStatus {
	switch s {
	case autoDeniedStr:
		return autoDenied
	case deniedStr:
		return denied
	case waitingForConfirmStr:
		return waitingForConfirm
	case confirmedStr:
		return confirmed
	}
	return unknown
}

var datastoreTracer = logrus.WithField("comp", "datastore")

type Record struct {
	ID        int       `json:"id"`
	Hash      string    `json:"hash"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"createdAt"`
	GraphUrl  string    `json:"graph_url"`
	Status    string    `json:"status"`
}

type RecordQueryOption struct {
	from, to             time.Time
	minStatus, maxStatus string
}

type DataStore interface {
	CreateRecordAndCheckHash(ctx context.Context, record Record) (bool, error)
	GetRecordsByLeader(ctx context.Context, leader string, option RecordQueryOption) ([]Record, error)
	GetRecordsByUser(ctx context.Context, user string, option RecordQueryOption) ([]Record, error)

	CreateUser(ctx context.Context, username, leader string) error

	Close() error
}

type MysqlDatastore struct {
	db *sql.DB
}

func NewMysqlDatastore(ctx context.Context, cfg src.Config, cleanup bool) (*MysqlDatastore, error) {
	timeout := 10 * time.Minute
	datastoreCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	datastoreTracer.Infof("creating mysql datastore, timeout=%s", timeout.String())

	db, err := sql.Open("mysql", cfg.Dns())
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
			    id INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
			    hash VARCHAR(250) NOT NULL,
			    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, 
			    /* 
			     * https://stackoverflow.com/questions/219569/best-database-field-type-for-a-url
			     * Lowest common denominator max URL length among popular web browsers: 2,083
			     */
			    graph_url VARCHAR(2083) NOT NULL,  
			    username VARCHAR(64) NOT NULL,
			    status INT DEFAULT 2,
			    reserve1 VARCHAR(128) DEFAULT NULL,
 			    reserve2 VARCHAR(128) DEFAULT NULL, 
			    CONSTRAINT FOREIGN KEY(username) REFERENCES users(username),
			    INDEX (hash)
			) ENGINE=Innodb DEFAULT CHARACTER SET=utf8;
		`
		createTableUsers = `
			CREATE TABLE IF NOT EXISTS users(
			    username VARCHAR(64) NOT NULL PRIMARY KEY,
			    leader VARCHAR(64),
			    active BOOLEAN DEFAULT true,
			    reserve VARCHAR(128)
			) ENGINE=Innodb DEFAULT CHARACTER SET=utf8;
		`
	)

	if cleanup {
		datastoreTracer.Info("clean up tables")
		if _, err := store.db.ExecContext(ctx, `DROP TABLE IF EXISTS records;`); err != nil {
			return fmt.Errorf("fail to drop records table, %w", err)
		}
		if _, err := store.db.ExecContext(ctx, `DROP TABLE IF EXISTS users;`); err != nil {
			return fmt.Errorf("fail to drop users table, %w", err)
		}
	}

	datastoreTracer.Debug("creating users table")
	if _, err := store.db.ExecContext(ctx, createTableUsers); err != nil {
		return fmt.Errorf("fail to create users table, %w", err)
	}

	datastoreTracer.Debug("creating records table")
	if _, err := store.db.ExecContext(ctx, createTableRecords); err != nil {
		return fmt.Errorf("fail to create records table, %w", err)
	}
	return nil
}

func (store *MysqlDatastore) GetRecordsByLeader(ctx context.Context, leader string, option RecordQueryOption) ([]Record, error) {
	const (
		queryByLeader = `
			SELECT 
			    target_records.id AS id,
			    target_records.username AS username, 
			    target_records.graph_url AS graph_url, 
			    target_records.created_at AS created_at,
				target_records.status AS status
			FROM 
			    (
					SELECT
						username
					FROM
						users
					WHERE 
						leader = ?
				)	AS target_users
			LEFT JOIN 
				(
				    SELECT
				        id, username, graph_url, created_at, status
				    FROM 
				        records
				    WHERE 
				        status >= ?
				      	AND status <= ?
				        AND created_at >= ?
						AND created_at <= ?
				) AS target_records
			ON
				target_records.username = target_users.username
			WHERE 
			    target_records.graph_url IS NOT NULL
		`
	)

	rows, err := store.db.QueryContext(ctx, queryByLeader, leader, ToRecordStatus(option.minStatus),
		ToRecordStatus(option.maxStatus), option.from, option.to)
	if err != nil {
		return nil, fmt.Errorf("fail to query db, %w", err)
	}

	var records []Record
	for rows.Next() {
		var username, graphUrl string
		var createdAt time.Time
		var status RecordStatus
		var id int

		if err := rows.Scan(&id, &username, &graphUrl, &createdAt, &status); err != nil {
			return nil, fmt.Errorf("fail scan records from rows, %w", err)
		}
		records = append(records, Record{
			ID:        id,
			CreatedAt: createdAt,
			Username:  username,
			GraphUrl:  graphUrl,
			Status:    status.String(),
		})
	}
	return records, nil
}

func (store *MysqlDatastore) GetRecordsByUser(ctx context.Context, user string, option RecordQueryOption) ([]Record, error) {
	const (
		queryByUser = `
			SELECT
				username, graph_url, created_at, status
			FROM 
				records
			WHERE 
				status >= ?
				AND status <= ?
				AND created_at >= ?
				AND created_at <= ?
				AND username = ?
		`
	)

	rows, err := store.db.QueryContext(ctx, queryByUser, ToRecordStatus(option.minStatus),
		ToRecordStatus(option.maxStatus), option.from, option.to, user)
	if err != nil {
		return nil, fmt.Errorf("fail to query db, %w", err)
	}

	var records []Record
	for rows.Next() {
		var username, graphUrl string
		var createdAt time.Time
		var status RecordStatus

		if err := rows.Scan(&username, &graphUrl, &createdAt, &status); err != nil {
			return nil, fmt.Errorf("fail scan records from rows, %w", err)
		}
		records = append(records, Record{
			CreatedAt: createdAt,
			Username:  username,
			GraphUrl:  graphUrl,
			Status:    status.String(),
		})
	}
	return records, nil
}

func (store *MysqlDatastore) CreateRecordAndCheckHash(ctx context.Context, record Record) (bool, error) {
	const insertRecord = `
		INSERT INTO 
			records(hash, graph_url, username, status)
		VALUES 
		    (?, ?, ?, ?)
	`

	const insertRecordWhenHashNotExist = `
		INSERT INTO 
			records(hash, graph_url, username, status)
		SELECT 
			?, ?, ?, ?
		FROM 
			DUAL
		WHERE NOT EXISTS (SELECT 1 FROM records WHERE hash=?)
	`

	status := ToRecordStatus(record.Status)
	if status == waitingForConfirm {
		rows, err := store.db.ExecContext(ctx, insertRecordWhenHashNotExist, record.Hash, record.GraphUrl, record.Username, status, record.Hash)
		if err != nil {
			return false, fmt.Errorf("fail to exec insert sql, %w", err)
		}
		affected, _ := rows.RowsAffected()
		if affected > 0 {
			return true, nil
		} else {
			status = autoDenied
		}
	}
	_, err := store.db.ExecContext(ctx, insertRecord, record.Hash, record.GraphUrl, record.Username, status)
	if err != nil {
		return false, fmt.Errorf("fail to exec insert sql, %w", err)
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
