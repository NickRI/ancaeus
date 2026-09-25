package cache

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	bolt "go.etcd.io/bbolt"
)

var errMiss = errors.New("cache miss")

type entry[T any] struct {
	Data      T         `json:"data"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Store is a TTL key-value cache backed by bbolt.
type Store[T any] struct {
	db     *bolt.DB
	bucket string
	ttl    time.Duration
}

func NewStore[T any](db *bolt.DB, bucket string, ttl time.Duration) (*Store[T], error) {
	err := db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucket))
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("create bucket: %w", err)
	}
	return &Store[T]{db: db, bucket: bucket, ttl: ttl}, nil
}

func (c *Store[T]) Set(key string, cacheData T) error {
	raw, err := json.Marshal(entry[T]{
		Data:      cacheData,
		ExpiresAt: time.Now().Add(c.ttl),
	})
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	return c.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(c.bucket))
		if b == nil {
			return fmt.Errorf("bucket %q missing", c.bucket)
		}
		return b.Put([]byte(key), raw)
	})
}

// Get returns (value, true, nil) on hit, (zero, false, nil) on miss/expiry,
// and (zero, false, err) on storage failures.
func (c *Store[T]) Get(key string) (T, bool, error) {
	var zero T
	var e entry[T]
	err := c.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(c.bucket))
		if b == nil {
			return fmt.Errorf("bucket %q missing", c.bucket)
		}
		data := b.Get([]byte(key))
		if data == nil {
			return errMiss
		}
		if err := json.Unmarshal(data, &e); err != nil {
			return fmt.Errorf("unmarshal: %w", err)
		}
		return nil
	})
	if errors.Is(err, errMiss) {
		return zero, false, nil
	}
	if err != nil {
		return zero, false, err
	}
	if time.Now().After(e.ExpiresAt) {
		return zero, false, nil
	}
	return e.Data, true, nil
}
