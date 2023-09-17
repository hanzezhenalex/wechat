package datastore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hanzezhenalex/wechat/src"

	_ "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type DataStore interface {
	CreateNewUser(ctx context.Context, user UserInfo) error
	GetAllUsers(ctx context.Context) ([]UserInfo, error)
	GetUserById(ctx context.Context, id string) (UserInfo, bool, error)

	CreateRecord(ctx context.Context, record RecordInfo, md5 string, checkExist bool) (existed bool, err error)

	GetAllHashes(ctx context.Context, option HashQueryOption) ([]Hash, error)
}

type UserInfo struct {
	WechatID string    `gorm:"column:wechat_id;size:256;primaryKey;not null" json:"wechat_id"`
	Name     string    `gorm:"size:128;not null" json:"name"`
	LeaderID string    `gorm:"column:leader_id;size:128" json:"leader_id"`
	Active   bool      `gorm:"default:true" json:"active"`
	CreateAt time.Time `gorm:"type:TIMESTAMP;default:CURRENT_TIMESTAMP;<-:create" json:"create_at"`
	Reserve  string    `gorm:"size:256" json:",omitempty"`
}

type RecordInfo struct {
	ID        int          `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UpdatedBy string       `gorm:"size:256" json:"updated_by"`
	OwnerID   string       `gorm:"column:owner_id;size:256;not null" json:"owner_id"`
	Status    RecordStatus `gorm:"not null" json:"status"`
	GraphUrl  string       `gorm:"not null" json:"graph_url"`
	CreateAt  time.Time    `gorm:"type:TIMESTAMP;default:CURRENT_TIMESTAMP;<-:create" json:"create_at"`
	UpdatedAt time.Time    `gorm:"type:TIMESTAMP;default:CURRENT_TIMESTAMP on update current_timestamp" json:"updated_at"`
	Reserve1  string       `gorm:"size:256" json:",omitempty"`
	Reserve2  string       `gorm:"size:256" json:",omitempty"`
}

func NewRecordInfo(ownerID, status, graphUrl string) (RecordInfo, error) {
	var record RecordInfo

	rStatus, err := RecordStatusFromString(status)
	if err != nil {
		return record, fmt.Errorf("illeagal record status, %w", err)
	}

	record.Status = rStatus
	record.OwnerID = ownerID
	record.GraphUrl = graphUrl
	return record, nil
}

type RecordStatus int

func RecordStatusFromString(status string) (RecordStatus, error) {
	switch status {
	case Confirmed:
		return confirmed, nil
	case WaitingForConfirm:
		return waitingForConfirm, nil
	case Denied:
		return denied, nil
	case AutoDenied:
		return autoDenied, nil
	}
	return autoDenied, fmt.Errorf("unknown status %s", status)
}

const (
	Confirmed         = "confirmed"
	WaitingForConfirm = "waitingForConfirm"
	Denied            = "denied"
	AutoDenied        = "autoDenied"

	autoDenied = iota + 1
	denied
	waitingForConfirm
	confirmed
)

type Hash struct {
	MD5      string `gorm:"column:md5;size:512;primaryKey;not null" json:"md5"`
	RecordID int    `gorm:"column:record_id" json:"record_id"`
	Reserve  string `gorm:"size:256" json:",omitempty"`
}

type mysqlDataStore struct {
	db *gorm.DB
}

func NewMysqlDataStore(cfg src.Config, cleanup bool) (*mysqlDataStore, error) { // cleanup -> only for test
	db, err := gorm.Open(mysql.Open(cfg.Dns()), &gorm.Config{
		Logger: Logger{slowThreshold: 500 * time.Millisecond},
	})
	if err != nil {
		return nil, fmt.Errorf("fail to connect to mysql, %w", err)
	}

	store := &mysqlDataStore{db: db}
	if cleanup {
		if err := store.cleanup(); err != nil {
			return nil, fmt.Errorf("fail to clean up tables, %w", err)
		}
	}
	if err := db.AutoMigrate(&UserInfo{}, &RecordInfo{}, &Hash{}); err != nil {
		return nil, fmt.Errorf("fail to migrate tables, %w", err)
	}
	return store, nil
}

func (store *mysqlDataStore) cleanup() error {
	var result *gorm.DB
	const drop = "DROP TABLE IF EXISTS %s"
	if result = store.db.Exec(fmt.Sprintf(drop, "user_infos")); result.Error != nil {
		return fmt.Errorf("fail to clean up table UserInfo, %w", result.Error)
	}
	if result = store.db.Exec(fmt.Sprintf(drop, "record_infos")); result.Error != nil {
		return fmt.Errorf("fail to clean up table RecordInfo, %w", result.Error)
	}
	if result = store.db.Exec(fmt.Sprintf(drop, "hashes")); result.Error != nil {
		return fmt.Errorf("fail to clean up table Hash, %w", result.Error)
	}
	return nil
}

/*
 * CURD for users
 */

func (store *mysqlDataStore) CreateNewUser(ctx context.Context, user UserInfo) error {
	result := store.db.WithContext(ctx).Create(&user)
	return result.Error
}

func (store *mysqlDataStore) GetAllUsers(ctx context.Context) ([]UserInfo, error) {
	var users []UserInfo
	result := store.db.WithContext(ctx).Find(&users)
	return users, result.Error
}

func (store *mysqlDataStore) GetUserById(ctx context.Context, id string) (UserInfo, bool, error) {
	var user UserInfo
	result := store.db.WithContext(ctx).Where("wechat_id=?", id).First(&user)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return user, false, nil
	}
	return user, true, result.Error
}

/*
 * CURD for records
 */

type RecordQueryOption struct {
	from, to               time.Time
	minorStatus, maxStatus RecordStatus
}

func NewRecordQueryOption(from, to time.Time, minorStatus, maxStatus string) (RecordQueryOption, error) {
	var op RecordQueryOption
	min, err := RecordStatusFromString(minorStatus)
	if err != nil {
		return op, fmt.Errorf("illeagal minor record status, %w", err)
	}
	max, err := RecordStatusFromString(maxStatus)
	if err != nil {
		return op, fmt.Errorf("illeagal max record status, %w", err)
	}
	op.minorStatus = min
	op.maxStatus = max
	op.from = from
	op.to = to
	return op, nil
}

// CreateRecord WARNING: MUST NOT reply on "existed" when set "checkExist" to false
func (store *mysqlDataStore) CreateRecord(ctx context.Context, record RecordInfo, md5 string, _ bool) (existed bool, err error) {
	db := store.db.WithContext(ctx)

	// check md5 and set status accordingly
	tx := db.Begin()
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	result := tx.Clauses(clause.Insert{Modifier: "IGNORE"}).Create(&Hash{MD5: md5})
	if result.Error != nil {
		return false, fmt.Errorf("fail to insert hash, %w", result.Error)
	}

	// check if duplicated
	if result.RowsAffected == 0 {
		existed = true
		// set status if duplicated
		record.Status = autoDenied
	}

	// insert record
	if result := tx.Create(&record); result.Error != nil {
		err = fmt.Errorf("fail to insert record, %w", result.Error)
		return
	}
	return
}

/*
 * CURD for hash
 */

type HashQueryOption struct {
	from, to time.Time
}

func NewHashQueryOption(from, to time.Time) HashQueryOption {
	return HashQueryOption{from: from, to: to}
}

func (store *mysqlDataStore) GetAllHashes(ctx context.Context, option HashQueryOption) ([]Hash, error) {
	var hashes []Hash
	// TODO: not gorm style? limit?
	if result := store.db.WithContext(ctx).Raw(`
select 
	distinct hashes.md5 as md5
from 
	(select id from record_infos where status >= 0 and create_at > ? and create_at < ? ) as records
	left join hashes
	on hashes.record_id = records.id`,
		option.from, option.to).Scan(&hashes); result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("fail to get all hashes, %w", result.Error)
	}
	return hashes, nil
}
