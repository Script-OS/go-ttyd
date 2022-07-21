package mergefs

import (
	"errors"
	"io/fs"
	"log"
	"os"
)

var Logger = log.New(os.Stderr, "", log.LstdFlags)

// Merge filesystems
func Merge(filesystems ...fs.FS) fs.FS {
	return MergedFS{filesystems: filesystems, allowErrors: []error{fs.ErrNotExist}}
}

type Options func(m MergedFS) MergedFS

// AllowError provides a way to ignore certain errors
func AllowError(err error) Options {
	return func(m MergedFS) MergedFS {
		m.allowErrors = append(m.allowErrors, err)
		return m
	}
}

func MergeWithOptions(o Options, filesystems ...fs.FS) fs.FS {
	return o(MergedFS{filesystems: filesystems})
}

// MergedFS combines filesystems. Each filesystem can serve different paths. The first FS takes precedence
type MergedFS struct {
	filesystems []fs.FS
	allowErrors []error
}

func (mfs MergedFS) allow(err error) bool {
	for _, allowed := range mfs.allowErrors {
		if errors.Is(err, allowed) {
			return true
		}
	}
	return false
}

// Open opens the named file.
func (filesystem MergedFS) Open(name string) (fs.File, error) {
	for _, mfs := range filesystem.filesystems {
		file, err := mfs.Open(name)
		if err == nil {
			return file, nil
		}
		if !filesystem.allow(err) {
			return nil, err
		}
		/*
			if e, ok := err.(*fs.PathError); ok {
				if !errors.Allow(fs.Err) {
				}
				if !errors.Is(e.Err, fs.ErrNotExist) {
					for _, aerr := range mfs.allowErrors {
						if errors.Is(err, aerr) {
							continue outer
						}
					}
					return nil, err
				}
			}
		*/
	}
	return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
}

// ReadDir reads from the directory, and produces a DirEntry array of different
// directories.
//
// It iterates through all different filesystems that exist in the mfs MergeFS
// filesystem slice and it identifies overlapping directories that exist in different
// filesystems
func (mfs MergedFS) ReadDir(name string) ([]fs.DirEntry, error) {
	dirsMap := make(map[string]fs.DirEntry)
	notExistCount := 0
	for _, filesystem := range mfs.filesystems {
		dir, err := fs.ReadDir(filesystem, name)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				notExistCount++
				Logger.Printf("directory in filepath %s was not found in filesystem", name)
				continue
			}
			return nil, err
		}
		for _, v := range dir {
			if _, ok := dirsMap[v.Name()]; !ok {
				dirsMap[v.Name()] = v
			}
		}
		continue
	}
	if len(mfs.filesystems) == notExistCount {
		return nil, fs.ErrNotExist
	}
	dirs := make([]fs.DirEntry, 0, len(dirsMap))

	for _, value := range dirsMap {
		dirs = append(dirs, value)
	}

	return dirs, nil
}
