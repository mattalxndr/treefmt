package caching

import (
	"runtime"
	"time"
)

const (
	pathsBucket      = "paths"
	formattersBucket = "formatters"
)

// Entry represents a cache entry, indicating the last size and modified time for a file path.
type Entry struct {
	Size     int64
	Modified time.Time
}

var (
	ReadBatchSize = 1024 * runtime.NumCPU()
)

type Tx interface {
	Commit() error
	Rollback() error
}

type Cache interface {
	View(func(tx Tx) error) error
	Update(func(tx Tx) error) error

	BeginRead() (Tx, error)

	GetPath(tx Tx, path string) (*Entry, error)
	UpdatePath(tx Tx, path string, entry *Entry) error
	RemoveAllPaths(tx Tx) error

	//HaveFormattersChanged(tx Tx, formatters map[string]*format.Formatter) (bool, error)
	Close() error
}

//
//// ChangeSet is used to walk a filesystem, starting at root, and outputting any new or changed paths using pathsCh.
//// It determines if a path is new or has changed by comparing against cache entries.
//func ChangeSet(ctx context.Context, wk walker.Walker, filesCh chan<- *walker.File) error {
//	start := time.Now()
//
//	defer func() {
//		c.log.Debugf("finished generating change set in %v", time.Since(start))
//	}()
//
//	var tx *bolt.Tx
//	var bucket *bolt.Bucket
//	var processed int
//
//	defer func() {
//		// close any pending read tx
//		if tx != nil {
//			_ = tx.Rollback()
//		}
//	}()
//
//	return wk.Walk(ctx, func(file *walker.File, err error) error {
//		select {
//		case <-ctx.Done():
//			return ctx.Err()
//		default:
//			if err != nil {
//				return fmt.Errorf("failed to walk path: %w", err)
//			} else if file.Info.IsDir() {
//				// ignore directories
//				return nil
//			}
//		}
//
//		// open a new read tx if there isn't one in progress
//		// we have to periodically open a new read tx to prevent writes from being blocked
//		if tx == nil {
//			tx, err = db.Begin(false)
//			if err != nil {
//				return fmt.Errorf("failed to open a new cache read tx: %w", err)
//			}
//			bucket = tx.Bucket([]byte(pathsBucket))
//		}
//
//		cached, err := getEntry(bucket, file.RelPath)
//		if err != nil {
//			return err
//		}
//
//		changedOrNew := cached == nil || !(cached.Modified == file.Info.ModTime() && cached.Size == file.Info.Size())
//
//		stats.Add(stats.Traversed, 1)
//		if !changedOrNew {
//			// no change
//			return nil
//		}
//
//		stats.Add(stats.Emitted, 1)
//
//		// pass on the path
//		select {
//		case <-ctx.Done():
//			return ctx.Err()
//		default:
//			filesCh <- file
//		}
//
//		// close the current tx if we have reached the batch size
//		processed += 1
//		if processed == ReadBatchSize {
//			err = tx.Rollback()
//			tx = nil
//			return err
//		}
//
//		return nil
//	})
//}
//
//// Update is used to record updated cache information for the specified list of paths.
//func Update(files []*walker.File) error {
//	start := time.Now()
//	defer func() {
//		logger.Debugf("finished processing %v paths in %v", len(files), time.Since(start))
//	}()
//
//	if len(files) == 0 {
//		return nil
//	}
//
//	return db.Update(func(tx *bolt.Tx) error {
//		bucket := tx.Bucket([]byte(pathsBucket))
//
//		for _, f := range files {
//			entry := Entry{
//				Size:     f.Info.Size(),
//				Modified: f.Info.ModTime(),
//			}
//
//			if err := putEntry(bucket, f.RelPath, &entry); err != nil {
//				return err
//			}
//		}
//
//		return nil
//	})
//}
