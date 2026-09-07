package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	bolt "go.etcd.io/bbolt"
)

type cacheEntry[T any] struct {
	Data      T         `json:"data"`
	ExpiresAt time.Time `json:"expires_at"` // в миллисекундах
}

type Cache[T any] struct {
	db     *bolt.DB
	bucket string
	ttl    time.Duration
}

func NewCache[T any](db *bolt.DB, bucket string, ttl time.Duration) (*Cache[T], error) {
	err := db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucket))
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create bucket: %w", err)
	}

	return &Cache[T]{db: db, bucket: bucket, ttl: ttl}, nil
}

func (c *Cache[T]) Set(key string, cacheData T) error {
	entry := cacheEntry[T]{
		Data:      cacheData,
		ExpiresAt: time.Now().Add(c.ttl),
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal entry: %w", err)
	}

	return c.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(c.bucket))
		return b.Put([]byte(key), data)
	})
}

func (c *Cache[T]) Get(key string) (T, bool, error) {
	var (
		zero  T
		entry cacheEntry[T]
	)

	err := c.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(c.bucket))

		data := b.Get([]byte(key))
		if data == nil {
			return errors.New("key not found")
		}

		return json.Unmarshal(data, &entry)
	})

	if err != nil {
		return zero, false, nil // just skip
	}

	// check ttl
	if time.Now().After(entry.ExpiresAt) {
		return zero, false, nil
	}

	return entry.Data, true, nil
}
