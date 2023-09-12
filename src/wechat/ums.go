package wechat

import (
	"context"
	"fmt"
	"sync"

	"github.com/hanzezhenalex/wechat/src/datastore"
)

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
		usm.cache.Store(user.WechatId, user)
	}
	return usm, nil
}

func (ums *UserMngr) CreateNewUser(ctx context.Context, user datastore.User) error {
	key := user.WechatId

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

	if err := ums.store.CreateUser(ctx, user); err != nil {
		return fmt.Errorf("fail to create user %s in datastore, %w", key, err)
	}

	ums.cache.Store(key, user)

	return nil
}

func (ums *UserMngr) GetUserById(_ context.Context, id string) (datastore.User, bool) {
	val, loaded := ums.cache.Load(id)
	if !loaded {
		return datastore.User{}, false
	}
	return val.(datastore.User), true
}
