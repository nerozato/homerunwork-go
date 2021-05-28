package main

import (
	"net/http"
	"strings"

	"github.com/pkg/errors"
)

//custom HTTP file system that only reads files and excludes directories
type fileOnlyFileSystem struct {
	fs http.FileSystem
}

//Open : open the path
func (fofs fileOnlyFileSystem) Open(path string) (http.File, error) {
	_, logger := GetLogger(nil)

	//open the file
	f, err := fofs.fs.Open(path)
	if err != nil {
		logger.Debugw("open file", "path", path)
		return nil, errors.Wrap(err, "open file")
	}

	//check if what's opened is a directory
	stat, err := f.Stat()
	if stat.IsDir() {
		//by default, return index.html if available
		indexPath := strings.TrimSuffix(path, "/") + "/index.html"
		_, err := fofs.fs.Open(indexPath)
		if err != nil {
			logger.Debugw("open file", "path", path)
			return nil, errors.Wrap(err, "open file")
		}
	}
	return f, nil
}
