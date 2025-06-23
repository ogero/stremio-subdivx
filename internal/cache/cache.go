package cache

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/dgraph-io/badger/v4"
)

var cacheDefaultPath = ".cache"

var badgerDB *badger.DB

func init() {

	var err error
	badgerDB, err = badger.Open(
		badger.DefaultOptions(cacheDefaultPath).
			WithNumVersionsToKeep(0).
			WithValueLogFileSize(1024 * 1024 * 100).
			WithLogger(&l{}),
	)
	if err != nil {
		log.Fatal("Failed to initialize cache:", err)
	}
}

// Memoize retrieves a cached value for the specified cacheKey.
// If the value is present and its type matches, it is returned. Otherwise, the provided function fn
// is called with cacheKey as its argument to compute the value, which is then stored in the cache
// with the specified expiration and returned. If the cached value has an unexpected type or if fn returns an error,
// Memoize returns the corresponding error.
func Memoize[V any](cacheKey string, ttl time.Duration, fn func(string) (*V, error)) (*V, error) {

	value := new(V)

	err := badgerDB.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(cacheKey))
		if err != nil {
			return err
		}

		err = item.Value(func(val []byte) error {
			return json.Unmarshal(val, value)
		})
		if err != nil {
			return fmt.Errorf("failed to json.Unmarshal: %w", err)
		}

		return nil
	})
	if err != nil && !errors.Is(err, badger.ErrKeyNotFound) {
		return nil, fmt.Errorf("failed to get from cache: %w", err)
	} else if err == nil {
		return value, nil
	}

	value, err = fn(cacheKey)
	if err != nil {
		return nil, err
	}

	err = badgerDB.Update(func(txn *badger.Txn) error {
		valueJSONBytes, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to json.Marshal: %w", err)
		}
		entry := badger.NewEntry([]byte(cacheKey), valueJSONBytes).WithTTL(ttl)
		return txn.SetEntry(entry)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to store on cache: %w", err)
	}

	return value, nil
}

// Close closes the cache DB. It's crucial to call it to ensure all the pending updates make their way to disk. Calling DB.Close() multiple times would still only close the DB once.
func Close() error {
	return badgerDB.Close()
}

type l struct{}

func (l *l) Errorf(s string, i ...interface{}) {
	log.Printf(s, i...)
}

func (l *l) Warningf(s string, i ...interface{}) {
	log.Printf(s, i...)
}

func (l *l) Infof(s string, i ...interface{}) {
	log.Printf(s, i...)
}

func (l *l) Debugf(s string, i ...interface{}) {
	log.Printf(s, i...)
}
