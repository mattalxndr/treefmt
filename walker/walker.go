package walker

import (
	"context"
	"fmt"
	"git.numtide.com/numtide/treefmt/caching"
	"github.com/charmbracelet/log"
	"io/fs"
	"os"
	"time"
)

type Type string

const (
	Git        Type = "git"
	Auto       Type = "auto"
	Filesystem Type = "filesystem"
)

type File struct {
	Path    string
	RelPath string
	Info    fs.FileInfo
}

func (f File) HasChanged() (bool, fs.FileInfo, error) {
	// get the file's current state
	current, err := os.Stat(f.Path)
	if err != nil {
		return false, nil, fmt.Errorf("failed to stat %s: %w", f.Path, err)
	}

	// check the size first
	if f.Info.Size() != current.Size() {
		return true, current, nil
	}

	// POSIX specifies EPOCH time for Mod time, but some filesystems give more precision.
	// Some formatters mess with the mod time (e.g., dos2unix) but not to the same precision,
	// triggering false positives.
	// We truncate everything below a second.
	if f.Info.ModTime().Truncate(time.Second) != current.ModTime().Truncate(time.Second) {
		return true, current, nil
	}

	return false, nil, nil
}

func (f File) String() string {
	return f.Path
}

type WalkFunc func(file *File, err error) error

type Walker interface {
	Root() string
	Walk(ctx context.Context, fn WalkFunc) error
	UpdatePaths(batch []*File) error
	Close() error
}

func New(
	walkerType Type,
	root string,
	cached bool,
	clearCache bool,
	pathsCh chan string,
) (Walker, error) {

	// open the cache if configured
	var err error
	var cache caching.Cache = caching.NoOpCache{}
	if cached {
		cache, err = caching.NewBoltCache(root, clearCache)
		if err != nil {
			// if we can't open the cache, we log a warning and fallback to no cache
			log.Warnf("failed to open cache: %v", err)
			cache = caching.NoOpCache{}
		}

		// TODO formatter changes
		//err = cache.Update(func(tx caching.Tx) error {
		//	changed, err := cache.HaveFormattersChanged(tx, formatters)
		//	if err != nil {
		//		return err
		//	}
		//	if changed {
		//		// bust the paths cache
		//		return cache.RemoveAllPaths(tx)
		//	}
		//	return nil
		//})
		//if err != nil {
		//	return nil, fmt.Errorf("failed to check if formatters have changed: %w", err)
		//}
	}

	switch walkerType {
	case Git:
		return NewGit(root, cache, pathsCh)
	case Auto:
		return Detect(root, cache, pathsCh)
	case Filesystem:
		return NewFilesystem(root, cache, pathsCh)
	default:
		return nil, fmt.Errorf("unknown walker type: %v", walkerType)
	}
}

func Detect(root string, cache caching.Cache, pathsCh chan string) (Walker, error) {
	// for now, we keep it simple and try git first, filesystem second
	w, err := NewGit(root, cache, pathsCh)
	if err == nil {
		return w, err
	}
	return NewFilesystem(root, cache, pathsCh)
}
