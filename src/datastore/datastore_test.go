//go:build docker

package datastore

import (
	"context"
	"testing"
	"time"

	"github.com/hanzezhenalex/wechat/src"

	"github.com/stretchr/testify/require"
)

func Zero(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func TestMysqlDatastore(t *testing.T) {
	rq := require.New(t)
	ctx := context.Background()

	cfg := src.Config{
		DbConfig: src.DbConfig{
			Host:     "10.237.150.156",
			Port:     3306,
			Username: "sergey",
			Password: "sergey",
		},
	}

	mysql, err := NewMysqlDatastore(ctx, cfg, true)
	rq.NoError(err)

	rq.NoError(mysql.CreateUser(ctx, User{
		WechatId: "leader_1",
	}))
	rq.NoError(mysql.CreateUser(ctx, User{
		WechatId: "user_1",
		Leader:   "leader_1",
	}))
	rq.NoError(mysql.CreateUser(ctx, User{
		WechatId: "user_2",
		Leader:   "leader_1",
	}))

	var records = []Record{
		{
			UserWechatId: "leader_1",
			Hash:         "hash0",
			GraphUrl:     "http://www.baidu.com",
			Status:       confirmedStr,
		},
		{
			UserWechatId: "user_1",
			Hash:         "hash1",
			GraphUrl:     "http://www.baidu.com",
		},
		{
			UserWechatId: "user_1",
			Hash:         "hash2",
			GraphUrl:     "http://www.baidu.com",
		},
		{
			UserWechatId: "user_2",
			Hash:         "hash3",
			GraphUrl:     "http://www.baidu.com",
		},
		{
			UserWechatId: "user_2",
			Hash:         "hash4",
			GraphUrl:     "http://www.baidu.com",
		},
	}

	for _, record := range records {
		n, err := mysql.CreateRecordAndCheckHash(ctx, record)
		rq.NoError(err)
		rq.True(n)
	}

	t.Run("get record by leader", func(t *testing.T) {
		now := time.Now()
		records, err := mysql.GetRecordsByLeader(ctx, "leader_1", RecordQueryOption{
			minStatus: unknownStr,
			maxStatus: confirmedStr,
			from:      Zero(now),
			to:        now,
		})
		rq.NoError(err)
		rq.Len(records, 4)
	})
}
