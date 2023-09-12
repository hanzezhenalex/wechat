package wechat

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/hanzezhenalex/wechat/src/datastore"

	"github.com/golang/mock/gomock"
	mock "github.com/hanzezhenalex/wechat/src/datastore/mocks"
	"github.com/stretchr/testify/require"
)

func TestUMS(t *testing.T) {
	rq := require.New(t)

	ctrl := gomock.NewController(t)
	store := mock.NewMockDataStore(ctrl)
	store.EXPECT().GetAllUsers(gomock.Any()).Return([]datastore.User{}, nil)

	ums, err := NewUMS(store)
	rq.NoError(err)

	t.Run("create new user", func(t *testing.T) {
		store.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Return(nil)

		rq.NoError(ums.CreateNewUser(context.Background(), datastore.User{
			WechatId: "id1",
		}))

		_, ok := ums.GetUserById(context.Background(), "id1")
		rq.True(ok)
	})

	t.Run("duplicated creation", func(t *testing.T) {
		store.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(func(ctx context.Context, new datastore.User) error {
			rq.Equal("id2", new.WechatId)
			return nil
		})
		// first one, success
		rq.NoError(ums.CreateNewUser(context.Background(), datastore.User{
			WechatId: "id2",
		}))
		// second one, fail
		rq.Error(ums.CreateNewUser(context.Background(), datastore.User{
			WechatId: "id2",
		}))
		rq.Equal(0, len(ums.updating))
	})

	t.Run("read write", func(t *testing.T) {
		store.EXPECT().CreateUser(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(_ context.Context, _ datastore.User) error {
			time.Sleep(time.Second)
			return nil
		})

		wg := sync.WaitGroup{}
		wg.Add(1)

		go func() {
			rq.NoError(ums.CreateNewUser(context.Background(), datastore.User{
				WechatId: "id3",
			}))
			wg.Done()
		}()

		_, ok := ums.GetUserById(context.Background(), "id3")
		rq.False(ok)

		wg.Wait()
		_, ok = ums.GetUserById(context.Background(), "id3")
		rq.True(ok)
	})
}
