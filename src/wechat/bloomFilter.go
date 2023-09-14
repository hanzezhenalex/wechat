package wechat

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hanzezhenalex/wechat/src/datastore"

	"github.com/bits-and-blooms/bloom/v3"
)

const (
	periodContainsInFilter = 4 * 7 * 24 * time.Hour
)

type BloomFilter struct {
	store datastore.DataStore

	mutex  sync.Mutex
	filter *bloom.BloomFilter
}

func newOriginBloomFilter() *bloom.BloomFilter {
	return bloom.NewWithEstimates(1000, 0.01)
}

func NewBloomFilter(store datastore.DataStore) (*BloomFilter, error) {
	bf := &BloomFilter{
		store: store,
	}
	filter, err := bf.build()
	if err != nil {
		return nil, fmt.Errorf("fail to build filter, %w", err)
	}
	bf.filter = filter
	return bf, nil
}

func (bf *BloomFilter) TestAndAdd(key string) bool {
	bf.mutex.Lock()
	defer bf.mutex.Unlock()

	if bf.filter.TestString(key) {
		return true
	}

	bf.filter.AddString(key)
	return false
}

func (bf *BloomFilter) build() (*bloom.BloomFilter, error) {
	hashes, err := bf.store.GetAllHashes(
		context.Background(),
		datastore.NewHashQueryOption(time.Now().Add(-1*periodContainsInFilter), time.Now()),
	)
	if err != nil {
		return nil, fmt.Errorf("fail to get all hashes from datastore, %w", err)
	}

	filter := newOriginBloomFilter()
	for _, record := range hashes {
		filter.AddString(record.MD5)
	}
	return filter, nil
}
