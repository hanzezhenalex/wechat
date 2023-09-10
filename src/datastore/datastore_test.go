//go:build docker

package datastore

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Zero(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func TestMysqlDatastore(t *testing.T) {
	rq := require.New(t)
	ctx := context.Background()

	cfg := DefaultMysqlConfig
	cfg.Host = "10.237.150.156"

	mysql, err := NewMysqlDatastore(ctx, cfg, true)
	rq.NoError(err)

	rq.NoError(mysql.CreateUser(ctx, "leader_1", ""))
	rq.NoError(mysql.CreateUser(ctx, "user_1", "leader_1"))
	rq.NoError(mysql.CreateUser(ctx, "user_2", "leader_1"))

	var records = []Record{
		{
			Username: "leader_1",
			Hash:     "hash0",
			GraphUrl: "http://www.baidu.com",
		},
		{
			Username: "user_1",
			Hash:     "hash1",
			GraphUrl: "http://www.baidu.com",
		},
		{
			Username: "user_1",
			Hash:     "hash2",
			GraphUrl: "http://www.baidu.com",
		},
		{
			Username: "user_2",
			Hash:     "hash3",
			GraphUrl: "http://www.baidu.com",
		},
		{
			Username: "user_2",
			Hash:     "hash4",
			GraphUrl: "http://www.baidu.com",
		},
	}

	for _, record := range records {
		created, err := mysql.CreateRecordIfNotExist(ctx, record)
		rq.NoError(err)
		rq.True(created)
	}

	t.Run("get record by leader", func(t *testing.T) {
		now := time.Now()
		records, err := mysql.GetRecordByLeader(ctx, "leader_1", Zero(now), now)
		rq.NoError(err)
		rq.Len(records, 4)
	})

	t.Run("duplicate hash", func(t *testing.T) {
		created, err := mysql.CreateRecordIfNotExist(ctx, Record{
			Username: "user_2",
			Hash:     "hash4",
			GraphUrl: "http://www.baidu.com",
		})
		rq.NoError(err)
		rq.False(created)
	})

}
