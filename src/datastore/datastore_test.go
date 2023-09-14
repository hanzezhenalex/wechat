//go:build docker

package datastore

import (
	"context"
	"testing"
	"time"

	"github.com/hanzezhenalex/wechat/src"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func Zero(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func TestDataStore(t *testing.T) {
	rq := require.New(t)

	// TODO: read if from config?
	cfg := src.Config{
		DbConfig: src.DbConfig{
			Host:     "10.237.150.156",
			Port:     3306,
			Username: "sergey",
			Password: "sergey",
		},
	}

	logrus.SetLevel(logrus.DebugLevel)

	store, err := NewMysqlDataStore(cfg, true)
	rq.NoError(err)

	ctx := context.Background()

	t.Run("users", func(t *testing.T) {
		rq.NoError(store.CreateNewUser(ctx, UserInfo{
			WechatID: "id_1",
			Name:     "user_1",
		}))
		rq.NoError(store.CreateNewUser(ctx, UserInfo{
			WechatID: "id_2",
			Name:     "user_2",
		}))

		users, err := store.GetAllUsers(ctx)
		rq.NoError(err)
		rq.Equal(2, len(users))

		_, exist, err := store.GetUserById(ctx, "id_2")
		rq.NoError(err)
		rq.True(exist)

		_, exist, err = store.GetUserById(ctx, "id_3")
		rq.NoError(err)
		rq.False(exist)
	})

	t.Run("record", func(t *testing.T) {
		r1 := RecordInfo{
			OwnerID:  "id_1",
			Status:   waitingForConfirm,
			GraphUrl: "http://www.baidu.com",
		}

		// create record, success
		exist, err := store.CreateRecord(ctx, r1, "123", true)
		rq.False(exist)
		rq.NoError(err)

		// create record with duplicated md5,
		// record -> success
		// duplicated md5 -> exist = true
		exist, err = store.CreateRecord(ctx, r1, "123", true)
		rq.True(exist)
		rq.NoError(err)

		// create record with checkExist true
		exist, err = store.CreateRecord(ctx, r1, "123", false)
		rq.False(exist)
		rq.NoError(err)

		hashes, err := store.GetAllHashes(ctx, HashQueryOption{
			from: Zero(time.Now()),
			to:   time.Now(),
		})
		rq.NoError(err)
		rq.Equal(1, len(hashes))
	})
}
