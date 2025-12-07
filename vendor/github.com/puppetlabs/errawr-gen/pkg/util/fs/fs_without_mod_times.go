package fs

import (
	"net/http"
	"os"
	"time"
)

type fileInfoWithoutModTime struct {
	os.FileInfo
}

func (fiwmt fileInfoWithoutModTime) ModTime() time.Time {
	// Nobody cares about file modification time and I wish it would stop being
	// a thing. make be damned.
	return time.Time{}
}

type fileWithoutModTime struct {
	http.File
}

func (fwmt fileWithoutModTime) Readdir(count int) ([]os.FileInfo, error) {
	fis, err := fwmt.File.Readdir(count)
	if err != nil {
		return nil, err
	}

	for i, fi := range fis {
		fis[i] = fileInfoWithoutModTime{fi}
	}

	return fis, nil
}

func (fwmt fileWithoutModTime) Stat() (os.FileInfo, error) {
	fi, err := fwmt.File.Stat()
	if err != nil {
		return nil, err
	}

	return fileInfoWithoutModTime{fi}, nil
}

type fileSystemWithoutModTimes struct {
	http.FileSystem
}

func (fswmt fileSystemWithoutModTimes) Open(name string) (http.File, error) {
	f, err := fswmt.FileSystem.Open(name)
	if err != nil {
		return nil, err
	}

	return &fileWithoutModTime{f}, nil
}

func FileSystemWithoutModTimes(delegate http.FileSystem) http.FileSystem {
	return fileSystemWithoutModTimes{delegate}
}

func DirWithoutModTimes(path string) http.FileSystem {
	return FileSystemWithoutModTimes(http.Dir(path))
}
