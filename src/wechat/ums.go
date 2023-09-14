package wechat

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hanzezhenalex/wechat/src"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/hanzezhenalex/wechat/src/datastore"
	"github.com/sirupsen/logrus"
)

var umsTracer = func(ctx context.Context) *logrus.Entry {
	return logrus.WithField("comp", "ums").WithContext(ctx)
}

type item struct {
	wg *sync.WaitGroup
}

type UserMngr struct {
	// cache through
	cache sync.Map // wechat_id -> User
	store datastore.DataStore

	// once a time for each user req, blocking others
	updating map[string]*item
	mutex    sync.Mutex
}

func NewUMS(store datastore.DataStore) (*UserMngr, error) {
	usm := &UserMngr{
		store:    store,
		updating: make(map[string]*item),
	}
	users, err := store.GetAllUsers(context.Background())
	if err != nil {
		return nil, fmt.Errorf("fail to get all users from datastore, %w", err)
	}

	// cache all users
	for _, user := range users {
		usm.cache.Store(user.WechatID, user)
	}
	return usm, nil
}

func (ums *UserMngr) CreateNewUser(ctx context.Context, user datastore.UserInfo) error {
	key := user.WechatID

	_, loaded := ums.cache.Load(key)
	if loaded {
		return fmt.Errorf("user %s exists", key)
	}

	wg := &sync.WaitGroup{}
	ums.mutex.Lock()
	if _, ok := ums.updating[key]; ok {
		ums.mutex.Unlock()
		return fmt.Errorf("user %s is creating", key)
	}
	ums.updating[key] = &item{wg: wg}
	ums.mutex.Unlock()

	wg.Add(1)
	defer func() {
		ums.mutex.Lock()
		delete(ums.updating, key)
		ums.mutex.Unlock()

		wg.Done()
	}()

	if err := ums.store.CreateNewUser(ctx, user); err != nil {
		return fmt.Errorf("fail to create user %s in datastore, %w", key, err)
	}

	ums.cache.Store(key, user)

	return nil
}

func (ums *UserMngr) GetUserById(_ context.Context, id string) (datastore.UserInfo, bool) {
	val, loaded := ums.cache.Load(id)
	if !loaded {
		return datastore.UserInfo{}, false
	}
	return val.(datastore.UserInfo), true
}

func (ums *UserMngr) RegisterEndpoints(group *gin.RouterGroup) {
	group.POST("/create", func(context *gin.Context) {
		ctx := context.Request.Context()
		tracer := umsTracer(ctx)

		auth := context.Request.Header.Get("x-alex-auth")
		if auth != src.DefaultApiToken {
			tracer.Warningf("req rejected, invalid auth token")
			context.Writer.WriteHeader(http.StatusUnauthorized)
			return
		}

		var user datastore.UserInfo
		if err := json.NewDecoder(context.Request.Body).Decode(&user); err != nil {
			tracer.Errorf("fail to decode req body, %s", err.Error())
			_, _ = context.Writer.Write([]byte(err.Error()))
			context.Writer.WriteHeader(http.StatusInternalServerError)
		}

		if err := ums.CreateNewUser(ctx, user); err != nil {
			tracer.Errorf("fail to create new user, %s", err.Error())
			_, _ = context.Writer.Write([]byte(err.Error()))
			context.Writer.WriteHeader(http.StatusInternalServerError)
		}

		tracer.Infof("new user created, id=%s, name=%s", user.WechatID, user.Name)
		context.Writer.WriteHeader(http.StatusOK)
	})
}
