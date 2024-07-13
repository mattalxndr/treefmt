package caching

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"github.com/adrg/xdg"
	"github.com/charmbracelet/log"
	"github.com/vmihailenco/msgpack/v5"
	bolt "go.etcd.io/bbolt"
)

type boltTx struct {
	tx *bolt.Tx
}

func (b boltTx) Commit() error {
	return b.tx.Commit()
}

func (b boltTx) Rollback() error {
	return b.tx.Rollback()
}

type BoltCache struct {
	db  *bolt.DB
	log *log.Logger

	readSize int
}

func (c BoltCache) BeginRead() (Tx, error) {
	tx, err := c.db.Begin(false)
	return boltTx{tx}, err
}

func (c BoltCache) View(f func(tx Tx) error) error {
	return c.db.View(func(tx *bolt.Tx) error {
		return f(&boltTx{tx})
	})
}

func (c BoltCache) Update(f func(tx Tx) error) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		return f(&boltTx{tx})
	})
}

func (c BoltCache) unwrapTx(tx Tx) (*bolt.Tx, error) {
	switch v := tx.(type) {
	case boltTx:
		return v.tx, nil
	default:
		return nil, fmt.Errorf("tx must be of type NoOpTx")
	}
}

func (c BoltCache) GetPath(tx Tx, path string) (*Entry, error) {
	if boltTx, err := c.unwrapTx(tx); err != nil {
		return nil, err
	} else {
		return getEntry(boltTx.Bucket([]byte(pathsBucket)), path)
	}
}

func (c BoltCache) UpdatePath(tx Tx, path string, entry *Entry) error {
	if boltTx, err := c.unwrapTx(tx); err != nil {
		return err
	} else {
		return putEntry(boltTx.Bucket([]byte(pathsBucket)), path, entry)
	}
}

func (c BoltCache) RemoveAllPaths(tx Tx) error {
	if boltTx, err := c.unwrapTx(tx); err != nil {
		return err
	} else {
		cursor := boltTx.Bucket([]byte(pathsBucket)).Cursor()
		for k, v := cursor.First(); !(k == nil && v == nil); k, v = cursor.Next() {
			if err := cursor.Delete(); err != nil {
				return fmt.Errorf("failed to remove path entry: %w", err)
			}
		}
		return nil
	}
}

//func (c BoltCache) HaveFormattersChanged(tx Tx, formatters map[string]*format.Formatter) (bool, error) {
//	boltTx, err := c.unwrapTx(tx)
//	if err != nil {
//		return false, err
//	}
//
//	// flag to indicate if we should bust the cache because the formatters have changed
//	changed := false
//
//	// get a reference to the formatters bucket
//	bucket := boltTx.Bucket([]byte(formattersBucket))
//
//	// check for any newly configured or modified formatters
//	for name, formatter := range formatters {
//
//		// check the formatter's executable exists and is executable
//		stat, err := os.Lstat(formatter.Executable())
//		if err != nil {
//			return changed, fmt.Errorf("failed to stat formatter executable %v: %w", formatter.Executable(), err)
//		}
//
//		// retrieve the cache entry for the formatter if one exists
//		entry, err := getEntry(bucket, name)
//		if err != nil {
//			return changed, fmt.Errorf("failed to retrieve cache entry for formatter %v: %w", name, err)
//		}
//
//		// determine if it is new or has changed
//		isNew := entry == nil
//		hasChanged := entry != nil && !(entry.Size == stat.Size() && entry.Modified == stat.ModTime())
//
//		if isNew {
//			c.log.Debugf("formatter '%s' is new", name)
//		} else if hasChanged {
//			c.log.Debug("formatter '%s' has changed",
//				name,
//				"size", stat.Size(),
//				"modTime", stat.ModTime(),
//				"cachedSize", entry.Size,
//				"cachedModTime", entry.Modified,
//			)
//		}
//
//		// update the overall clean flag
//		changed = changed || isNew || hasChanged
//
//		// record formatters info
//		entry = &Entry{
//			Size:     stat.Size(),
//			Modified: stat.ModTime(),
//		}
//
//		if err = putEntry(bucket, name, entry); err != nil {
//			return changed, fmt.Errorf("failed to write cache entry for formatter %v: %w", name, err)
//		}
//	}
//
//	// check for any removed formatters
//	if err := bucket.ForEach(func(key []byte, _ []byte) error {
//		if _, ok := formatters[string(key)]; !ok {
//			// remove the formatter entry from the cache
//			if err := bucket.Delete(key); err != nil {
//				return fmt.Errorf("failed to remove cache entry for formatter %v: %w", key, err)
//			}
//			// indicate a clean is required
//			changed = true
//		}
//		return nil
//	}); err != nil {
//		return changed, fmt.Errorf("failed to check cache for removed formatters: %w", err)
//	}
//
//	return changed, nil
//}

// Close closes any open instance of the cache.
func (c BoltCache) Close() error {
	if c.db == nil {
		return nil
	}
	return c.db.Close()
}

// NewBoltCache creates an instance of bolt.DB for a given treeRoot path.
// If clear is true, Open will delete any existing data in the cache.
//
// The database will be located in `XDG_CACHE_DIR/treefmt/eval-cache/<id>.db`, where <id> is determined by hashing
// the treeRoot path. This associates a given treeRoot with a given instance of the cache.
func NewBoltCache(treeRoot string, clear bool) (Cache, error) {
	// determine a unique and consistent db name for the tree root
	h := sha1.New()
	h.Write([]byte(treeRoot))
	digest := h.Sum(nil)

	name := hex.EncodeToString(digest)
	path, err := xdg.CacheFile(fmt.Sprintf("treefmt/eval-cache/%v.db", name))
	if err != nil {
		return nil, fmt.Errorf("could not resolve local path for the cache: %w", err)
	}

	db, err := bolt.Open(path, 0o600, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open cache at %v: %w", path, err)
	}

	// ensure the buckets exist
	err = db.Update(func(tx *bolt.Tx) error {
		// create bucket for tracking paths
		_, err := tx.CreateBucketIfNotExists([]byte(pathsBucket))
		if err != nil {
			return fmt.Errorf("failed to create paths bucket: %w", err)
		}

		// create bucket for tracking formatters
		_, err = tx.CreateBucketIfNotExists([]byte(formattersBucket))
		if err != nil {
			return fmt.Errorf("failed to create formatters bucket: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialise db: %w", err)
	}

	result := &BoltCache{
		db:  db,
		log: log.WithPrefix("cache"),
	}

	if clear {
		// remove all paths from the cache
		err = db.Update(func(tx *bolt.Tx) error {
			return result.RemoveAllPaths(tx)
		})
		if err != nil {
			return nil, fmt.Errorf("failed to clear paths: %w", err)
		}
	}

	return result, nil
}

// getEntry is a helper for reading cache entries from bolt.
func getEntry(bucket *bolt.Bucket, path string) (*Entry, error) {
	b := bucket.Get([]byte(path))
	if b != nil {
		var cached Entry
		if err := msgpack.Unmarshal(b, &cached); err != nil {
			return nil, fmt.Errorf("failed to unmarshal cache info for path '%v': %w", path, err)
		}
		return &cached, nil
	} else {
		return nil, nil
	}
}

// putEntry is a helper for writing cache entries into bolt.
func putEntry(bucket *bolt.Bucket, path string, entry *Entry) error {
	bytes, err := msgpack.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal cache path %v: %w", path, err)
	}

	if err = bucket.Put([]byte(path), bytes); err != nil {
		return fmt.Errorf("failed to put cache path %v: %w", path, err)
	}
	return nil
}
