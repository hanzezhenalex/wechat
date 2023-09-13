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

var datastoreTracer = logrus.WithField("comp", "datastore")

type DataStore interface {
	CreateRecordAndCheckIfHashExist(ctx context.Context, record Record) (bool, error)
	GetRecordsByLeader(ctx context.Context, id string, option RecordQueryOption) ([]Record, error)
	GetRecordsByUser(ctx context.Context, id string, option RecordQueryOption) ([]Record, error)

	CreateUser(ctx context.Context, new User) error
	GetAllUsers(ctx context.Context) ([]User, error)

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
			    user_wechat_id VARCHAR(128) NOT NULL,
			    status INT DEFAULT 2,
			    reserve1 VARCHAR(128) DEFAULT NULL,
 			    reserve2 VARCHAR(128) DEFAULT NULL, 
			    CONSTRAINT FOREIGN KEY(user_wechat_id) REFERENCES users(wechat_id),
			    INDEX (hash)
			) ENGINE=Innodb DEFAULT CHARACTER SET=utf8;
		`
		createTableUsers = `
			CREATE TABLE IF NOT EXISTS users(
			    wechat_id VARCHAR(128) NOT NULL PRIMARY KEY,
			    username VARCHAR(64),
			    leader_wechat_id VARCHAR(64),
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

func (store *MysqlDatastore) Close() error {
	return store.db.Close()
}

/*
 *
 * CURD for Record
 *
 */

type Record struct {
	ID           int       `json:"id"`
	Hash         string    `json:"hash"`
	UserWechatId string    `json:"user_wechat_id"`
	CreatedAt    time.Time `json:"createdAt"`
	GraphUrl     string    `json:"graph_url"`
	Status       string    `json:"status"`
}

func NewRecord(hash string, id string, url string) Record {
	return Record{
		Hash:         hash,
		UserWechatId: id,
		GraphUrl:     url,
		Status:       waitingForConfirmStr,
	}
}

type RecordQueryOption struct {
	from, to             time.Time
	minStatus, maxStatus string
}

const (
	confirmedStr         = "confirmed"
	waitingForConfirmStr = "waitingForConfirm"
	deniedStr            = "denied"
	autoDeniedStr        = "autoDenied"
	unknownStr           = "unknown"
)

type RecordStatus int

const (
	unknown RecordStatus = iota + 1
	autoDenied
	denied
	waitingForConfirm
	confirmed
)

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

func (store *MysqlDatastore) GetRecordsByLeader(ctx context.Context, leaderId string, option RecordQueryOption) ([]Record, error) {
	const (
		queryByLeader = `
			SELECT 
			    target_records.id AS id,
			    target_records.user_wechat_id AS user_wechat_id, 
			    target_records.graph_url AS graph_url, 
			    target_records.created_at AS created_at,
				target_records.status AS status
			FROM 
			    (
					SELECT
						wechat_id
					FROM
						users
					WHERE 
						leader_wechat_id = ?
				)	AS target_users
			LEFT JOIN 
				(
				    SELECT
				        id, user_wechat_id, graph_url, created_at, status
				    FROM 
				        records
				    WHERE 
				        status >= ?
				      	AND status <= ?
				        AND created_at >= ?
						AND created_at <= ?
				) AS target_records
			ON
				target_records.user_wechat_id = target_users.wechat_id
			WHERE 
			    target_records.graph_url IS NOT NULL
		`
	)

	rows, err := store.db.QueryContext(ctx, queryByLeader, leaderId, ToRecordStatus(option.minStatus),
		ToRecordStatus(option.maxStatus), option.from, option.to)
	if err != nil {
		return nil, fmt.Errorf("fail to query db, %w", err)
	}

	var records []Record
	for rows.Next() {
		var userWechatId, graphUrl string
		var createdAt time.Time
		var status RecordStatus
		var id int

		if err := rows.Scan(&id, &userWechatId, &graphUrl, &createdAt, &status); err != nil {
			return nil, fmt.Errorf("fail scan records from rows, %w", err)
		}
		records = append(records, Record{
			ID:           id,
			CreatedAt:    createdAt,
			UserWechatId: userWechatId,
			GraphUrl:     graphUrl,
			Status:       status.String(),
		})
	}
	return records, nil
}

func (store *MysqlDatastore) GetRecordsByUser(ctx context.Context, id string, option RecordQueryOption) ([]Record, error) {
	const (
		queryByUser = `
			SELECT
				user_wechat_id, graph_url, created_at, status
			FROM 
				records
			WHERE 
				status >= ?
				AND status <= ?
				AND created_at >= ?
				AND created_at <= ?
				AND user_wechat_id = ?
		`
	)

	rows, err := store.db.QueryContext(ctx, queryByUser, ToRecordStatus(option.minStatus),
		ToRecordStatus(option.maxStatus), option.from, option.to, id)
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
			CreatedAt:    createdAt,
			UserWechatId: username,
			GraphUrl:     graphUrl,
			Status:       status.String(),
		})
	}
	return records, nil
}

func (store *MysqlDatastore) CreateRecordAndCheckIfHashExist(ctx context.Context, record Record) (bool, error) {
	const insertRecord = `
		INSERT INTO 
			records(hash, graph_url, user_wechat_id, status)
		VALUES 
		    (?, ?, ?, ?)
	`

	const insertRecordWhenHashNotExist = `
		INSERT INTO 
			records(hash, graph_url, user_wechat_id, status)
		SELECT 
			?, ?, ?, ?
		FROM 
			DUAL
		WHERE NOT EXISTS (SELECT 1 FROM records WHERE hash=?)
	`

	status := ToRecordStatus(record.Status)
	duplicated := false

	if status == waitingForConfirm {
		rows, err := store.db.ExecContext(ctx, insertRecordWhenHashNotExist, record.Hash, record.GraphUrl, record.UserWechatId, status, record.Hash)
		if err != nil {
			return false, fmt.Errorf("fail to exec insert sql, %w", err)
		}
		affected, _ := rows.RowsAffected()
		if affected > 0 {
			return true, nil
		} else {
			status = autoDenied
			duplicated = true
		}
	}
	_, err := store.db.ExecContext(ctx, insertRecord, record.Hash, record.GraphUrl, record.UserWechatId, status)
	if err != nil {
		return false, fmt.Errorf("fail to exec insert sql, %w", err)
	}
	return !duplicated, nil
}

/*
 *
 * CURD for Users
 *
 */

type User struct {
	Username string `json:"username"`
	WechatId string `json:"wechat_id"`
	LeaderId string `json:"leader_id"`
	Active   bool   `json:"active"`
}

func (store *MysqlDatastore) CreateUser(ctx context.Context, user User) error {
	const insertUser = `
		INSERT IGNORE INTO 
			users(username, leader_wechat_id, wechat_id)
		VALUES 
		    (?, ?, ?)
	`
	rows, err := store.db.ExecContext(ctx, insertUser, user.Username, user.LeaderId, user.WechatId)
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

func (store *MysqlDatastore) GetUserByWechatId(ctx context.Context, id string) (User, error) {
	var user User
	const query = `
		SELECT 
			username, wechat_id, leader_wechat_id
		FROM
			users
		WHERE
			active = true
		    AND wechat_id = ?
	`
	row := store.db.QueryRowContext(ctx, query, id)
	if row.Err() != nil {
		return user, fmt.Errorf("fail to query user, %w", row.Err())
	}
	if err := row.Scan(&user.Username, &user.WechatId, &user.LeaderId); err != nil {
		return user, fmt.Errorf("fail to read row from user query result, %w", row.Err())
	}
	return user, nil
}

func (store *MysqlDatastore) GetAllUsers(ctx context.Context) ([]User, error) {
	const query = `
		SELECT 
			username, leader_wechat_id, wechat_id
		FROM
			users
		WHERE
			active = true
	`
	rows, err := store.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("fail to query all user, %w", err)
	}

	var users []User
	var username, leaderWechatId, wechatId string
	for rows.Next() {
		if err := rows.Scan(&username, &leaderWechatId, &wechatId); err != nil {
			return nil, fmt.Errorf("fail scan users from rows, %w", err)
		}
		users = append(users, User{
			Username: username,
			WechatId: wechatId,
			LeaderId: leaderWechatId,
		})
	}
	return users, nil
}
